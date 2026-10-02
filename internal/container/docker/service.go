package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/histogram"
	"github.com/amir20/dozzle/internal/container/logparse"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/internal/selfupdate"

	"github.com/moby/moby/api/pkg/stdcopy"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/rs/zerolog/log"
)

// UpdateClient extends container.Client with Docker-specific update operations.
type UpdateClient interface {
	container.Client
	ImagePull(ctx context.Context, image string) (io.ReadCloser, error)
	ImageRepoDigests(ctx context.Context, imageID string) ([]string, error)
	ImageID(ctx context.Context, ref string) (string, error)
	ContainerInspect(ctx context.Context, containerID string) (docker_types.InspectResponse, error)
	ContainerRemove(ctx context.Context, containerID string) error
	ContainerCreate(ctx context.Context, inspectResp docker_types.InspectResponse, name string) (string, error)
	NetworkDependents(ctx context.Context, id string, name string) ([]string, error)
	ServiceUpdate(ctx context.Context, serviceID string, image string) error
	ContainerLogsTail(ctx context.Context, id string, lines int) (io.ReadCloser, error)
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
	if strings.Contains(selfupdate.ImageRef(inspect.Config), "amir20/dozzle") {
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

type pullEvent struct {
	Status         string `json:"status"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	ID string `json:"id"`
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
	return d.checker.Check(ctx, selfupdate.ImageRef(inspect.Config), digests, force), nil
}

func (d *Service) UpdateContainer(ctx context.Context, c container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	defer close(progressCh)

	// The consumer is a request: an SSE handler that returns the moment a write
	// to the client fails, or an agent stream that ends with its RPC. An
	// unguarded send outlives it and parks this goroutine mid-update forever.
	progress := func(p container.UpdateProgress) {
		select {
		case progressCh <- p:
		case <-ctx.Done():
		}
	}

	// 1. Inspect container to get full config
	inspectResp, err := d.client.ContainerInspect(ctx, c.ID)
	if err != nil {
		progress(container.UpdateProgress{Status: "error", Error: fmt.Sprintf("inspect failed: %v", err)})
		return false, err
	}

	imageName := selfupdate.ImageRef(inspectResp.Config)

	// 2. Pull image with progress
	reader, err := d.client.ImagePull(ctx, imageName)
	if err != nil {
		progress(container.UpdateProgress{Status: "error", Error: fmt.Sprintf("pull failed: %v", err)})
		return false, err
	}
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	for {
		var event pullEvent
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			progress(container.UpdateProgress{Status: "error", Error: fmt.Sprintf("pull decode failed: %v", err)})
			return false, err
		}

		progress(container.UpdateProgress{
			Status:  "pulling",
			Layer:   event.ID,
			Current: event.ProgressDetail.Current,
			Total:   event.ProgressDetail.Total,
		})
	}

	// 3. Compare what the tag resolves to now against what the container is
	// actually running. Reading this from the pull output instead would miss
	// the case where the newer image is already in the local store, which
	// happens whenever it was pulled or built before the container was
	// recreated.
	updated := false
	if newImageID, err := d.client.ImageID(ctx, imageName); err != nil {
		log.Warn().Err(err).Str("image", imageName).Msg("unable to resolve pulled image, falling back to recreate")
		updated = true
	} else {
		updated = newImageID != inspectResp.Image
	}

	if !updated {
		progress(container.UpdateProgress{Status: "up-to-date"})
		return false, nil
	}

	// 4. Check if this is a swarm service
	serviceName := c.Labels["com.docker.swarm.service.name"]
	if serviceName != "" {
		progress(container.UpdateProgress{Status: "recreating"})
		serviceID := c.Labels["com.docker.swarm.service.id"]
		if err := d.client.ServiceUpdate(ctx, serviceID, imageName); err != nil {
			progress(container.UpdateProgress{Status: "error", Error: fmt.Sprintf("service update failed: %v", err)})
			return false, err
		}
		progress(container.UpdateProgress{Status: "done"})
		return true, nil
	}

	// 5. Standalone container: stopping Dozzle's own container would kill this
	// process mid-update, so a helper container on the new image does the swap.
	if isSelf(c.ID) {
		return startSelfUpdate(ctx, inspectResp.ID, progress)
	}
	if mayBeSelf(inspectResp) {
		progress(container.UpdateProgress{Status: "error", Error: "Dozzle cannot identify its own container, so it cannot update it. Please update it manually."})
		return false, fmt.Errorf("cannot self-update: own container id is unknown")
	}

	// 6. Standalone container: stop -> remove -> create -> start
	progress(container.UpdateProgress{Status: "recreating"})

	containerName := strings.TrimPrefix(inspectResp.Name, "/")

	// Containers joined to this one's network namespace (network_mode:
	// service:x in compose) lose it with the old container, so they are
	// recreated after it.
	dependents, err := d.client.NetworkDependents(ctx, inspectResp.ID, containerName)
	if err != nil {
		progress(container.UpdateProgress{Status: "error", Error: fmt.Sprintf("list dependents failed: %v", err)})
		return false, err
	}

	// From here on the old container is going away, so a client that
	// disconnects must not leave the swap half done. That includes Dozzle
	// itself when it shares this container's network.
	ctx = context.WithoutCancel(ctx)

	// Stop if running
	if c.State == "running" {
		if err := d.client.ContainerActions(ctx, container.Stop, c.ID); err != nil {
			progress(container.UpdateProgress{Status: "error", Error: fmt.Sprintf("stop failed: %v", err)})
			return false, err
		}
	}

	// Remove
	if err := d.client.ContainerRemove(ctx, c.ID); err != nil {
		progress(container.UpdateProgress{Status: "error", Error: fmt.Sprintf("remove failed: %v", err)})
		return false, err
	}

	// Create with same config
	newID, err := d.client.ContainerCreate(ctx, inspectResp, containerName)
	if err != nil {
		progress(container.UpdateProgress{Status: "error", Error: fmt.Sprintf("create failed: %v", err)})
		return false, err
	}

	// Start
	if err := d.client.ContainerActions(ctx, container.Start, newID); err != nil {
		progress(container.UpdateProgress{Status: "error", Error: fmt.Sprintf("start failed: %v", err)})
		return false, err
	}

	if err := d.rejoinDependents(ctx, dependents, inspectResp.ID, newID); err != nil {
		progress(container.UpdateProgress{Status: "error", Error: err.Error()})
		return true, err
	}

	progress(container.UpdateProgress{Status: "done"})
	return true, nil
}

// rejoinDependents recreates every container in ids, which shared the network
// namespace of oldID, joined to newID instead. Each keeps its own image; one
// that was stopped is recreated but left stopped. Dozzle's own container is
// handed to the self-update helper, since recreating it here would stop this
// process halfway through.
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
	return d.client.Host(), nil
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
