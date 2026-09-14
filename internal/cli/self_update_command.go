package cli

import (
	"context"
	"embed"
	"time"

	"github.com/amir20/dozzle/internal/selfupdate"
)

// SelfUpdateCmd is what the helper container started by selfupdate.Start runs.
// It is not meant to be run by hand.
type SelfUpdateCmd struct {
	Target string `arg:"--target,required" help:"id of the Dozzle container to replace"`
}

func (s *SelfUpdateCmd) Run(args Args, embeddedCerts embed.FS) error {
	// Deliberately not tied to signals: abandoning the swap halfway is worse
	// than finishing it or rolling back.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	return selfupdate.Run(ctx, s.Target)
}
