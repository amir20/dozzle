package web

import (
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/profile"
	log "github.com/sirupsen/logrus"
)

func (h *handler) saveSettings(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unable to find user", http.StatusInternalServerError)
		return
	}

	if err := profile.SaveUserProfile(*user, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Errorf("Unable to save user settings: %s", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
