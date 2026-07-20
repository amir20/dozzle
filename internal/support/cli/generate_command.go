package cli

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

type GenerateCmd struct {
	Username        string `arg:"positional"`
	Password        string `arg:"--password, -p" help:"sets the password for the user"`
	Name            string `arg:"--name, -n" help:"sets the display name for the user"`
	Email           string `arg:"--email, -e" help:"sets the email for the user"`
	Filter          string `arg:"--user-filter" help:"sets the filter for the user. This can be a comma separated list of filters."`
	RolesConfigured string `arg:"--user-roles" help:"sets the roles for the user. This can be a comma separated list of roles."`
}

func (g *GenerateCmd) Run(args Args, embeddedCerts embed.FS) error {
	writer := zerolog.NewConsoleWriter()
	log.Logger = log.Output(writer)
	StartEvent(args, "", nil, "generate")
	if args.Generate.Username == "" {
		return fmt.Errorf("username is required")
	}

	password := args.Generate.Password
	if password == "" {
		var err error
		if password, err = readPassword(); err != nil {
			return err
		}
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}

	buffer := auth.GenerateUsers(auth.User{
		Username:        args.Generate.Username,
		Password:        password,
		Name:            args.Generate.Name,
		Email:           args.Generate.Email,
		Filter:          args.Generate.Filter,
		RolesConfigured: args.Generate.RolesConfigured,
	}, true)

	if _, err := os.Stdout.Write(buffer.Bytes()); err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	return nil
}

// readPassword reads a password from stdin. Prompts are written to stderr so
// they don't pollute stdout (which is commonly redirected to users.yml). When
// stdin is a terminal the input is read without echo; otherwise a single line
// is read (supports piping, e.g. `echo secret | dozzle generate ...`).
func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprint(os.Stderr, "Password: ")
		bytePassword, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		return strings.TrimRight(string(bytePassword), "\r\n"), nil
	}

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("failed to read password from stdin: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}
