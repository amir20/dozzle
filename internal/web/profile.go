package web

import (
	"io"
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/profile"
	"github.com/rs/zerolog/log"
)

func (h *handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unable to find user", http.StatusInternalServerError)
		return
	}

	if err := profile.UpdateFromReader(*user, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to update profile")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *handler) avatar(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unable to find user", http.StatusInternalServerError)
		return
	}

	url := user.AvatarURL()

	if url == "" {
		http.Error(w, "Unable to find avatar", http.StatusNotFound)
		return
	}

	log.Trace().Str("url", url).Msg("Fetching avatar")
	response, err := http.Get(url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Error().Str("url", url).Int("status", response.StatusCode).Msg("Failed to fetch avatar")
		return
	}

	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	io.Copy(w, response.Body)
}
