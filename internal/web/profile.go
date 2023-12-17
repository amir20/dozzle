package web

import (
	"io"
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/profile"
	log "github.com/sirupsen/logrus"
)

func (h *handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unable to find user", http.StatusInternalServerError)
		return
	}

	if err := profile.UpdateFromReader(*user, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Errorf("Unable to save user settings: %s", err)
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

	log.Debugf("Fetching avatar from %s", url)
	response, err := http.Get(url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Errorf("Received status code %d from %s", response.StatusCode, url)
		return
	}

	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	io.Copy(w, response.Body)
}
