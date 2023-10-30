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

type proxyAuthAuth struct {
}

func hashEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	hash := md5.Sum([]byte(email))

	return hex.EncodeToString(hash[:])
}

func NewForwardProxyAuth() *proxyAuthAuth {
	return &proxyAuthAuth{}
}

func (p *proxyAuthAuth) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Remote-Email") != "" {
			user := newUser(r.Header.Get("Remote-User"), r.Header.Get("Remote-Email"), r.Header.Get("Remote-Name"))
			ctx := context.WithValue(r.Context(), remoteUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func (p *proxyAuthAuth) CreateToken(username, password string) (string, error) {
	log.Fatalf("CreateToken not implemented for proxy auth")
	return "", nil
}
