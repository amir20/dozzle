package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
)

type containerActionArgs struct {
	ContainerID string `json:"container_id"`
	Host        string `json:"host_id"`
}

func executeContainerAction(ctx context.Context, name string, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	var args containerActionArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	action, err := resolveAction(name)
	if err != nil {
		return nil, err
	}

	hostID, containerID, err := resolveContainerRef(args.ContainerID, args.Host, deps)
	if err != nil {
		return nil, err
	}

	cs, err := deps.scoped().FindContainer(hostID, containerID)
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}

	if err := cs.Action(ctx, action); err != nil {
		return nil, fmt.Errorf("action failed: %w", err)
	}

	message := fmt.Sprintf("Successfully %s container %s.", pastTense(action), cs.Container.Name)

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Action{Action: &pb.ActionResult{
			Success:     true,
			ContainerId: cs.Container.ID,
			Action:      string(action),
			Message:     message,
		}},
	}, nil
}

func executeUpdateContainer(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	var args containerActionArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	hostID, containerID, err := resolveContainerRef(args.ContainerID, args.Host, deps)
	if err != nil {
		return nil, err
	}

	cs, err := deps.scoped().FindContainer(hostID, containerID)
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}
	// The host refuses too; this answers without a round trip to an agent.
	if cs.Container.State != "running" {
		return nil, notRunningError(cs.Container)
	}

	var updated bool
	updateErr := runIgnoringProgress(func(progressCh chan<- container.UpdateProgress) (err error) {
		updated, err = cs.Update(ctx, container.UpdateSourceCloud, progressCh)
		return err
	})
	if updateErr != nil {
		return nil, fmt.Errorf("update failed: %w", updateErr)
	}

	message := fmt.Sprintf("Successfully updated container %s by pulling the latest image and recreating it.", cs.Container.Name)
	if !updated {
		message = fmt.Sprintf("Container %s is already running the latest image. No update was needed.", cs.Container.Name)
	}

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Action{Action: &pb.ActionResult{
			Success:     true,
			ContainerId: cs.Container.ID,
			Action:      "update",
			Message:     message,
		}},
	}, nil
}

// runIgnoringProgress runs an update or a rollback, which closes the channel
// it is given when it ends, and drains its progress so it never blocks.
func runIgnoringProgress(run func(chan<- container.UpdateProgress) error) error {
	progressCh := make(chan container.UpdateProgress)
	var err error
	done := make(chan struct{})
	go func() {
		err = run(progressCh)
		close(done)
	}()
	for range progressCh {
	}
	<-done
	return err
}

// notRunningError refuses to update or roll back c, which is not running.
func notRunningError(c container.Container) error {
	return fmt.Errorf("container %s is %s. Start the container first: a stopped container is never updated or rolled back", c.Name, c.State)
}

type rollbackContainerArgs struct {
	ContainerID        string `json:"container_id"`
	Host               string `json:"host_id"`
	ExpectedFromDigest string `json:"expected_from_digest"`
}

// executeRollbackContainer swaps a container back to the image it ran before
// its last update. The caller names the digest it expects the container to run
// now, so a request made against an older state (a stale button, a second
// update since) is refused rather than rolling back something else.
func executeRollbackContainer(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	var args rollbackContainerArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}
	if strings.TrimSpace(args.ExpectedFromDigest) == "" {
		return nil, fmt.Errorf("expected_from_digest is required")
	}

	hostID, containerID, err := resolveContainerRef(args.ContainerID, args.Host, deps)
	if err != nil {
		return nil, err
	}

	cs, err := deps.scoped().FindContainer(hostID, containerID)
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}
	if cs.Container.State != "running" {
		return nil, notRunningError(cs.Container)
	}

	rollbackErr := runIgnoringProgress(func(progressCh chan<- container.UpdateProgress) error {
		return cs.Rollback(ctx, container.RollbackOptions{
			ExpectedFromDigest: args.ExpectedFromDigest,
		}, progressCh)
	})
	if rollbackErr != nil {
		return nil, fmt.Errorf("rollback failed: %w", rollbackErr)
	}

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Action{Action: &pb.ActionResult{
			Success:     true,
			ContainerId: cs.Container.ID,
			Action:      "rollback",
			Message:     fmt.Sprintf("Rolled back container %s to the image it ran before its last update.", cs.Container.Name),
		}},
	}, nil
}

func pastTense(action container.ContainerAction) string {
	switch action {
	case container.Start:
		return "started"
	case container.Stop:
		return "stopped"
	case container.Restart:
		return "restarted"
	case container.Remove:
		return "removed"
	default:
		return string(action) + "ed"
	}
}

func resolveAction(name string) (container.ContainerAction, error) {
	switch name {
	case "start_container":
		return container.Start, nil
	case "stop_container":
		return container.Stop, nil
	case "restart_container":
		return container.Restart, nil
	case "remove_container":
		return container.Remove, nil
	default:
		return "", fmt.Errorf("unknown action: %s", name)
	}
}
