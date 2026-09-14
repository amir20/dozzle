package selfupdate

import (
	"context"
	"fmt"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/client"
)

// dockerAPI is the slice of the moby client this package uses, so tests can
// swap in a fake that records what was asked of the daemon.
type dockerAPI interface {
	ContainerInspect(ctx context.Context, containerID string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error)
	ContainerCreate(ctx context.Context, options client.ContainerCreateOptions) (client.ContainerCreateResult, error)
	ContainerStart(ctx context.Context, containerID string, options client.ContainerStartOptions) (client.ContainerStartResult, error)
	ContainerStop(ctx context.Context, containerID string, options client.ContainerStopOptions) (client.ContainerStopResult, error)
	ContainerRename(ctx context.Context, containerID string, options client.ContainerRenameOptions) (client.ContainerRenameResult, error)
	ContainerRemove(ctx context.Context, containerID string, options client.ContainerRemoveOptions) (client.ContainerRemoveResult, error)
	ImageInspect(ctx context.Context, imageID string, opts ...client.ImageInspectOption) (client.ImageInspectResult, error)
	ImagePull(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error)
	ServiceInspect(ctx context.Context, serviceID string, options client.ServiceInspectOptions) (client.ServiceInspectResult, error)
	ServiceUpdate(ctx context.Context, serviceID string, options client.ServiceUpdateOptions) (client.ServiceUpdateResult, error)
	Close() error
}

// newClient connects to the engine the same way Dozzle's local client does:
// DOCKER_HOST and friends, falling back to the default socket.
var newClient = func(ctx context.Context) (dockerAPI, error) {
	cli, err := client.New(client.FromEnv, client.WithUserAgent("Docker-Client/Dozzle"))
	if err != nil {
		return nil, err
	}
	if _, err := cli.Ping(ctx, client.PingOptions{NegotiateAPIVersion: true}); err != nil {
		cli.Close()
		return nil, fmt.Errorf("docker daemon unreachable: %w", err)
	}
	return cli, nil
}

func isNotFound(err error) bool {
	return err != nil && cerrdefs.IsNotFound(err)
}

func isConflict(err error) bool {
	return err != nil && cerrdefs.IsConflict(err)
}
