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

type proxyAuthContext struct {
	headerUser  string
	headerEmail string
	headerName  string
}

func hashEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	hash := md5.Sum([]byte(email))

	return hex.EncodeToString(hash[:])
}

func NewForwardProxyAuth(user, email, name string) *proxyAuthContext {
	return &proxyAuthContext{
		headerUser:  user,
		headerEmail: email,
		headerName:  name,
	}
}

func (p *proxyAuthContext) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(p.headerUser) != "" {
			user := newUser(r.Header.Get(p.headerUser), r.Header.Get(p.headerEmail), r.Header.Get(p.headerName))
			ctx := context.WithValue(r.Context(), remoteUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func (p *proxyAuthContext) CreateToken(username, password string) (string, error) {
	log.Fatalf("CreateToken not implemented for proxy auth")
	return "", nil
}
