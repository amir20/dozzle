package web

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/agentcerts"
	"github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// agentService is a HostService that can add and remove agents while running.
type agentService interface {
	CanAddAgents() bool
	AddAgent(ctx context.Context, endpoint string, cert *tls.Certificate) (container.Host, error)
	RemoveAgent(endpoint string) error
	AgentHostID(endpoint string) string
}

func (h *handler) agentService() (agentService, bool) {
	s, ok := h.hostService.(agentService)
	if !ok || !s.CanAddAgents() {
		return nil, false
	}
	return s, true
}

type setupAgent struct {
	Endpoint string `json:"endpoint"`
	Address  string `json:"address"`
	Name     string `json:"name,omitempty"`
	HostID   string `json:"hostId,omitempty"`
	// Locked agents came from DOZZLE_REMOTE_AGENT, so only the operator's
	// compose file can remove them.
	Locked bool `json:"locked"`
	// Private agents authenticate with the hub's own pair.
	Private bool `json:"private"`
}

// setupAgents lists the operator's agents first, then the ones added from the
// UI. A file entry that repeats an env one is the env one.
func (h *handler) setupAgents(file config.File) []setupAgent {
	service, _ := h.agentService()
	agents := make([]setupAgent, 0, len(h.config.Setup.EnvAgents)+len(file.RemoteAgents))
	seen := map[string]bool{}
	add := func(endpoint string, locked bool) {
		address, name, _, err := agent.ParseEndpoint(endpoint)
		if err != nil || seen[address] {
			return
		}
		seen[address] = true
		a := setupAgent{Endpoint: endpoint, Address: address, Name: name, Locked: locked,
			Private: !locked && slices.Contains(file.PrivateAgents, endpoint)}
		if service != nil {
			a.HostID = service.AgentHostID(endpoint)
		}
		agents = append(agents, a)
	}
	for _, endpoint := range h.config.Setup.EnvAgents {
		add(endpoint, true)
	}
	for _, endpoint := range file.RemoteAgents {
		add(endpoint, false)
	}
	return agents
}

// setupAgentAddress checks a host:port the UI sends. The pipe is the endpoint
// separator, so it can appear in neither half.
func setupAgentAddress(address, name string) (string, error) {
	if address == "" {
		return "", errors.New("address is required")
	}
	if strings.ContainsAny(address, "| \t") || strings.Contains(address, "://") {
		return "", errors.New("address must be host:port")
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return "", errors.New("address must be host:port")
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return "", errors.New("port must be between 1 and 65535")
	}
	if strings.Contains(name, "|") || len(name) > 64 {
		return "", errors.New("name can't contain | and is at most 64 characters")
	}
	if name == "" {
		return address, nil
	}
	return address + "|" + name, nil
}

type setupAddAgentRequest struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	// Private dials the agent with the hub's own pair, which the snippet from
	// POST /api/setup/agent-cert handed to it.
	Private bool `json:"private"`
}

type setupAddAgentResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
}

// addSetupAgent connects to an agent and saves it to dozzle.yml only once it
// answers, so the UI can tell a typo or a closed port apart from success right
// away. The host shows up live, the same way a reconnecting agent does.
func (h *handler) addSetupAgent(w http.ResponseWriter, r *http.Request) {
	if !h.setupCanWrite(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	service, ok := h.agentService()
	if !ok {
		http.Error(w, "agents cannot be added in this mode", http.StatusConflict)
		return
	}
	if !setupPersisted() {
		http.Error(w, "data directory is not persisted", http.StatusPreconditionFailed)
		return
	}

	var req setupAddAgentRequest
	if !decodeSetupBody(w, r, &req) {
		return
	}
	address, name := strings.TrimSpace(req.Address), strings.TrimSpace(req.Name)
	endpoint, err := setupAgentAddress(address, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, err := config.Load(setupConfigPath)
	if err != nil {
		log.Error().Err(err).Msg("could not read setup config")
		http.Error(w, "could not read dozzle.yml", http.StatusInternalServerError)
		return
	}
	if slices.ContainsFunc(h.setupAgents(file), func(a setupAgent) bool { return a.Address == address }) {
		http.Error(w, "this agent is already added", http.StatusConflict)
		return
	}

	var cert *tls.Certificate
	if req.Private {
		pair, err := agentcerts.LoadAgentPair(setupDataDir())
		if err != nil {
			http.Error(w, "no private certificate yet, create one first", http.StatusPreconditionFailed)
			return
		}
		parsed, err := pair.TLS()
		if err != nil {
			log.Error().Err(err).Msg("could not parse the private agent certificate")
			http.Error(w, "could not read the private certificate", http.StatusInternalServerError)
			return
		}
		cert = &parsed
	}

	// Each add dials an address the caller picked. With no login, anyone who can
	// reach the page can do that during the setup window, so dials are rationed
	// and the answer names only the kind of failure, never the raw error text.
	if !agentDialLimiter.Allow() {
		http.Error(w, "too many attempts, wait a minute and try again", http.StatusTooManyRequests)
		return
	}

	host, err := service.AddAgent(r.Context(), endpoint, cert)
	switch {
	case errors.Is(err, hostservice.ErrAgentExists):
		http.Error(w, "this agent is already added", http.StatusConflict)
		return
	case errors.Is(err, hostservice.ErrDuplicateHost):
		http.Error(w, "this agent is already connected under another address", http.StatusConflict)
		return
	case err != nil:
		log.Debug().Err(err).Str("endpoint", endpoint).Msg("setup could not connect to agent")
		http.Error(w, "could not connect to agent: "+dialFailure(err), http.StatusBadGateway)
		return
	}

	if err := config.Update(setupConfigPath, func(c *config.File) {
		c.RemoteAgents = append(c.RemoteAgents, endpoint)
		if req.Private {
			c.PrivateAgents = append(c.PrivateAgents, endpoint)
		}
	}); err != nil {
		// Not saved means it would vanish on restart, so do not serve it now either.
		if err := service.RemoveAgent(endpoint); err != nil {
			log.Warn().Err(err).Str("endpoint", endpoint).Msg("could not undo agent add")
		}
		log.Error().Err(err).Msg("could not update dozzle.yml")
		http.Error(w, "could not write dozzle.yml", http.StatusInternalServerError)
		return
	}

	log.Info().Str("endpoint", endpoint).Str("host", host.Name).Msg("setup added an agent")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(setupAddAgentResponse{ID: host.ID, Name: host.Name, Endpoint: endpoint}); err != nil {
		log.Error().Err(err).Msg("error encoding agent")
	}
}

// agentDialLimiter allows a burst of five adds, then one every six seconds.
var agentDialLimiter = rate.NewLimiter(rate.Every(6*time.Second), 5)

// dialFailure turns a dial error into one of a few plain reasons. The raw text
// can quote whatever answered on that port, which says more about the network
// than someone adding an agent needs to know.
func dialFailure(err error) string {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "certificate"):
		return "the agent refused this Dozzle's certificate"
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "server misbehaving"):
		return "no such host"
	case strings.Contains(msg, "connection refused"):
		return "connection refused"
	case strings.Contains(msg, "deadline exceeded"), strings.Contains(msg, "timeout"), strings.Contains(msg, "timed out"):
		return "timed out"
	default:
		return "no Dozzle agent answered at that address"
	}
}

type setupRemoveAgentRequest struct {
	Endpoint string `json:"endpoint"`
}

func (h *handler) removeSetupAgent(w http.ResponseWriter, r *http.Request) {
	if !h.setupCanWrite(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	service, ok := h.agentService()
	if !ok {
		http.Error(w, "agents cannot be removed in this mode", http.StatusConflict)
		return
	}

	var req setupRemoveAgentRequest
	if !decodeSetupBody(w, r, &req) {
		return
	}
	if slices.Contains(h.config.Setup.EnvAgents, req.Endpoint) {
		http.Error(w, "agent is set by flag or env", http.StatusConflict)
		return
	}

	found := false
	if err := config.Update(setupConfigPath, func(c *config.File) {
		c.RemoteAgents = slices.DeleteFunc(c.RemoteAgents, func(e string) bool {
			if e == req.Endpoint {
				found = true
				return true
			}
			return false
		})
		c.PrivateAgents = slices.DeleteFunc(c.PrivateAgents, func(e string) bool { return e == req.Endpoint })
	}); err != nil {
		log.Error().Err(err).Msg("could not update dozzle.yml")
		http.Error(w, "could not write dozzle.yml", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	if err := service.RemoveAgent(req.Endpoint); err != nil && !errors.Is(err, hostservice.ErrAgentNotFound) {
		log.Warn().Err(err).Str("endpoint", req.Endpoint).Msg("could not disconnect agent")
	}
	log.Info().Str("endpoint", req.Endpoint).Msg("setup removed an agent")
	w.WriteHeader(http.StatusNoContent)
}

type setupAgentCertResponse struct {
	Cert     string `json:"cert"`
	Key      string `json:"key"`
	NotAfter string `json:"notAfter"`
}

// agentCert returns the hub's private pair for agents added from the UI,
// making it on first use. It is only ever given to new agents, so turning it
// on never changes anything for agents that already connect.
//
// POST because the first call writes files, and because the answer contains a
// private key: it goes to the same people who may add a host, never into GET
// /api/setup where every page load would carry it.
func (h *handler) agentCert(w http.ResponseWriter, r *http.Request) {
	if !h.setupCanWrite(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	if _, ok := h.agentService(); !ok {
		http.Error(w, "agents cannot be added in this mode", http.StatusConflict)
		return
	}
	if h.config.Setup.CustomCert {
		// Agents of a hub with its own pair need that pair, not a second one.
		http.Error(w, "this hub already uses a custom certificate", http.StatusConflict)
		return
	}
	if !setupPersisted() {
		http.Error(w, "data directory is not persisted", http.StatusPreconditionFailed)
		return
	}

	pair, err := agentcerts.LoadOrCreateAgentPair(setupDataDir())
	if err != nil {
		log.Error().Err(err).Msg("could not create the private agent certificate")
		http.Error(w, "could not create the private certificate", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(setupAgentCertResponse{
		Cert:     string(pair.Cert),
		Key:      string(pair.Key),
		NotAfter: pair.NotAfter.Format("2006-01-02"),
	}); err != nil {
		log.Error().Err(err).Msg("error encoding agent certificate")
	}
}
