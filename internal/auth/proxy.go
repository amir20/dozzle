package auth

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type contextKey string

const remoteUser contextKey = "remoteUser"

type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar,omitempty"`
}

func hashEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	hash := md5.Sum([]byte(email))

	return hex.EncodeToString(hash[:])
}

func newUser(username, email, name string) *User {
	avatar := ""
	if email != "" {
		avatar = "https://gravatar.com/avatar/" + hashEmail(email)
	}
	return &User{
		Username: username,
		Email:    email,
		Name:     name,
		Avatar:   avatar,
	}
}

func ForwardProxyAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Remote-Email") == "" {
			log.Error("Unable to find remote email. Please check your proxy configuration. Expecting headers Remote-Email, Remote-User, Remote-Name")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		user := newUser(r.Header.Get("Remote-User"), r.Header.Get("Remote-Email"), r.Header.Get("Remote-Name"))

		ctx := context.WithValue(r.Context(), remoteUser, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RemoteUserFromContext(ctx context.Context) *User {
	user, ok := ctx.Value(remoteUser).(*User)
	if !ok {
		return nil
	}
	return user
}
