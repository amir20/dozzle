package cli

import (
	"os"
	"strings"

	"github.com/amir20/dozzle/internal/config"
	"github.com/rs/zerolog/log"
)

// Locked records which settings came from a flag or env var. Those always win
// over dozzle.yml, and the setup wizard shows them read-only.
type Locked struct {
	AuthProvider  bool
	EnableActions bool
	EnableShell   bool
}

// setByOperator reports whether flag appears in argv (as flag or flag=value,
// with any number of leading dashes, as go-arg accepts) or env is present in
// the environment. flag is the bare name, e.g. "auth-provider".
func setByOperator(argv []string, lookupEnv func(string) (string, bool), flag, env string) bool {
	if _, ok := lookupEnv(env); ok {
		return true
	}
	for _, a := range argv {
		if a == "--" {
			break
		}
		if !strings.HasPrefix(a, "-") {
			continue
		}
		name, _, _ := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if name == flag {
			return true
		}
	}
	return false
}

// applyConfigFile layers file values under flags and env vars. argv excludes
// the program name.
func applyConfigFile(args *Args, file config.File, argv []string, lookupEnv func(string) (string, bool)) {
	args.Locked = Locked{
		AuthProvider:  setByOperator(argv, lookupEnv, "auth-provider", "DOZZLE_AUTH_PROVIDER"),
		EnableActions: setByOperator(argv, lookupEnv, "enable-actions", "DOZZLE_ENABLE_ACTIONS"),
		EnableShell:   setByOperator(argv, lookupEnv, "enable-shell", "DOZZLE_ENABLE_SHELL"),
	}

	if !args.Locked.AuthProvider && file.AuthProvider != nil {
		args.AuthProvider = *file.AuthProvider
	}
	if !args.Locked.EnableActions && file.EnableActions != nil {
		args.EnableActions = *file.EnableActions
	}
	if !args.Locked.EnableShell && file.EnableShell != nil {
		args.EnableShell = *file.EnableShell
	}
}

func loadConfigFile(args *Args) {
	file, err := config.Load(config.Path)
	if err != nil {
		log.Warn().Err(err).Str("path", config.Path).Msg("Could not read config file, ignoring it")
		file = config.File{}
	}
	applyConfigFile(args, file, os.Args[1:], os.LookupEnv)
}
