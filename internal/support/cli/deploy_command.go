package cli

import (
	"context"
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/amir20/dozzle/internal/deploy"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

type DeployCmd struct {
	File    string `arg:"positional,required" help:"path to compose file"`
	Project string `arg:"-p,--project" default:"dozzle" help:"project name for resource prefixes"`
}

func (dc *DeployCmd) Run(args Args, _ embed.FS) error {
	data, err := os.ReadFile(dc.File)
	if err != nil {
		return fmt.Errorf("reading compose file: %w", err)
	}

	log.Info().Str("project", dc.Project).Str("file", dc.File).Msg("Deploying compose file")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("creating docker client: %w", err)
	}
	defer cli.Close()

	timeout := args.Timeout
	if timeout < 10*time.Minute {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	mgr := deploy.NewManager(cli, "./data/stacks")
	if err := mgr.UpdateConfig(ctx, dc.Project, data, nil); err != nil {
		// If project doesn't exist yet, create it
		if err := mgr.CreateProject(ctx, dc.Project, data); err != nil {
			return fmt.Errorf("deploying: %w", err)
		}
	}

	log.Info().Str("project", dc.Project).Msg("Deployment complete")
	return nil
}
