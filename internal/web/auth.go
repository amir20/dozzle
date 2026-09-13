package web

import (
	"encoding/json"
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/rs/zerolog/log"
)

func (h *handler) createToken(w http.ResponseWriter, r *http.Request) {
	user := r.PostFormValue("username")
	pass := r.PostFormValue("password")

	if token, err := h.config.Authorization.Authorizer.CreateToken(user, pass); err == nil {
		auth.SetSessionCookie(w, r, token, h.config.Authorization.TTL)
		log.Info().Str("user", user).Msg("Token created")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	} else {
		log.Error().Err(err).Msg("Failed to create token")
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}
}

// LogoutRedirector is an Authorizer that sends the browser somewhere once its
// session is cleared, the issuer's end_session_endpoint under oidc.
type LogoutRedirector interface {
	LogoutRedirect(*http.Request) *auth.LogoutTarget
}

func (h *handler) deleteToken(w http.ResponseWriter, r *http.Request) {
	// Read before the cookie is cleared: the session names what to hand back
	// to the issuer.
	var target *auth.LogoutTarget
	if redirector, ok := h.config.Authorization.Authorizer.(LogoutRedirector); ok {
		target = redirector.LogoutRedirect(r)
	}

	auth.ClearSessionCookie(w, r)

	if target != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(target)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
