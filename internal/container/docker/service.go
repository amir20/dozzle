package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/docker/swap"
	"github.com/amir20/dozzle/internal/container/histogram"
	"github.com/amir20/dozzle/internal/container/logparse"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/internal/selfupdate"

	"github.com/moby/moby/api/pkg/stdcopy"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/rs/zerolog/log"
)

// UpdateClient extends container.Client with Docker-specific update operations.
type UpdateClient interface {
	container.Client
	ImagePull(ctx context.Context, image string) (io.ReadCloser, error)
	ImageRepoDigests(ctx context.Context, imageID string) ([]string, error)
	ImageID(ctx context.Context, ref string) (string, error)
	ImageInspect(ctx context.Context, ref string) (image.InspectResponse, error)
	ContainerInspect(ctx context.Context, containerID string) (docker_types.InspectResponse, error)
	ContainerRemove(ctx context.Context, containerID string) error
	ContainerCreate(ctx context.Context, inspectResp docker_types.InspectResponse, name string) (string, error)
	NetworkDependents(ctx context.Context, id string, name string) ([]string, error)
	ServiceUpdate(ctx context.Context, serviceID string, image string) error
	ContainerLogsTail(ctx context.Context, id string, lines int) (io.ReadCloser, error)
	// SwapAPI is the engine client UpdateContainer swaps containers, and
	// cleans up images, with.
	SwapAPI() swap.API
}

var (
	selfContainerID = selfupdate.SelfID
	startSelfUpdate = selfupdate.Start
	startRejoin     = selfupdate.StartRejoin
	hostname        = os.Hostname
)

// isSelf reports whether id (Dozzle's 12-character form or a full id) is the
// container this process runs in.
func isSelf(id string) bool {
	self := selfContainerID()
	return self != "" && len(id) >= 12 && strings.HasPrefix(self, id)
}

// mayBeSelf reports whether a container could be Dozzle's own when that id is
// unknown (Podman's mountinfo never names it, for one). Recreating it in place
// would stop this process mid-update, so anything on a Dozzle image, or whose
// short id is this process's hostname, is refused.
func mayBeSelf(inspect docker_types.InspectResponse) bool {
	if selfContainerID() != "" || inspect.Config == nil {
		return false
	}
	if strings.Contains(swap.ImageRef(inspect.Config), "amir20/dozzle") {
		return true
	}
	h, err := hostname()
	return err == nil && len(h) >= 12 && strings.HasPrefix(inspect.ID, h)
}

type Service struct {
	client     UpdateClient
	store      *container.Store
	checker    *imagecheck.Checker
	histograms *histogram.Counter
}

func NewService(client UpdateClient, labels container.ContainerLabels) *Service {
	statsCollector := NewStatsCollector(client, labels)
	return &Service{
		client:     client,
		store:      container.NewStore(context.Background(), client, statsCollector, labels),
		checker:    imagecheck.Shared(),
		histograms: histogram.NewCounter(),
	}
}

// Client returns the underlying docker client.
func (d *Service) Client() UpdateClient {
	return d.client
}

func (d *Service) RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
	reader, err := d.client.ContainerLogsBetweenDates(ctx, container.ID, from, to, stdTypes)
	if err != nil {
		return nil, err
	}

	in, out := io.Pipe()

	go func() {
		if container.Tty {
			if _, err := io.Copy(out, reader); err != nil {
				log.Error().Err(err).Msgf("error copying logs for container %s", container.ID)
			}
		} else {
			if _, err := stdcopy.StdCopy(out, out, reader); err != nil {
				log.Error().Err(err).Msgf("error copying logs for container %s", container.ID)
			}
		}

		out.Close()
	}()

	return in, nil

}

func (d *Service) LogsBetweenDates(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	reader, err := d.client.ContainerLogsBetweenDates(ctx, c.ID, from, to, stdTypes)
	if err != nil {
		return nil, err
	}

	dockerReader := NewLogReader(reader, c.Tty)
	g := logparse.NewEventGenerator(ctx, dockerReader, c)
	return g.Events, nil
}

func (d *Service) LogHistogram(ctx context.Context, c container.Container, from time.Time, to time.Time, width time.Duration) (container.LogHistogram, error) {
	return d.histograms.Count(ctx, c.ID, from, to, width, func(ctx context.Context, lines int) (histogram.LineReader, io.Closer, error) {
		reader, err := d.client.ContainerLogsTail(ctx, c.ID, lines)
		if err != nil {
			return nil, nil, err
		}
		return skipBadHeaders{NewLogReader(reader, c.Tty)}, reader, nil
	})
}

// skipBadHeaders reads past a frame whose header is malformed, as the event
// generator does, instead of ending the count there.
type skipBadHeaders struct{ *LogReader }

func (r skipBadHeaders) Read() (string, container.StdType, error) {
	for {
		line, std, err := r.LogReader.Read()
		if err == ErrBadHeader {
			continue
		}
		return line, std, err
	}
}

func (d *Service) StreamLogs(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	reader, err := d.client.ContainerLogs(ctx, c.ID, from, stdTypes)
	if err != nil {
		return err
	}

	dockerReader := NewLogReader(reader, c.Tty)
	g := logparse.NewEventGenerator(ctx, dockerReader, c)
	for event := range g.Events {
		select {
		case events <- event:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	select {
	case e := <-g.Errors:
		return e
	default:
		return nil
	}
}

func (d *Service) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	return d.store.FindContainer(ctx, id, labels)
}

func (d *Service) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
	return d.client.ContainerActions(ctx, action, container.ID)
}

// CheckImageUpdate reports whether the registry serves a newer image than the
// one this container is running.
func (d *Service) CheckImageUpdate(ctx context.Context, c container.Container, force bool) (imagecheck.Result, error) {
	if imagecheck.Skipped(c.Labels) {
		log.Debug().Str("container", c.Name).Msg("image update check: skipped by label")
		return imagecheck.Result{Image: c.Image, Status: imagecheck.StatusSkipped, CheckedAt: time.Now()}, nil
	}

	inspect, err := d.client.ContainerInspect(ctx, c.ID)
	if err != nil {
		log.Debug().Err(err).Str("container", c.Name).Msg("image update check: inspect failed")
		return imagecheck.Result{}, err
	}

	// Inspect by image ID rather than by name. If a newer image was pulled
	// under the same tag but the container was never recreated, the container
	// is still running the old image and should report an available update.
	digests, err := d.client.ImageRepoDigests(ctx, inspect.Image)
	if err != nil {
		log.Debug().Err(err).Str("container", c.Name).Str("imageId", inspect.Image).Msg("image update check: image inspect failed")
		return imagecheck.Result{}, err
	}

	log.Debug().
		Str("container", c.Name).
		Str("ref", inspect.Config.Image).
		Str("imageId", inspect.Image).
		Msg("image update check: resolved local image")

	// Config.Image is the reference the container was created from, which is
	// what the registry must be queried for.
	// A rolled-back Dozzle runs from a bare image id and keeps its tag in a label.
	return d.checker.Check(ctx, swap.ImageRef(inspect.Config), digests, force), nil
}

func (d *Service) UpdateContainer(ctx context.Context, c container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	defer close(progressCh)

	// Every send is unguarded, so the final done or rolled-back, which carries
	// the Result the update is recorded from, is never dropped because the
	// request ended. Every consumer drains progressCh until it is closed:
	// ContainerService.recorded, which records the Result and forwards to the
	// request only while it lasts, and the agent server, which keeps reading
	// after its stream fails.
	progress := func(p container.UpdateProgress) { progressCh <- p }
	fail := func(err error) (bool, error) {
		progress(container.UpdateProgress{Status: container.UpdateError, Error: err.Error()})
		return false, err
	}

	// 1. Inspect container to get full config
	inspectResp, err := d.client.ContainerInspect(ctx, c.ID)
	if err != nil {
		return fail(fmt.Errorf("inspect failed: %w", err))
	}
	if inspectResp.Config == nil {
		return fail(fmt.Errorf("inspect failed: container has no config"))
	}
	// Checked before the pull: there is nothing to pull for a container that
	// will not be swapped.
	if !swap.Running(inspectResp.State) {
		return fail(container.ErrNotRunning)
	}

	imageName := swap.ImageRef(inspectResp.Config)

	// 2. Pull image with progress
	reader, err := d.client.ImagePull(ctx, imageName)
	if err != nil {
		return fail(fmt.Errorf("pull failed: %w", err))
	}
	defer reader.Close()

	if err := swap.ReadPull(reader, progress); err != nil {
		return fail(err)
	}

	// 3. Compare what the tag resolves to now against what the container is
	// actually running. Reading this from the pull output instead would miss
	// the case where the newer image is already in the local store, which
	// happens whenever it was pulled or built before the container was
	// recreated.
	updated := false
	newImageID, err := d.client.ImageID(ctx, imageName)
	if err != nil {
		log.Warn().Err(err).Str("image", imageName).Msg("unable to resolve pulled image, falling back to recreate")
		updated = true
	} else {
		updated = newImageID != inspectResp.Image
	}

	if !updated {
		progress(container.UpdateProgress{Status: container.UpdateUpToDate})
		return false, nil
	}

	// 4. Check if this is a swarm service
	serviceName := c.Labels[container.SwarmServiceNameLabel]
	if serviceName != "" {
		progress(container.UpdateProgress{Status: container.UpdateRecreating})
		serviceID := c.Labels[container.SwarmServiceIDLabel]
		if err := d.client.ServiceUpdate(ctx, serviceID, imageName); err != nil {
			return fail(fmt.Errorf("service update failed: %w", err))
		}
		progress(container.UpdateProgress{Status: container.UpdateDone})
		return true, nil
	}

	// 5. Standalone container: stopping Dozzle's own container would kill this
	// process mid-update, so a helper container on the new image does the swap.
	if isSelf(c.ID) {
		return startSelfUpdate(ctx, inspectResp.ID, progress)
	}
	if mayBeSelf(inspectResp) {
		progress(container.UpdateProgress{Status: container.UpdateError, Error: "Dozzle cannot identify its own container, so it cannot update it. Please update it manually."})
		return false, fmt.Errorf("cannot self-update: own container id is unknown")
	}

	// 6. Standalone container: swap it for one on the new image, keeping the
	// old one until the new one has stayed up.

	// The old image's own settings are dropped from the replacement so the
	// new image's defaults apply. It is still in the local store: nothing has
	// removed it yet, and the old container still uses it. Not cancellable,
	// so a client leaving now cannot make the replacement keep them.
	var oldImage *image.InspectResponse
	if img, err := d.client.ImageInspect(context.WithoutCancel(ctx), inspectResp.Image); err == nil {
		oldImage = &img
	} else {
		log.Warn().Err(err).Str("image", inspectResp.Image).Msg("could not inspect the old image, keeping its settings on the replacement")
	}

	result, rejoinErr, err := d.swapAndRejoin(ctx, inspectResp, swap.Options{
		OldImage: oldImage,
		Labels:   swap.PreviousLabels(inspectResp, oldImage, imageName),
	}, progress)
	ctx = context.WithoutCancel(ctx)
	if err != nil {
		if result.RolledBack {
			undone := d.updateResult(ctx, inspectResp, oldImage, imageName, result.RestoredID, newImageID)
			undone.RolledBack = true
			progress(container.UpdateProgress{Status: container.UpdateRolledBack, Error: err.Error(), Result: &undone})
			return false, fmt.Errorf("update rolled back: %w", err)
		}
		return fail(err)
	}
	done := d.updateResult(ctx, inspectResp, oldImage, imageName, result.NewID, newImageID)

	if rejoinErr != nil {
		progress(container.UpdateProgress{Status: container.UpdateError, Error: rejoinErr.Error(), Result: &done})
		return true, rejoinErr
	}

	swap.CleanupImage(ctx, d.client.SwapAPI(), inspectResp)

	progress(container.UpdateProgress{Status: container.UpdateDone, Result: &done})
	return true, nil
}

// swapAndRejoin reports recreating, swaps old as opts describe, and moves the
// containers joined to old's network namespace (network_mode: service:x in
// compose), which lose it with the old container, onto whichever container
// holds it afterwards. From the swap on, ctx being cancelled is ignored: a
// client that disconnects must not leave the swap half done, and that
// includes Dozzle itself when it shares old's network.
//
// err is the swap's (with a failed rejoin after a rollback added to it), or a
// failure to list the dependents before anything was touched. rejoinErr is a
// failure to move them after a swap that committed.
func (d *Service) swapAndRejoin(ctx context.Context, old docker_types.InspectResponse, opts swap.Options, progress func(container.UpdateProgress)) (result swap.Result, rejoinErr error, err error) {
	progress(container.UpdateProgress{Status: container.UpdateRecreating})

	dependents, err := d.client.NetworkDependents(ctx, old.ID, strings.TrimPrefix(old.Name, "/"))
	if err != nil {
		return swap.Result{}, nil, fmt.Errorf("list dependents failed: %w", err)
	}

	ctx = context.WithoutCancel(ctx)
	opts.OnVerifying = func() { progress(container.UpdateProgress{Status: container.UpdateVerifying}) }
	result, err = swap.Swap(ctx, d.client.SwapAPI(), old, opts)
	if err != nil {
		// A stopped old container took its namespace with it, so dependents
		// rejoin whichever container now holds the name.
		if result.OldStopped && result.RolledBack {
			if rejoinErr := d.rejoinDependents(ctx, dependents, old.ID, result.RestoredID); rejoinErr != nil {
				err = fmt.Errorf("%w; %v", err, rejoinErr)
			}
		}
		return result, nil, err
	}
	return result, d.rejoinDependents(ctx, dependents, old.ID, result.NewID), nil
}

// updateResult is what a swap of old (running oldImage, nil when it could not
// be inspected, and following ref) changed: newID runs now, on toImageID.
func (d *Service) updateResult(ctx context.Context, old docker_types.InspectResponse, oldImage *image.InspectResponse, ref, newID, toImageID string) container.UpdateResult {
	r := container.UpdateResult{
		OldID:       shortContainerID(old.ID),
		NewID:       shortContainerID(newID),
		FromImageID: old.Image,
		ToImageID:   toImageID,
		FromDigest:  swap.PreviousRef(oldImage, ref),
	}
	if old.State != nil {
		if startedAt, err := time.Parse(time.RFC3339Nano, old.State.StartedAt); err == nil {
			r.OldStartedAt = startedAt.UTC()
		}
	}
	if toImageID != "" {
		if img, err := d.client.ImageInspect(ctx, toImageID); err == nil {
			r.ToDigest = swap.PreviousRef(&img, ref)
		}
	}
	return r
}

// shortContainerID is the 12-character id the store keys containers by.
func shortContainerID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// rejoinDependents recreates every container in ids, which shared the network
// namespace of oldID, joined to newID instead. Each keeps its own image; one
// that was stopped is recreated but left stopped. newID is running: only a
// running container is swapped. Dozzle's own container is handed to the
// self-update helper, since recreating it here would stop this process halfway
// through.
func (d *Service) rejoinDependents(ctx context.Context, ids []string, oldID, newID string) error {
	var errs []error
	// The helper stops this process within seconds, so Dozzle goes last:
	// stopped between a remove and a create, a dependent would be lost.
	var self *docker_types.InspectResponse
	var selfMode string
	for _, id := range ids {
		inspect, err := d.client.ContainerInspect(ctx, id)
		if err != nil {
			errs = append(errs, fmt.Errorf("inspect %s failed: %w", id[:min(12, len(id))], err))
			continue
		}
		name := strings.TrimPrefix(inspect.Name, "/")

		mode := string(inspect.HostConfig.NetworkMode)
		if ref := strings.TrimPrefix(mode, "container:"); strings.HasPrefix(oldID, ref) {
			// A reference by id is dead; one by name already finds the
			// replacement.
			mode = "container:" + newID
		}

		if isSelf(inspect.ID) {
			self, selfMode = &inspect, mode
			continue
		}
		if mayBeSelf(inspect) {
			errs = append(errs, fmt.Errorf("%s shares this container's network and may be Dozzle itself, recreate it manually", name))
			continue
		}

		log.Info().Str("container", name).Str("networkMode", mode).Msg("recreating container that shares the updated container's network")
		wasRunning := inspect.State != nil && inspect.State.Running
		if wasRunning {
			if err := d.client.ContainerActions(ctx, container.Stop, inspect.ID); err != nil {
				errs = append(errs, fmt.Errorf("stop %s failed: %w", name, err))
				continue
			}
		}
		if err := d.client.ContainerRemove(ctx, inspect.ID); err != nil {
			errs = append(errs, fmt.Errorf("remove %s failed: %w", name, err))
			continue
		}
		hc := *inspect.HostConfig
		hc.NetworkMode = docker_types.NetworkMode(mode)
		inspect.HostConfig = &hc
		depID, err := d.client.ContainerCreate(ctx, inspect, name)
		if err != nil {
			errs = append(errs, fmt.Errorf("create %s failed: %w", name, err))
			continue
		}
		if wasRunning {
			if err := d.client.ContainerActions(ctx, container.Start, depID); err != nil {
				errs = append(errs, fmt.Errorf("start %s failed: %w", name, err))
			}
		}
	}
	if self != nil {
		if err := startRejoin(ctx, self.ID, selfMode); err != nil {
			errs = append(errs, fmt.Errorf("rejoin %s failed: %w", strings.TrimPrefix(self.Name, "/"), err))
		}
	}
	return errors.Join(errs...)
}

func (d *Service) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	return d.store.ListContainers(ctx, labels)
}

func (d *Service) Host(ctx context.Context) (container.Host, error) {
	host := d.client.Host()
	host.Reclaimable = d.store.Reclaimable()
	return host, nil
}

func (d *Service) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	d.store.SubscribeStats(ctx, stats)
}

func (d *Service) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	d.store.SubscribeEvents(ctx, events)
}

func (d *Service) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	d.store.SubscribeNewContainers(ctx, containers)
}

func (d *Service) Attach(ctx context.Context, c container.Container, events container.ExecEventReader, stdout io.Writer) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	session, err := d.client.ContainerAttach(cancelCtx, c.ID)
	if err != nil {
		cancel()
		return err
	}

	var wg sync.WaitGroup

	wg.Go(func() {
	loop:
		for {
			event, err := events.ReadEvent()
			if err != nil {
				if err != io.EOF {
					log.Error().Err(err).Msg("error while reading event")
				}
				break
			}

			switch event.Type {
			case "userinput":
				if _, err := session.Writer.Write([]byte(event.Data)); err != nil {
					log.Error().Err(err).Msg("error while writing to container")
					break loop
				}
			case "resize":
				if err := session.Resize(event.Width, event.Height); err != nil {
					log.Error().Err(err).Msg("error while resizing terminal")
				}
			default:
				log.Warn().Str("type", event.Type).Msg("unknown event type")
			}
		}
		cancel()
		session.Writer.Close()
	})

	wg.Go(func() {
		if c.Tty {
			if _, err := io.Copy(stdout, session.Reader); err != nil {
				log.Error().Err(err).Msg("error while writing to ws")
			}
		} else {
			if _, err := stdcopy.StdCopy(stdout, stdout, session.Reader); err != nil {
				log.Error().Err(err).Msg("error while writing to ws")
			}
		}
		cancel()
	})

	wg.Wait()

	return nil
}

func (d *Service) Exec(ctx context.Context, c container.Container, cmd []string, events container.ExecEventReader, stdout io.Writer) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	session, err := d.client.ContainerExec(cancelCtx, c.ID, cmd)
	if err != nil {
		cancel()
		return err
	}

	var wg sync.WaitGroup

	wg.Go(func() {
	loop:
		for {
			event, err := events.ReadEvent()
			if err != nil {
				if err != io.EOF {
					log.Error().Err(err).Msg("error while reading event")
				}
				break
			}

			switch event.Type {
			case "userinput":
				if _, err := session.Writer.Write([]byte(event.Data)); err != nil {
					log.Error().Err(err).Msg("error while writing to container")
					break loop
				}
			case "resize":
				if err := session.Resize(event.Width, event.Height); err != nil {
					log.Error().Err(err).Msg("error while resizing terminal")
				}
			default:
				log.Warn().Str("type", event.Type).Msg("unknown event type")
			}
		}
		cancel()
		session.Writer.Close()
	})

	wg.Go(func() {
		// TTY mode outputs raw bytes without Docker's multiplexing headers.
		if _, err := io.Copy(stdout, session.Reader); err != nil {
			log.Error().Err(err).Msg("error while writing to ws")
		}
		cancel()
	})

	wg.Wait()

	return nil
}
