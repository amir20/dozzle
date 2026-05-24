package web

import (
	"net/http"
	"strings"
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
			Secure:   isHTTPS(r),
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
		Secure:   isHTTPS(r),
		Expires:  time.Unix(0, 0),
	})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

// isHTTPS reports whether the original client request used HTTPS, accounting
// for TLS terminated at an upstream reverse proxy via X-Forwarded-Proto.
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
