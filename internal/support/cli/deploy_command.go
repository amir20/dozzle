package cli

import (
	"context"
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/amir20/dozzle/internal/deploy"
	"github.com/amir20/dozzle/internal/docker"
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

	localClient, err := docker.NewLocalClient("")
	if err != nil {
		return fmt.Errorf("creating docker client: %w", err)
	}
	raw := localClient.RawClient()
	if raw == nil {
		return fmt.Errorf("local Docker client is missing a raw handle")
	}

	timeout := max(args.Timeout, 10*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	mgr := deploy.NewManager(raw, deploy.DefaultStacksDir)
	if err := mgr.Deploy(ctx, dc.Project, data, nil); err != nil {
		return fmt.Errorf("deploying: %w", err)
	}

	log.Info().Str("project", dc.Project).Msg("Deployment complete")
	return nil
}
