package cli

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/amir20/dozzle/internal/healthcheck"
)

type HealthcheckCmd struct {
}

func (h *HealthcheckCmd) Run(args Args, embeddedCerts embed.FS) error {
	files, err := os.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	agentAddress := ""
	for _, file := range files {
		if match, _ := filepath.Match("agent-*.addr", file.Name()); match {
			data, err := os.ReadFile(file.Name())
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			agentAddress = string(data)
			break
		}
	}
	if agentAddress == "" {
		if err := healthcheck.HttpRequest(args.Addr, args.Base); err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
	} else {
		certs, err := ReadCertificates(embeddedCerts)
		if err != nil {
			return fmt.Errorf("failed to read certificates: %w", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), args.Timeout)
		defer cancel()
		if err := healthcheck.RPCRequest(ctx, agentAddress, certs); err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
	}

	return nil
}
