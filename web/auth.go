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
	if h.config.Username != "" && h.config.Password != "" {
		store = sessions.NewCookieStore([]byte(h.config.Key))
		store.Options.HttpOnly = true
		store.Options.SameSite = http.SameSiteLaxMode
		secured = true
	}
}

func authorizationRequired(f http.HandlerFunc) http.Handler {
	if secured {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, _ := store.Get(r, sessionName)
			if session.IsNew {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			} else {
				f(w, r)
			}
		})
	} else {
		return http.HandlerFunc(f)
	}
}

func (h *handler) isAuthorized(r *http.Request) bool {
	if !secured {
		return true
	}

	session, _ := store.Get(r, sessionName)
	if session.IsNew {
		return false
	}

	if val, ok := session.Values[authorityKey]; ok {
		println(val)
		// TODO check for timestamp
		return true
	}

	return false
}

func (h *handler) isAuthorizationNeeded(r *http.Request) bool {
	return secured && !h.isAuthorized(r)
}

func (h *handler) validateCredentials(w http.ResponseWriter, r *http.Request) {
	if !secured {
		log.Panic("Validating credentials with secured=false should not happen")
	}

	if r.Method != "POST" {
		log.Fatal("Expecting meethod to be POST")
		http.Error(w, http.StatusText(http.StatusNotAcceptable), http.StatusNotAcceptable)
		return
	}

	r.ParseForm()

	user := r.Form["username"][0]
	pass := r.Form["password"][0]
	session, _ := store.Get(r, sessionName)

	if user == h.config.Username && pass == h.config.Password {
		session.Values[authorityKey] = time.Now()
		session.Save(r, w)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
		return
	}

	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
