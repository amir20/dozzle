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
	"github.com/amir20/dozzle/internal/analytics"
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

// agentErrorHeader carries a stable code for an agent error, so the UI can
// explain it without matching the English text in the body.
const agentErrorHeader = "X-Dozzle-Error"

func agentError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set(agentErrorHeader, code)
	http.Error(w, msg, status)
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
			Private: !locked && slices.ContainsFunc(file.PrivateAgents, sameEndpoint(endpoint))}
		if service != nil {
			a.HostID = service.AgentHostID(endpoint)
		}
		agents = append(agents, a)
	}
	for _, endpoint := range h.config.Setup.EnvAgents {
		add(endpoint, true)
	}
	// Only a mode that can add agents dials the ones in dozzle.yml (swarm keeps to
	// the operator's), so elsewhere listing them would show hosts that never connect.
	if service != nil {
		for _, endpoint := range file.RemoteAgents {
			// Trimmed the way startup reads them, so a hand-edited entry still
			// matches the agent that was dialed under it.
			if endpoint = strings.TrimSpace(endpoint); endpoint != "" {
				add(endpoint, false)
			}
		}
	}
	return agents
}

// sameEndpoint matches a dozzle.yml entry to endpoint, ignoring the whitespace a
// hand edit can leave around it.
func sameEndpoint(endpoint string) func(string) bool {
	return func(e string) bool { return strings.TrimSpace(e) == endpoint }
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
	if outcome := h.addSetupAgentOnce(w, r); outcome != "" {
		analytics.Count(outcome)
	}
}

// addSetupAgentOnce answers the request and returns the usage counter for it, or
// "" when it never got as far as trying: no permission, or a body that did not
// decode.
func (h *handler) addSetupAgentOnce(w http.ResponseWriter, r *http.Request) string {
	if !h.setupCanWrite(r) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return ""
	}
	service, ok := h.agentService()
	if !ok {
		agentError(w, http.StatusConflict, "unsupported-mode", "agents cannot be added in this mode")
		return "host.add.other"
	}
	if !setupPersisted() {
		agentError(w, http.StatusPreconditionFailed, "not-persisted", "data directory is not persisted")
		return "host.add.other"
	}

	var req setupAddAgentRequest
	if !decodeSetupBody(w, r, &req) {
		return ""
	}
	address, name := strings.TrimSpace(req.Address), strings.TrimSpace(req.Name)
	endpoint, err := setupAgentAddress(address, name)
	if err != nil {
		agentError(w, http.StatusBadRequest, "invalid", err.Error())
		return "host.add.other"
	}

	file, err := config.Load(setupConfigPath)
	if err != nil {
		log.Error().Err(err).Msg("could not read setup config")
		http.Error(w, "could not read dozzle.yml", http.StatusInternalServerError)
		return "host.add.other"
	}
	if slices.ContainsFunc(h.setupAgents(file), func(a setupAgent) bool { return a.Address == address }) {
		agentError(w, http.StatusConflict, "exists", "this agent is already added")
		return "host.add.duplicate"
	}

	var cert *tls.Certificate
	if req.Private {
		pair, err := agentcerts.LoadAgentPair(setupDataDir())
		if err != nil {
			agentError(w, http.StatusPreconditionFailed, "no-private-cert", "no private certificate yet, create one first")
			return "host.add.other"
		}
		parsed, err := pair.TLS()
		if err != nil {
			log.Error().Err(err).Msg("could not parse the private agent certificate")
			http.Error(w, "could not read the private certificate", http.StatusInternalServerError)
			return "host.add.other"
		}
		cert = &parsed
	}

	// Each add dials an address the caller picked. With no login, anyone who can
	// reach the page can do that during the setup window, so dials are rationed
	// and the answer names only the kind of failure, never the raw error text.
	if !agentDialLimiter.Allow() {
		agentError(w, http.StatusTooManyRequests, "rate-limited", "too many attempts, wait a minute and try again")
		return "host.add.other"
	}

	host, err := service.AddAgent(r.Context(), endpoint, cert)
	switch {
	case errors.Is(err, hostservice.ErrAgentExists):
		agentError(w, http.StatusConflict, "exists", "this agent is already added")
		return "host.add.duplicate"
	case errors.Is(err, hostservice.ErrDuplicateHost):
		agentError(w, http.StatusConflict, "duplicate-host", "this agent is already connected under another address")
		return "host.add.duplicate"
	case err != nil:
		log.Debug().Err(err).Str("endpoint", endpoint).Msg("setup could not connect to agent")
		reason, outcome := dialFailure(err)
		code := "unreachable"
		if outcome == "host.add.cert" {
			code = "cert-mismatch"
		}
		agentError(w, http.StatusBadGateway, code, "could not connect to agent: "+reason)
		return outcome
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
		return "host.add.other"
	}

	log.Info().Str("endpoint", endpoint).Str("host", host.Name).Msg("setup added an agent")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(setupAddAgentResponse{ID: host.ID, Name: host.Name, Endpoint: endpoint}); err != nil {
		log.Error().Err(err).Msg("error encoding agent")
	}
	return "host.add.ok"
}

// agentDialLimiter allows a burst of five adds, then one every six seconds.
var agentDialLimiter = rate.NewLimiter(rate.Every(6*time.Second), 5)

// dialFailure turns a dial error into one of a few plain reasons, plus the usage
// counter for it. The raw text can quote whatever answered on that port, which
// says more about the network than someone adding an agent needs to know.
func dialFailure(err error) (reason, outcome string) {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "certificate"):
		return "the agent refused this Dozzle's certificate", "host.add.cert"
	case strings.Contains(msg, "no such host"), strings.Contains(msg, "server misbehaving"):
		return "no such host", "host.add.refused"
	case strings.Contains(msg, "connection refused"):
		return "connection refused", "host.add.refused"
	case strings.Contains(msg, "deadline exceeded"), strings.Contains(msg, "timeout"), strings.Contains(msg, "timed out"):
		return "timed out", "host.add.timeout"
	default:
		return "no Dozzle agent answered at that address", "host.add.refused"
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
		agentError(w, http.StatusConflict, "unsupported-mode", "agents cannot be removed in this mode")
		return
	}

	var req setupRemoveAgentRequest
	if !decodeSetupBody(w, r, &req) {
		return
	}
	if slices.Contains(h.config.Setup.EnvAgents, req.Endpoint) {
		agentError(w, http.StatusConflict, "env-agent", "agent is set by flag or env")
		return
	}

	found := false
	if err := config.Update(setupConfigPath, func(c *config.File) {
		matches := sameEndpoint(req.Endpoint)
		c.RemoteAgents = slices.DeleteFunc(c.RemoteAgents, func(e string) bool {
			if matches(e) {
				found = true
				return true
			}
			return false
		})
		c.PrivateAgents = slices.DeleteFunc(c.PrivateAgents, matches)
	}); err != nil {
		log.Error().Err(err).Msg("could not update dozzle.yml")
		http.Error(w, "could not write dozzle.yml", http.StatusInternalServerError)
		return
	}
	if !found {
		agentError(w, http.StatusNotFound, "not-found", "agent not found")
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
		agentError(w, http.StatusConflict, "unsupported-mode", "agents cannot be added in this mode")
		return
	}
	if h.config.Setup.CustomCert {
		// Agents of a hub with its own pair need that pair, not a second one.
		agentError(w, http.StatusConflict, "custom-cert", "this hub already uses a custom certificate")
		return
	}
	if !setupPersisted() {
		agentError(w, http.StatusPreconditionFailed, "not-persisted", "data directory is not persisted")
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
