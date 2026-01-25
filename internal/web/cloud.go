package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

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

	name := "Dozzle Cloud"

	cloudDispatcher, err := dispatcher.NewCloudDispatcher(name, tokenResp.Key)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create cloud dispatcher")
		http.Error(w, "failed to create cloud dispatcher", http.StatusInternalServerError)
		return
	}

	id := h.hostService.AddDispatcher(cloudDispatcher)

	base := h.config.Base
	if base == "/" {
		base = ""
	}
	redirectURL := fmt.Sprintf("%s/notifications?newCloudLink=%d", base, id)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
