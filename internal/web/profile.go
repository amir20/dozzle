package web

import (
	"encoding/json"
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/profile"
	log "github.com/sirupsen/logrus"
)

func (h *handler) saveSettings(w http.ResponseWriter, r *http.Request) {
	var settings profile.Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unable to find user", http.StatusInternalServerError)
		return
	}

	if err := profile.SaveUserSettings(*user, settings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Errorf("Unable to save user settings: %s", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
