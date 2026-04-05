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

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Action{Action: &pb.ActionResult{
			Success:     true,
			ContainerId: args.ContainerID,
			Action:      string(action),
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

	progressCh := make(chan container.UpdateProgress, 100)
	errCh := make(chan error, 1)

	go func() {
		errCh <- cs.Update(ctx, progressCh)
	}()

	// Drain progress channel and capture final status
	var lastStatus string
	var lastError string
	for progress := range progressCh {
		lastStatus = progress.Status
		if progress.Error != "" {
			lastError = progress.Error
		}
	}

	if err := <-errCh; err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}

	action := "update"
	if lastStatus == "up-to-date" {
		action = "update (already up-to-date)"
	}
	if lastError != "" {
		return nil, fmt.Errorf("update failed: %s", lastError)
	}

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Action{Action: &pb.ActionResult{
			Success:     true,
			ContainerId: args.ContainerID,
			Action:      action,
		}},
	}, nil
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
