package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

// fileBackedSecrets maps a setting's environment variable to the field it fills.
//
// Each of these also accepts a <VAR>_FILE counterpart naming a file to read the
// value from, which is the convention Docker's own images use for secrets. A
// value passed that way stays out of `docker inspect`, the compose file, and the
// shell history, so a Docker or Swarm secret can be mounted at /run/secrets and
// pointed at directly.
//
// Only genuine secrets belong here. Client ids are not secret and reading one
// from a file buys nothing.
func fileBackedSecrets(args *Args) map[string]*string {
	return map[string]*string{
		"DOZZLE_AUTH_GITHUB_CLIENT_SECRET": &args.AuthGithubClientSecret,
		"DOZZLE_AUTH_OIDC_CLIENT_SECRET":   &args.AuthOidcClientSecret,
	}
}

// fileBackedEnvNames is the sorted list of settings that accept a _FILE variant,
// used to keep ValidateEnvVars from warning about them.
func fileBackedEnvNames() []string {
	names := make([]string, 0, 2)
	for env := range fileBackedSecrets(&Args{}) {
		names = append(names, env)
	}
	slices.Sort(names)

	return names
}

// applyFileBackedSecrets fills each secret from the file named by its _FILE
// counterpart. lookup is injected so this is testable without touching the
// process environment.
func applyFileBackedSecrets(args *Args, lookup func(string) string) error {
	secrets := fileBackedSecrets(args)

	for _, env := range fileBackedEnvNames() {
		field := secrets[env]

		path := strings.TrimSpace(lookup(env + "_FILE"))
		if path == "" {
			continue
		}

		// Silently preferring one would leave an operator staring at a value
		// they thought they had overridden.
		if *field != "" {
			return fmt.Errorf("%s and %s_FILE are both set, use one or the other", env, env)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read %s_FILE: %w", env, err)
		}

		// Trailing newlines are what `echo secret > file` and most secret
		// managers produce, and a client secret never has meaningful
		// surrounding whitespace.
		secret := strings.TrimSpace(string(data))
		if secret == "" {
			return fmt.Errorf("%s_FILE at %s is empty", env, path)
		}

		*field = secret
	}

	return nil
}
