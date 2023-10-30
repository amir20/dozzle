package web

import (
	"crypto/sha256"
	"fmt"
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
		store = sessions.NewCookieStore(generateSessionStorageKey(h.config.Username, h.config.Password))
		store.Options.HttpOnly = true
		store.Options.SameSite = http.SameSiteLaxMode
		store.Options.MaxAge = 0
		secured = true
	}
}

func authorizationRequired(next http.Handler) http.Handler {
	if secured {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAuthorized(r) {
				next.ServeHTTP(w, r)
			} else {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			}
		})
	} else {
		return next
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

func (h *handler) createToken(w http.ResponseWriter, r *http.Request) {
	user := r.PostFormValue("username")
	pass := r.PostFormValue("password")

	if token, err := h.config.Authorizer.CreateToken(user, pass); err == nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "jwt",
			Value:    token,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})
		log.Infof("Token created for user %s", user)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	} else {
		log.Errorf("Error while creating token: %v", err)
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

func generateSessionStorageKey(username string, password string) []byte {
	key := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", username, password)))
	return key[:]
}
