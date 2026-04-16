package cloud

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pb "github.com/amir20/dozzle/proto/cloud"
)

// errDeployManagerNotConfigured is returned when deploy tools are invoked in a
// mode without a local docker daemon (e.g., k8s).
var errDeployManagerNotConfigured = errors.New("deploy manager is not configured")

// Deploy tool args intentionally omit host_id: the cloud router uses it to
// pick the target Dozzle instance and strips it before forwarding, so the
// received arguments never contain it. JSON unmarshal ignores unknown fields,
// so any stray host_id is silently dropped.

type deployComposeArgs struct {
	YAML    string `json:"yaml"`
	Project string `json:"project"`
}

func executeDeployCompose(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.DeployManager == nil {
		return nil, errDeployManagerNotConfigured
	}

	args, err := parseArgs[deployComposeArgs](argsJSON)
	if err != nil {
		return nil, err
	}
	if err := requireNonEmpty(map[string]string{
		"yaml":    args.YAML,
		"project": args.Project,
	}); err != nil {
		return nil, err
	}

	if err := deps.DeployManager.Deploy(ctx, args.Project, []byte(args.YAML), nil); err != nil {
		return nil, fmt.Errorf("deploying: %w", err)
	}

	return deployResponse(args.Project, fmt.Sprintf("Successfully deployed project %q.", args.Project)), nil
}

type projectArgs struct {
	Project string `json:"project"`
}

func executeListDeployVersions(_ context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.DeployManager == nil {
		return nil, errDeployManagerNotConfigured
	}

	args, err := parseArgs[projectArgs](argsJSON)
	if err != nil {
		return nil, err
	}
	if err := requireNonEmpty(map[string]string{"project": args.Project}); err != nil {
		return nil, err
	}

	versions, err := deps.DeployManager.ListVersions(args.Project)
	if err != nil {
		return nil, fmt.Errorf("listing versions: %w", err)
	}

	if len(versions) == 0 {
		return deployResponse(args.Project, "No versions found."), nil
	}

	var sb strings.Builder
	sb.Grow(len(versions) * 64)
	for _, v := range versions {
		fmt.Fprintf(&sb, "%s  %s  %s\n", shortHash(v.Hash), v.Time.Format("2006-01-02 15:04:05"), v.Message)
	}

	return deployResponse(args.Project, sb.String()), nil
}

type rollbackArgs struct {
	Project    string `json:"project"`
	CommitHash string `json:"commit_hash"`
}

type removeDeployArgs struct {
	Project       string `json:"project"`
	RemoveVolumes bool   `json:"remove_volumes"`
}

func executeRemoveDeploy(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.DeployManager == nil {
		return nil, errDeployManagerNotConfigured
	}

	args, err := parseArgs[removeDeployArgs](argsJSON)
	if err != nil {
		return nil, err
	}
	if err := requireNonEmpty(map[string]string{"project": args.Project}); err != nil {
		return nil, err
	}

	if err := deps.DeployManager.Remove(ctx, args.Project, args.RemoveVolumes, nil); err != nil {
		return nil, fmt.Errorf("removing: %w", err)
	}

	msg := fmt.Sprintf("Removed project %q (containers and networks).", args.Project)
	if args.RemoveVolumes {
		msg = fmt.Sprintf("Removed project %q (containers, networks, and volumes).", args.Project)
	}

	return deployResponse(args.Project, msg), nil
}

func executeRollbackDeploy(ctx context.Context, argsJSON string, deps ToolDeps) (*pb.CallToolResponse, error) {
	if deps.DeployManager == nil {
		return nil, errDeployManagerNotConfigured
	}

	args, err := parseArgs[rollbackArgs](argsJSON)
	if err != nil {
		return nil, err
	}
	if err := requireNonEmpty(map[string]string{
		"project":     args.Project,
		"commit_hash": args.CommitHash,
	}); err != nil {
		return nil, err
	}

	if err := deps.DeployManager.RollbackVersion(ctx, args.Project, args.CommitHash, nil); err != nil {
		return nil, fmt.Errorf("rolling back: %w", err)
	}

	return deployResponse(args.Project, fmt.Sprintf("Successfully rolled back project %q to %s.", args.Project, args.CommitHash)), nil
}

func deployResponse(project, message string) *pb.CallToolResponse {
	return &pb.CallToolResponse{
		Success: true,
		Result: &pb.CallToolResponse_Deploy{Deploy: &pb.DeployResult{
			Success: true,
			Project: project,
			Message: message,
		}},
	}
}

// shortHash truncates a git hash to its first 12 chars, or returns it intact
// if shorter (defensive — commit hashes from go-git are always 40 chars).
func shortHash(h string) string {
	if len(h) < 12 {
		return h
	}
	return h[:12]
}
