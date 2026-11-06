package imagecheck

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// defaultConfigPath is where the Docker config is read from when DOCKER_CONFIG
// is unset. It matches Watchtower, so a compose file moving over keeps working.
const defaultConfigPath = "/config.json"

// login is one registry's credentials. It is never logged.
type login struct {
	username      string
	password      string
	serverAddress string
}

// basicHeader is the Authorization header value for this login.
func (l login) basicHeader() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(l.username+":"+l.password))
}

type configEntry struct {
	Auth          string `json:"auth"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	IdentityToken string `json:"identitytoken"`
}

type dockerConfig struct {
	Auths       map[string]configEntry `json:"auths"`
	CredsStore  string                 `json:"credsStore"`
	CredHelpers map[string]string      `json:"credHelpers"`
}

// configPath follows the Docker CLI: DOCKER_CONFIG names a directory.
func configPath() string {
	if dir := os.Getenv("DOCKER_CONFIG"); dir != "" {
		return filepath.Join(dir, "config.json")
	}
	return defaultConfigPath
}

// lookupLogin returns the login for registry, or nil. The file is read on every
// call rather than held in memory: this only runs on a registry challenge or a
// pull, and it means a `docker login` takes effect without a restart.
//
// A login only ever goes to the registry named by the image. Nothing is global
// across registries, which is why Watchtower's REPO_USER and REPO_PASS have no
// equivalent here: any image on the host would receive them.
func lookupLogin(registry string) *login {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			warnOnce("Could not read the Docker config, so private images cannot be checked or pulled", path, err)
		}
		return nil
	}

	var config dockerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		warnOnce("Could not parse the Docker config, so private images cannot be checked or pulled", path, err)
		return nil
	}

	want := normalizeRegistry(registry)
	// Sorted because "ghcr.io" and "https://ghcr.io" can both be present, and
	// which one wins should not change between runs.
	for _, server := range slices.Sorted(maps.Keys(config.Auths)) {
		if normalizeRegistry(server) != want {
			continue
		}
		entry := config.Auths[server]
		if found := entry.login(server); found != nil {
			return found
		}
		if entry.IdentityToken != "" {
			warnOnce("The Docker config has an identity token, which Dozzle does not support. Use a username and password or access token", path, nil)
		}
	}

	// The image has no shell or helper binaries, so these cannot be run.
	if config.CredsStore != "" || len(config.CredHelpers) > 0 {
		warnOnce("The Docker config uses a credential helper, which Dozzle does not support. Add an auth entry to a config.json made for Dozzle", path, nil)
	}
	return nil
}

func (e configEntry) login(server string) *login {
	username, password := e.Username, e.Password
	if e.Auth != "" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(e.Auth))
		if err != nil {
			return nil
		}
		var found bool
		// Only the first colon separates: a password may contain more.
		username, password, found = strings.Cut(string(decoded), ":")
		if !found {
			return nil
		}
		// `echo 'user:pass' | base64` is a common way to write this by hand,
		// and leaves a newline on the password.
		password = strings.TrimRight(password, "\x00\r\n")
	}
	if username == "" || password == "" {
		return nil
	}
	return &login{username: username, password: password, serverAddress: server}
}

// normalizeRegistry reduces a config.json key or an image's registry to a bare
// host and port, so that every spelling of Docker Hub is one registry.
func normalizeRegistry(server string) string {
	server = strings.ToLower(server)
	server = strings.TrimPrefix(server, "https://")
	server = strings.TrimPrefix(server, "http://")
	host, _, _ := strings.Cut(server, "/")
	if _, aliased := dockerHubAliases[host]; aliased {
		return defaultRegistry
	}
	return host
}

// PullAuth returns the value to send as X-Registry-Auth when pulling image, or
// "" when there is no login for its registry. The daemon does not read
// config.json itself: that is the Docker CLI's job, and here it is ours.
func PullAuth(image string) string {
	ref, err := ParseReference(image)
	if err != nil {
		return ""
	}
	found := lookupLogin(ref.Registry)
	if found == nil {
		return ""
	}
	encoded, err := json.Marshal(struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		ServerAddress string `json:"serveraddress"`
	}{found.username, found.password, found.serverAddress})
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(encoded)
}

// configChangedSince reports whether the Docker config was written after t.
func configChangedSince(t time.Time) bool {
	info, err := os.Stat(configPath())
	return err == nil && info.ModTime().After(t)
}

var warned sync.Map

// warnOnce keeps a config problem to one line: challenges are routine, so
// warning on each would repeat for every public image on every check.
func warnOnce(msg, path string, err error) {
	if !firstTime(msg) {
		return
	}
	log.Warn().Err(err).Str("path", path).Msg(msg)
}

// firstTime reports whether key has not been seen before.
func firstTime(key string) bool {
	_, seen := warned.LoadOrStore(key, struct{}{})
	return !seen
}
