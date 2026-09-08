package web

import (
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

func (h *handler) deleteToken(w http.ResponseWriter, r *http.Request) {
	auth.ClearSessionCookie(w, r)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
