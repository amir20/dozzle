package web

import (
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	log "github.com/sirupsen/logrus"
)

var secured = false
var store *sessions.CookieStore

const authorityKey = "AUTH_TIMESTAMP"
const sessionName = "session"

func initializeAuth(h *handler) {
	secured = false
	if h.config.Username != "" && h.config.Password != "" {
		store = sessions.NewCookieStore([]byte(h.config.Key))
		store.Options.HttpOnly = true
		store.Options.SameSite = http.SameSiteLaxMode
		store.Options.MaxAge = 0
		secured = true
	}
}

func authorizationRequired(f http.HandlerFunc) http.Handler {
	if secured {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAuthorized(r) {
				f(w, r)
			} else {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			}
		})
	} else {
		return http.HandlerFunc(f)
	}
}

func isAuthorized(r *http.Request) bool {
	if !secured {
		return true
	}

	session, _ := store.Get(r, sessionName)
	if session.IsNew {
		return false
	}

	if _, ok := session.Values[authorityKey]; ok {
		// TODO check for timestamp
		return true
	}

	return false
}

func (h *handler) isAuthorizationNeeded(r *http.Request) bool {
	return secured && !isAuthorized(r)
}

func (h *handler) validateCredentials(w http.ResponseWriter, r *http.Request) {
	if !secured {
		log.Panic("Validating credentials without username and password should not happen")
	}

	if r.Method != "POST" {
		log.Fatal("Expecting credential validation method to be POST")
		http.Error(w, http.StatusText(http.StatusNotAcceptable), http.StatusNotAcceptable)
		return
	}

	if err := r.ParseMultipartForm(4 * 1024); err != nil {
		log.Fatalf("Error while parsing form data: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user := r.PostFormValue("username")
	pass := r.PostFormValue("password")
	session, _ := store.Get(r, sessionName)

	if user == h.config.Username && pass == h.config.Password {
		session.Values[authorityKey] = time.Now().Unix()

		if err := session.Save(r, w); err != nil {
			log.Fatalf("Error while parsing saving session: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
		return
	}

	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

func (h *handler) clearSession(w http.ResponseWriter, r *http.Request) {
	if !secured {
		log.Panic("Validating credentials with secured=false should not happen")
	}

	session, _ := store.Get(r, sessionName)
	delete(session.Values, authorityKey)

	if err := session.Save(r, w); err != nil {
		log.Fatalf("Error while parsing saving session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, h.config.Base, http.StatusTemporaryRedirect)
}
