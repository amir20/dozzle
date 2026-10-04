package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container/agent"
	"github.com/amir20/dozzle/internal/updatepolicy"
	"github.com/rs/zerolog/log"
)

// Locked records which settings came from a flag or env var. Those always win
// over dozzle.yml, and the setup wizard shows them read-only.
type Locked struct {
	AuthProvider  bool
	EnableActions bool
	EnableShell   bool
	// AutoUpdate and AutoUpdateTime are locked separately, but the wizard shows
	// the schedule read-only when either is.
	AutoUpdate       bool
	AutoUpdateTime   bool
	UpdateContainers bool
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

		AutoUpdate:     setByOperator(argv, lookupEnv, "auto-update", "DOZZLE_AUTO_UPDATE"),
		AutoUpdateTime: setByOperator(argv, lookupEnv, "auto-update-time", "DOZZLE_AUTO_UPDATE_TIME"),

		UpdateContainers: setByOperator(argv, lookupEnv, "update-containers", "DOZZLE_UPDATE_CONTAINERS"),
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
	// The scheduler re-reads the file every minute; these only record what it
	// said at startup.
	if !args.Locked.AutoUpdate && file.AutoUpdate != nil {
		args.AutoUpdate = *file.AutoUpdate
	}
	if !args.Locked.AutoUpdateTime && file.AutoUpdateTime != nil {
		args.AutoUpdateTime = *file.AutoUpdateTime
	}
	if !args.Locked.UpdateContainers && file.UpdateContainers != nil {
		args.UpdateContainers = *file.UpdateContainers
	}

	// Agents from the file join the ones from the flag or env var. One listed in
	// both is the operator's, so it stays out of FileAgents and the UI cannot
	// remove it.
	args.EnvAgents = slices.Clone(args.RemoteAgent)
	args.FileAgents = nil
	for _, endpoint := range file.RemoteAgents {
		endpoint = strings.TrimSpace(endpoint)
		address, _, _, err := agent.ParseEndpoint(endpoint)
		if endpoint == "" || err != nil {
			continue
		}
		// Compared by address: "nas:7007" and "nas:7007|nas" are one agent, and
		// dialing it twice only ends in a duplicate host warning.
		sameAgent := func(existing string) bool {
			a, _, _, _ := agent.ParseEndpoint(strings.TrimSpace(existing))
			return a == address
		}
		if slices.ContainsFunc(args.RemoteAgent, sameAgent) || slices.ContainsFunc(args.FileAgents, sameAgent) {
			continue
		}
		args.FileAgents = append(args.FileAgents, endpoint)
	}
	args.RemoteAgent = append(args.RemoteAgent, args.FileAgents...)

	args.PrivateAgents = nil
	for _, endpoint := range file.PrivateAgents {
		endpoint = strings.TrimSpace(endpoint)
		if slices.Contains(args.FileAgents, endpoint) {
			args.PrivateAgents = append(args.PrivateAgents, endpoint)
		}
	}
}

// validateAutoUpdate rejects a bad --auto-update, --auto-update-time or
// --update-containers. A bad
// value in dozzle.yml is not fatal: the scheduler treats it as the default.
func validateAutoUpdate(args Args) error {
	if args.Locked.AutoUpdate && args.AutoUpdate != "" && !config.ValidAutoUpdateMode(args.AutoUpdate) {
		return fmt.Errorf("invalid auto update mode %q (expected off, daily or weekly)", args.AutoUpdate)
	}
	if args.Locked.AutoUpdateTime && args.AutoUpdateTime != "" && !config.ValidAutoUpdateTime(args.AutoUpdateTime) {
		return fmt.Errorf("invalid auto update time %q (expected HH:MM)", args.AutoUpdateTime)
	}
	if args.Locked.UpdateContainers && args.UpdateContainers != "" && !updatepolicy.ValidMode(args.UpdateContainers) {
		return fmt.Errorf("invalid update containers mode %q (expected off, labelled or all)", args.UpdateContainers)
	}
	return nil
}

func loadConfigFile(args *Args) {
	file, err := config.Load(config.Path)
	if err != nil {
		log.Warn().Err(err).Str("path", config.Path).Msg("Could not read config file, ignoring it")
		file = config.File{}
	}
	applyConfigFile(args, file, os.Args[1:], os.LookupEnv)
	if err := validateAutoUpdate(*args); err != nil {
		log.Fatal().Err(err).Msg("Invalid auto update setting")
	}
}
