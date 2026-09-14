package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/profile"
	"github.com/rs/zerolog/log"
)

// setupWindow is how long an install with no auth accepts setup writes from
// anyone who can reach it.
const setupWindow = 15 * time.Minute

// Seams for tests. None of these change at runtime.
var (
	setupConfigPath   = config.Path
	setupPersisted    = profile.Persisted
	setupSelfID       = profile.SelfContainerID
	setupRestartDelay = 500 * time.Millisecond
	// setupRestarter restarts this process's own container. nil means use the
	// local docker client.
	setupRestarter func(h *handler, id string) error
)

var setupUsernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

type setupLocked struct {
	AuthProvider  bool `json:"authProvider"`
	EnableActions bool `json:"enableActions"`
	EnableShell   bool `json:"enableShell"`
}

type setupPending struct {
	AuthProvider  *string `json:"authProvider,omitempty"`
	EnableActions *bool   `json:"enableActions,omitempty"`
	EnableShell   *bool   `json:"enableShell,omitempty"`
}

type setupState struct {
	Mode            string       `json:"mode"`
	DataPersisted   bool         `json:"dataPersisted"`
	AuthProvider    string       `json:"authProvider"`
	UsersFileExists bool         `json:"usersFileExists"`
	EnableActions   bool         `json:"enableActions"`
	EnableShell     bool         `json:"enableShell"`
	Locked          setupLocked  `json:"locked"`
	Pending         setupPending `json:"pending"`
	CanRestart      bool         `json:"canRestart"`
	WindowOpen      bool         `json:"windowOpen"`
	CanWrite        bool         `json:"canWrite"`
}

func setupDataDir() string {
	return filepath.Dir(setupConfigPath)
}

func setupUsersFileExists() bool {
	for _, name := range []string{"users.yml", "users.yaml"} {
		if _, err := os.Stat(filepath.Join(setupDataDir(), name)); err == nil {
			return true
		}
	}
	return false
}

// SetupWindowStart is when the no-login window of this process opens. A restart
// the wizard triggered carries the previous window's start forward, so calling
// restart just before it closes cannot keep it open. Any other restart, or one
// after the window has passed, opens a fresh window.
func SetupWindowStart(now time.Time) time.Time {
	file, err := config.Load(setupConfigPath)
	if err != nil || file.SetupWindowStartedAt == nil {
		return now
	}
	since := now.Sub(*file.SetupWindowStartedAt)
	if since < 0 || since >= setupWindow {
		return now
	}
	return *file.SetupWindowStartedAt
}

func (h *handler) setupWindowOpen() bool {
	return h.config.Authorization.Provider == NONE && time.Since(h.config.Setup.StartedAt) < setupWindow
}

func (h *handler) setupCanWrite(r *http.Request) bool {
	if h.config.Authorization.Provider == NONE {
		return h.setupWindowOpen()
	}
	user := auth.UserFromContext(r.Context())
	// Role.Has matches any bit, so compare the whole mask: setup is for
	// someone who holds every role.
	return user != nil && user.Roles&auth.All == auth.All
}

func (h *handler) setupCanRestart() bool {
	return h.config.Mode == "server" && setupSelfID() != ""
}

// setupPendingChanges lists what dozzle.yml asks for that this process is not
// running with, i.e. what a restart would change. A locked setting ignores the
// file, so it is never pending.
func (h *handler) setupPendingChanges() (setupPending, error) {
	var p setupPending
	file, err := config.Load(setupConfigPath)
	if err != nil {
		return p, err
	}
	locked := h.config.Setup
	if !locked.LockedAuthProvider && file.AuthProvider != nil && *file.AuthProvider != string(h.config.Authorization.Provider) {
		p.AuthProvider = file.AuthProvider
	}
	if !locked.LockedEnableActions && file.EnableActions != nil && *file.EnableActions != h.config.EnableActions {
		p.EnableActions = file.EnableActions
	}
	if !locked.LockedEnableShell && file.EnableShell != nil && *file.EnableShell != h.config.EnableShell {
		p.EnableShell = file.EnableShell
	}
	return p, nil
}

func (h *handler) getSetup(w http.ResponseWriter, r *http.Request) {
	pending, err := h.setupPendingChanges()
	if err != nil {
		log.Error().Err(err).Msg("could not read setup config")
		http.Error(w, "could not read dozzle.yml", http.StatusInternalServerError)
		return
	}

	state := setupState{
		Mode:            h.config.Mode,
		DataPersisted:   setupPersisted(),
		AuthProvider:    string(h.config.Authorization.Provider),
		UsersFileExists: setupUsersFileExists(),
		EnableActions:   h.config.EnableActions,
		EnableShell:     h.config.EnableShell,
		Locked: setupLocked{
			AuthProvider:  h.config.Setup.LockedAuthProvider,
			EnableActions: h.config.Setup.LockedEnableActions,
			EnableShell:   h.config.Setup.LockedEnableShell,
		},
		Pending:    pending,
		CanRestart: h.setupCanRestart(),
		WindowOpen: h.setupWindowOpen(),
		CanWrite:   h.setupCanWrite(r),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(state); err != nil {
		log.Error().Err(err).Msg("error encoding setup state")
	}
}

func decodeSetupBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return false
	}
	return true
}

type setupAccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// createSetupAccount writes the first users.yml and switches dozzle.yml to
// simple auth. It takes effect on the next restart.
func (h *handler) createSetupAccount(w http.ResponseWriter, r *http.Request) {
	if !h.setupCanWrite(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if !setupPersisted() {
		http.Error(w, "data directory is not persisted", http.StatusPreconditionFailed)
		return
	}
	if h.config.Setup.LockedAuthProvider {
		http.Error(w, "auth provider is set by flag or env", http.StatusConflict)
		return
	}
	if setupUsersFileExists() {
		http.Error(w, "users file already exists", http.StatusConflict)
		return
	}

	var req setupAccountRequest
	if !decodeSetupBody(w, r, &req) {
		return
	}
	if !setupUsernamePattern.MatchString(req.Username) {
		http.Error(w, "invalid username", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	buffer := auth.GenerateUsers(auth.User{
		Username: req.Username,
		Name:     req.Username,
		Email:    req.Email,
		Password: req.Password,
	}, true)

	// O_EXCL so two concurrent requests cannot both write an account.
	path := filepath.Join(setupDataDir(), "users.yml")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(err, os.ErrExist) {
		http.Error(w, "users file already exists", http.StatusConflict)
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("could not create users file")
		http.Error(w, "could not write users file", http.StatusInternalServerError)
		return
	}
	if _, err := f.Write(buffer.Bytes()); err != nil {
		f.Close()
		os.Remove(path)
		log.Error().Err(err).Msg("could not write users file")
		http.Error(w, "could not write users file", http.StatusInternalServerError)
		return
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		log.Error().Err(err).Msg("could not write users file")
		http.Error(w, "could not write users file", http.StatusInternalServerError)
		return
	}

	provider := string(SIMPLE)
	if err := config.Update(setupConfigPath, func(c *config.File) { c.AuthProvider = &provider }); err != nil {
		// A leftover users.yml would turn every retry into a 409.
		os.Remove(path)
		log.Error().Err(err).Msg("could not update dozzle.yml")
		http.Error(w, "could not write dozzle.yml", http.StatusInternalServerError)
		return
	}

	log.Info().Str("username", req.Username).Msg("setup created the first account")
	w.WriteHeader(http.StatusCreated)
}

type setupAuthRequest struct {
	Provider string `json:"provider"`
}

func (h *handler) updateSetupAuth(w http.ResponseWriter, r *http.Request) {
	if !h.setupCanWrite(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if !setupPersisted() {
		http.Error(w, "data directory is not persisted", http.StatusPreconditionFailed)
		return
	}
	if h.config.Setup.LockedAuthProvider {
		http.Error(w, "auth provider is set by flag or env", http.StatusConflict)
		return
	}

	var req setupAuthRequest
	if !decodeSetupBody(w, r, &req) {
		return
	}
	if req.Provider != string(FORWARD_PROXY) {
		http.Error(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	provider := req.Provider
	if err := config.Update(setupConfigPath, func(c *config.File) { c.AuthProvider = &provider }); err != nil {
		log.Error().Err(err).Msg("could not update dozzle.yml")
		http.Error(w, "could not write dozzle.yml", http.StatusInternalServerError)
		return
	}

	log.Info().Str("provider", provider).Msg("setup changed the auth provider")
	w.WriteHeader(http.StatusNoContent)
}

type setupConfigRequest struct {
	EnableActions *bool `json:"enableActions"`
	EnableShell   *bool `json:"enableShell"`
}

// updateSetupConfig only writes dozzle.yml. Action and shell routes are decided
// at startup, so nothing here turns them on without a restart.
func (h *handler) updateSetupConfig(w http.ResponseWriter, r *http.Request) {
	if !h.setupCanWrite(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	var req setupConfigRequest
	if !decodeSetupBody(w, r, &req) {
		return
	}
	if (req.EnableActions != nil && h.config.Setup.LockedEnableActions) ||
		(req.EnableShell != nil && h.config.Setup.LockedEnableShell) {
		http.Error(w, "setting is set by flag or env", http.StatusConflict)
		return
	}

	if req.EnableActions != nil || req.EnableShell != nil {
		err := config.Update(setupConfigPath, func(c *config.File) {
			if req.EnableActions != nil {
				c.EnableActions = req.EnableActions
			}
			if req.EnableShell != nil {
				c.EnableShell = req.EnableShell
			}
		})
		if err != nil {
			log.Error().Err(err).Msg("could not update dozzle.yml")
			http.Error(w, "could not write dozzle.yml", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) restartSetup(w http.ResponseWriter, r *http.Request) {
	pending, err := h.setupPendingChanges()
	if err != nil {
		log.Error().Err(err).Msg("could not read setup config")
		http.Error(w, "could not read dozzle.yml", http.StatusInternalServerError)
		return
	}
	allowed := h.setupCanWrite(r)
	if !allowed && h.config.Authorization.Provider == NONE {
		// Turning login on must never be stranded behind a closed window: the
		// restart that applies it is what protects the install. The login routes
		// only write while the window is open, so a pending provider was chosen
		// then (or by whoever can edit dozzle.yml).
		allowed = pending.AuthProvider != nil
	}
	if !allowed {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if pending == (setupPending{}) {
		http.Error(w, "nothing to apply", http.StatusConflict)
		return
	}
	if !h.setupCanRestart() {
		http.Error(w, "cannot restart this process", http.StatusConflict)
		return
	}
	if h.config.Authorization.Provider == NONE {
		// Carry this window into the next process so restarting cannot extend it.
		started := h.config.Setup.StartedAt
		if err := config.Update(setupConfigPath, func(c *config.File) { c.SetupWindowStartedAt = &started }); err != nil {
			log.Error().Err(err).Msg("could not update dozzle.yml")
			http.Error(w, "could not write dozzle.yml", http.StatusInternalServerError)
			return
		}
	}

	id := setupSelfID()
	restart := setupRestarter
	if restart == nil {
		restart = restartOwnContainer
	}

	w.WriteHeader(http.StatusAccepted)
	log.Info().Str("container", id).Msg("setup is restarting dozzle")
	time.AfterFunc(setupRestartDelay, func() {
		if err := restart(h, id); err != nil {
			log.Error().Err(err).Msg("setup could not restart dozzle")
		}
	})
}

// restartOwnContainer goes straight to the local docker clients rather than
// FindContainer, whose label filters may well exclude Dozzle itself.
func restartOwnContainer(h *handler, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var lastErr error = errors.New("own container not found on any local client")
	for _, client := range h.hostService.LocalClients() {
		if _, err := client.FindContainer(ctx, id); err != nil {
			lastErr = err
			continue
		}
		return client.ContainerActions(ctx, container.Restart, id)
	}
	return lastErr
}
