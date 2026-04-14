package cloud

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/amir20/dozzle/internal/deploy"
	pb "github.com/amir20/dozzle/proto/cloud"
	"github.com/docker/docker/client"
)

type deployComposeArgs struct {
	YAML    string `json:"yaml"`
	Project string `json:"project"`
}

func executeDeployCompose(ctx context.Context, argsJSON string) (*pb.CallToolResponse, error) {
	var args deployComposeArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.YAML == "" {
		return nil, fmt.Errorf("yaml is required")
	}
	if args.Project == "" {
		return nil, fmt.Errorf("project is required")
	}

	project, err := deploy.ParseCompose([]byte(args.YAML), args.Project)
	if err != nil {
		return nil, fmt.Errorf("parsing compose file: %w", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	defer cli.Close()

	deployer := deploy.NewDeployer(cli)
	if err := deployer.Deploy(ctx, project); err != nil {
		return nil, fmt.Errorf("deploying: %w", err)
	}

	message := fmt.Sprintf("Successfully deployed project %q with %d services.", args.Project, len(project.Services))

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Deploy{Deploy: &pb.DeployResult{
			Success: true,
			Project: args.Project,
			Message: message,
		}},
	}, nil
}
