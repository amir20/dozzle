package cli

import (
	"embed"
	"fmt"
	"os"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type GenerateCmd struct {
	Username    string `arg:"positional"`
	Password    string `arg:"--password, -p" help:"sets the password for the user"`
	SkipConfirm bool   `arg:"--skip-confirm" help:"skip password confirmation prompt"`
	Name        string `arg:"--name, -n" help:"sets the display name for the user"`
	Email       string `arg:"--email, -e" help:"sets the email for the user"`
	Filter      string `arg:"--user-filter" help:"sets the filter for the user. This can be a comma separated list of filters."`
}

func (g *GenerateCmd) Run(args Args, embeddedCerts embed.FS) error {
	writer := zerolog.NewConsoleWriter()
	log.Logger = log.Output(writer)
	StartEvent(args, "", nil, "generate")
	if args.Generate.Username == "" || args.Generate.Password == "" {
		return fmt.Errorf("username and password are required")
	}

	if !args.Generate.SkipConfirm {
		fmt.Print("Confirm password: ")
		var confirmPassword string
		fmt.Scanln(&confirmPassword)
		if confirmPassword != args.Generate.Password {
			return fmt.Errorf("passwords do not match")
		}
	}

	buffer := auth.GenerateUsers(auth.User{
		Username: args.Generate.Username,
		Password: args.Generate.Password,
		Name:     args.Generate.Name,
		Email:    args.Generate.Email,
		Filter:   args.Generate.Filter,
	}, true)

	if _, err := os.Stdout.Write(buffer.Bytes()); err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	return nil
}
