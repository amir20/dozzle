package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	pb "github.com/amir20/dozzle/proto/cloud"
)

// errDeployManagerNotConfigured is returned when deploy tools are invoked in a
// mode without a local docker daemon (e.g., k8s).
var errDeployManagerNotConfigured = errors.New("deploy manager is not configured")

// Deploy tools accept host_id per the tool schema, but the current Manager is
// local-only (see main.go). The field is parsed so it isn't silently dropped;
// multi-host routing is a future extension (TODO: agent support).
type deployComposeArgs struct {
	YAML    string `json:"yaml"`
	Project string `json:"project"`
	Host    string `json:"host_id"`
}

func executeDeployCompose(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.DeployManager == nil {
		return nil, errDeployManagerNotConfigured
	}

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

	if err := deps.DeployManager.Deploy(ctx, args.Project, []byte(args.YAML), nil); err != nil {
		return nil, fmt.Errorf("deploying: %w", err)
	}

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Deploy{Deploy: &pb.DeployResult{
			Success: true,
			Project: args.Project,
			Message: fmt.Sprintf("Successfully deployed project %q.", args.Project),
		}},
	}, nil
}

type projectArgs struct {
	Project string `json:"project"`
	Host    string `json:"host_id"`
}

func executeListDeployVersions(_ context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.DeployManager == nil {
		return nil, errDeployManagerNotConfigured
	}

	var args projectArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Project == "" {
		return nil, fmt.Errorf("project is required")
	}

	versions, err := deps.DeployManager.ListVersions(args.Project)
	if err != nil {
		return nil, fmt.Errorf("listing versions: %w", err)
	}

	if len(versions) == 0 {
		return &pb.CallToolResponse{
			Success: true,
			Result: &pb.CallToolResponse_Deploy{Deploy: &pb.DeployResult{
				Success: true,
				Project: args.Project,
				Message: "No versions found.",
			}},
		}, nil
	}

	var sb strings.Builder
	for _, v := range versions {
		fmt.Fprintf(&sb, "%s  %s  %s\n", v.Hash[:12], v.Time.Format("2006-01-02 15:04:05"), v.Message)
	}

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Deploy{Deploy: &pb.DeployResult{
			Success: true,
			Project: args.Project,
			Message: sb.String(),
		}},
	}, nil
}

type rollbackArgs struct {
	Project    string `json:"project"`
	CommitHash string `json:"commit_hash"`
	Host       string `json:"host_id"`
}

type removeDeployArgs struct {
	Project       string `json:"project"`
	RemoveVolumes bool   `json:"remove_volumes"`
	Host          string `json:"host_id"`
}

func executeRemoveDeploy(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.DeployManager == nil {
		return nil, errDeployManagerNotConfigured
	}

	var args removeDeployArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Project == "" {
		return nil, fmt.Errorf("project is required")
	}

	if err := deps.DeployManager.Remove(ctx, args.Project, args.RemoveVolumes, nil); err != nil {
		return nil, fmt.Errorf("removing: %w", err)
	}

	msg := fmt.Sprintf("Removed project %q (containers and networks).", args.Project)
	if args.RemoveVolumes {
		msg = fmt.Sprintf("Removed project %q (containers, networks, and volumes).", args.Project)
	}

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Deploy{Deploy: &pb.DeployResult{
			Success: true,
			Project: args.Project,
			Message: msg,
		}},
	}, nil
}

func executeRollbackDeploy(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.DeployManager == nil {
		return nil, errDeployManagerNotConfigured
	}

	var args rollbackArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if args.Project == "" {
		return nil, fmt.Errorf("project is required")
	}
	if args.CommitHash == "" {
		return nil, fmt.Errorf("commit_hash is required")
	}

	if err := deps.DeployManager.RollbackVersion(ctx, args.Project, args.CommitHash, nil); err != nil {
		return nil, fmt.Errorf("rolling back: %w", err)
	}

	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Deploy{Deploy: &pb.DeployResult{
			Success: true,
			Project: args.Project,
			Message: fmt.Sprintf("Successfully rolled back project %q to %s.", args.Project, args.CommitHash),
		}},
	}, nil
}
