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
	Host        string `json:"host"`
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
