package docker_support

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/docker"

	"github.com/moby/moby/api/pkg/stdcopy"
	docker_types "github.com/moby/moby/api/types/container"
	"github.com/rs/zerolog/log"
)

// DockerUpdateClient extends container.Client with Docker-specific update operations.
type DockerUpdateClient interface {
	container.Client
	ImagePull(ctx context.Context, image string) (io.ReadCloser, error)
	ContainerInspect(ctx context.Context, containerID string) (docker_types.InspectResponse, error)
	ContainerRemove(ctx context.Context, containerID string) error
	ContainerCreate(ctx context.Context, inspectResp docker_types.InspectResponse, name string) (string, error)
	ServiceUpdate(ctx context.Context, serviceID string, image string) error
}

type DockerClientService struct {
	client DockerUpdateClient
	store  *container.ContainerStore
}

func NewDockerClientService(client DockerUpdateClient, labels container.ContainerLabels) *DockerClientService {
	statsCollector := docker.NewDockerStatsCollector(client, labels)
	return &DockerClientService{
		client: client,
		store:  container.NewContainerStore(context.Background(), client, statsCollector, labels),
	}
}

func (d *DockerClientService) RawLogs(ctx context.Context, container container.Container, from time.Time, to time.Time, stdTypes container.StdType) (io.ReadCloser, error) {
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

func (d *DockerClientService) LogsBetweenDates(ctx context.Context, c container.Container, from time.Time, to time.Time, stdTypes container.StdType) (<-chan *container.LogEvent, error) {
	reader, err := d.client.ContainerLogsBetweenDates(ctx, c.ID, from, to, stdTypes)
	if err != nil {
		return nil, err
	}

	dockerReader := docker.NewLogReader(reader, c.Tty)
	g := container.NewEventGenerator(ctx, dockerReader, c)
	return g.Events, nil
}

func (d *DockerClientService) StreamLogs(ctx context.Context, c container.Container, from time.Time, stdTypes container.StdType, events chan<- *container.LogEvent) error {
	reader, err := d.client.ContainerLogs(ctx, c.ID, from, stdTypes)
	if err != nil {
		return err
	}

	dockerReader := docker.NewLogReader(reader, c.Tty)
	g := container.NewEventGenerator(ctx, dockerReader, c)
	for event := range g.Events {
		events <- event
	}

	select {
	case e := <-g.Errors:
		return e
	default:
		return nil
	}
}

func (d *DockerClientService) FindContainer(ctx context.Context, id string, labels container.ContainerLabels) (container.Container, error) {
	return d.store.FindContainer(id, labels)
}

func (d *DockerClientService) ContainerAction(ctx context.Context, container container.Container, action container.ContainerAction) error {
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

func (d *DockerClientService) UpdateContainer(ctx context.Context, c container.Container, progressCh chan<- container.UpdateProgress) (bool, error) {
	defer close(progressCh)

	// 1. Inspect container to get full config
	inspectResp, err := d.client.ContainerInspect(ctx, c.ID)
	if err != nil {
		progressCh <- container.UpdateProgress{Status: "error", Error: fmt.Sprintf("inspect failed: %v", err)}
		return false, err
	}

	imageName := inspectResp.Config.Image

	// 2. Pull image with progress
	reader, err := d.client.ImagePull(ctx, imageName)
	if err != nil {
		progressCh <- container.UpdateProgress{Status: "error", Error: fmt.Sprintf("pull failed: %v", err)}
		return false, err
	}
	defer reader.Close()

	updated := false
	decoder := json.NewDecoder(reader)
	for {
		var event pullEvent
		if err := decoder.Decode(&event); err == io.EOF {
			break
		} else if err != nil {
			progressCh <- container.UpdateProgress{Status: "error", Error: fmt.Sprintf("pull decode failed: %v", err)}
			return false, err
		}

		progressCh <- container.UpdateProgress{
			Status:  "pulling",
			Layer:   event.ID,
			Current: event.ProgressDetail.Current,
			Total:   event.ProgressDetail.Total,
		}

		if strings.HasPrefix(event.Status, "Status: Downloaded newer image") {
			updated = true
		}
	}

	// 3. If no new layers, report up-to-date
	if !updated {
		progressCh <- container.UpdateProgress{Status: "up-to-date"}
		return false, nil
	}

	// 4. Check if this is a swarm service
	serviceName := c.Labels["com.docker.swarm.service.name"]
	if serviceName != "" {
		progressCh <- container.UpdateProgress{Status: "recreating"}
		serviceID := c.Labels["com.docker.swarm.service.id"]
		if err := d.client.ServiceUpdate(ctx, serviceID, imageName); err != nil {
			progressCh <- container.UpdateProgress{Status: "error", Error: fmt.Sprintf("service update failed: %v", err)}
			return false, err
		}
		progressCh <- container.UpdateProgress{Status: "done"}
		return true, nil
	}

	// 5. Standalone container: check for self-update
	if strings.Contains(imageName, "amir20/dozzle") {
		progressCh <- container.UpdateProgress{Status: "error", Error: "Dozzle cannot update itself. Please restart manually."}
		return false, fmt.Errorf("cannot self-update: stopping Dozzle would terminate the update process")
	}

	// 6. Standalone container: stop -> remove -> create -> start
	progressCh <- container.UpdateProgress{Status: "recreating"}

	containerName := strings.TrimPrefix(inspectResp.Name, "/")

	// Stop if running
	if c.State == "running" {
		if err := d.client.ContainerActions(ctx, container.Stop, c.ID); err != nil {
			progressCh <- container.UpdateProgress{Status: "error", Error: fmt.Sprintf("stop failed: %v", err)}
			return false, err
		}
	}

	// Remove
	if err := d.client.ContainerRemove(ctx, c.ID); err != nil {
		progressCh <- container.UpdateProgress{Status: "error", Error: fmt.Sprintf("remove failed: %v", err)}
		return false, err
	}

	// Create with same config
	newID, err := d.client.ContainerCreate(ctx, inspectResp, containerName)
	if err != nil {
		progressCh <- container.UpdateProgress{Status: "error", Error: fmt.Sprintf("create failed: %v", err)}
		return false, err
	}

	// Start
	if err := d.client.ContainerActions(ctx, container.Start, newID); err != nil {
		progressCh <- container.UpdateProgress{Status: "error", Error: fmt.Sprintf("start failed: %v", err)}
		return false, err
	}

	progressCh <- container.UpdateProgress{Status: "done"}
	return true, nil
}

func (d *DockerClientService) ListContainers(ctx context.Context, labels container.ContainerLabels) ([]container.Container, error) {
	return d.store.ListContainers(labels)
}

func (d *DockerClientService) Host(ctx context.Context) (container.Host, error) {
	return d.client.Host(), nil
}

func (d *DockerClientService) SubscribeStats(ctx context.Context, stats chan<- container.ContainerStat) {
	d.store.SubscribeStats(ctx, stats)
}

func (d *DockerClientService) SubscribeEvents(ctx context.Context, events chan<- container.ContainerEvent) {
	d.store.SubscribeEvents(ctx, events)
}

func (d *DockerClientService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container) {
	d.store.SubscribeNewContainers(ctx, containers)
}

func (d *DockerClientService) Attach(ctx context.Context, c container.Container, events container.ExecEventReader, stdout io.Writer) error {
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

func (d *DockerClientService) Exec(ctx context.Context, c container.Container, cmd []string, events container.ExecEventReader, stdout io.Writer) error {
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
