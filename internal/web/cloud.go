package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/rs/zerolog/log"
)

type exchangeTokenResponse struct {
	Key       string  `json:"key"`
	Prefix    string  `json:"prefix"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

func (h *handler) cloudCallback(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	from := r.URL.Query().Get("from")
	if token == "" {
		http.Error(w, "missing token parameter", http.StatusBadRequest)
		return
	}

	cloudURL := os.Getenv("DOLIGENCE_URL")
	if cloudURL == "" {
		cloudURL = "https://doligence.dozzle.dev"
	}

	exchangeURL := fmt.Sprintf("%s/api/exchange-token", cloudURL)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, exchangeURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create request")
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", dispatcher.UserAgent)
	q := req.URL.Query()
	q.Set("token", token)
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to exchange token")
		http.Error(w, "failed to exchange token", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		log.Error().Int("status", resp.StatusCode).Str("body", string(body)).Msg("Token exchange failed")
		http.Error(w, "token exchange failed", http.StatusInternalServerError)
		return
	}

	var tokenResp exchangeTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Error().Err(err).Msg("Failed to decode token response")
		http.Error(w, "failed to decode token response", http.StatusInternalServerError)
		return
	}

	if tokenResp.Key == "" {
		log.Error().Msg("Empty key received")
		http.Error(w, "empty key received", http.StatusInternalServerError)
		return
	}

	var expiresAt *time.Time
	if tokenResp.ExpiresAt != nil {
		parsed, err := time.Parse(time.RFC3339, *tokenResp.ExpiresAt)
		if err != nil {
			log.Warn().Err(err).Str("expiresAt", *tokenResp.ExpiresAt).Msg("Failed to parse expiresAt, ignoring")
		} else {
			expiresAt = &parsed
		}
	}

	// Save cloud config (also creates the cloud dispatcher and broadcasts to agents)
	cc := &notification.CloudConfig{
		APIKey:    tokenResp.Key,
		Prefix:    tokenResp.Prefix,
		ExpiresAt: expiresAt,
	}
	h.hostService.SetCloudConfig(cc)

	if h.config.OnCloudSetup != nil {
		h.config.OnCloudSetup()
	}

	base := h.config.Base
	if base == "/" {
		base = ""
	}

	var redirectURL string
	if from == "notifications" {
		redirectURL = fmt.Sprintf("%s/notifications#cloudLinked", base)
	} else {
		redirectURL = fmt.Sprintf("%s/#cloudLinked", base)
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *handler) cloudStatus(w http.ResponseWriter, r *http.Request) {
	cc := h.hostService.CloudConfig()
	if cc == nil || cc.APIKey == "" {
		writeError(w, http.StatusNotFound, "no cloud configuration")
		return
	}
	apiKey := cc.APIKey

	cloudURL := os.Getenv("DOLIGENCE_URL")
	if cloudURL == "" {
		cloudURL = "https://doligence.dozzle.dev"
	}

	statusURL := fmt.Sprintf("%s/api/status", cloudURL)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, statusURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create cloud status request")
		writeError(w, http.StatusInternalServerError, "failed to create request")
		return
	}
	req.Header.Set("User-Agent", dispatcher.UserAgent)
	req.Header.Set("X-API-Key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch cloud status")
		writeError(w, http.StatusBadGateway, "failed to fetch cloud status")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		log.Warn().Int("status", resp.StatusCode).Str("body", string(body)).Msg("Cloud status check failed")
		writeError(w, resp.StatusCode, "cloud API key is invalid or expired")
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read cloud status response")
		writeError(w, http.StatusBadGateway, "failed to read cloud status")
		return
	}

	var statusResp struct {
		Plan struct {
			Name string `json:"name"`
		} `json:"plan"`
	}
	if json.Unmarshal(body, &statusResp) == nil && statusResp.Plan.Name == "pro" {
		if h.config.OnCloudSetup != nil {
			h.config.OnCloudSetup()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

type cloudConfigResponse struct {
	Prefix    string  `json:"prefix"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
	Linked    bool    `json:"linked"`
}

func (h *handler) cloudConfig(w http.ResponseWriter, r *http.Request) {
	cc := h.hostService.CloudConfig()
	if cc == nil {
		writeError(w, http.StatusNotFound, "no cloud configuration")
		return
	}

	resp := cloudConfigResponse{
		Prefix: cc.Prefix,
		Linked: true,
	}
	if cc.ExpiresAt != nil {
		s := cc.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &s
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *handler) deleteCloudConfig(w http.ResponseWriter, r *http.Request) {
	h.hostService.RemoveCloudConfig()
	w.WriteHeader(http.StatusNoContent)
}
