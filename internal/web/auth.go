package web

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

func (h *handler) createToken(w http.ResponseWriter, r *http.Request) {
	user := r.PostFormValue("username")
	pass := r.PostFormValue("password")

	if token, err := h.config.Authorization.Authorizer.CreateToken(user, pass); err == nil {
		expires := time.Time{}
		if h.config.Authorization.TTL > 0 {
			expires = time.Now().Add(h.config.Authorization.TTL)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "jwt",
			Value:    token,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Expires:  expires,
		})
		log.Info().Str("user", user).Msg("Token created")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	} else {
		log.Error().Err(err).Msg("Failed to create token")
		http.Error(w, err.Error(), http.StatusUnauthorized)
	}
}

func (h *handler) deleteToken(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
