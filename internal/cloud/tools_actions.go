package cloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amir20/dozzle/internal/container"
	pb "github.com/amir20/dozzle/proto/cloud"
)

type containerActionArgs struct {
	ContainerID string `json:"container_id"`
	Host        string `json:"host_id"`
}

func executeContainerAction(ctx context.Context, name string, argsJSON string, hostService ToolHostService, labels container.ContainerLabels) (*pb.CallToolResponse, error) {
	var args containerActionArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	action, err := resolveAction(name)
	if err != nil {
		return nil, err
	}

	if args.ContainerID == "" {
		return nil, fmt.Errorf("container_id is required")
	}
	if args.Host == "" {
		return nil, fmt.Errorf("host is required")
	}

	cs, err := hostService.FindContainer(args.Host, args.ContainerID, labels)
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}

	if err := cs.Action(ctx, action); err != nil {
		return nil, fmt.Errorf("action failed: %w", err)
	}

	message := fmt.Sprintf("Successfully %s container %s.", pastTense(action), args.ContainerID)

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Action{Action: &pb.ActionResult{
			Success:     true,
			ContainerId: args.ContainerID,
			Action:      string(action),
			Message:     message,
		}},
	}, nil
}

func executeUpdateContainer(ctx context.Context, argsJSON string, hostService ToolHostService, labels container.ContainerLabels) (*pb.CallToolResponse, error) {
	var args containerActionArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.ContainerID == "" {
		return nil, fmt.Errorf("container_id is required")
	}
	if args.Host == "" {
		return nil, fmt.Errorf("host is required")
	}

	cs, err := hostService.FindContainer(args.Host, args.ContainerID, labels)
	if err != nil {
		return nil, fmt.Errorf("container not found: %w", err)
	}

	progressCh := make(chan container.UpdateProgress)
	var updated bool
	var updateErr error
	done := make(chan struct{})
	go func() {
		updated, updateErr = cs.Update(ctx, progressCh)
		close(done)
	}()
	for range progressCh {
	}
	<-done
	if updateErr != nil {
		return nil, fmt.Errorf("update failed: %w", updateErr)
	}

	message := fmt.Sprintf("Successfully updated container %s by pulling the latest image and recreating it.", args.ContainerID)
	if !updated {
		message = fmt.Sprintf("Container %s is already running the latest image. No update was needed.", args.ContainerID)
	}

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Action{Action: &pb.ActionResult{
			Success:     true,
			ContainerId: args.ContainerID,
			Action:      "update",
			Message:     message,
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
	default:
		return "", fmt.Errorf("unknown action: %s", name)
	}
}
