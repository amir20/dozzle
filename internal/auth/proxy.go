package auth

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

type contextKey string

const remoteUser contextKey = "remoteUser"

// WithUser returns a copy of ctx carrying user so downstream handlers can
// resolve it via UserFromContext.
func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, remoteUser, user)
}

type proxyAuthContext struct {
	headerUser   string
	headerEmail  string
	headerName   string
	headerFilter string
	headerRoles  string
}

func hashEmail(email string) string {
	email = strings.TrimSpace(email)
	email = strings.ToLower(email)
	hash := md5.Sum([]byte(email))

	return hex.EncodeToString(hash[:])
}

func NewForwardProxyAuth(userHeader, emailHeader, nameHeader, filterHeader, rolesHeader string) *proxyAuthContext {
	return &proxyAuthContext{
		headerUser:   userHeader,
		headerEmail:  emailHeader,
		headerName:   nameHeader,
		headerFilter: filterHeader,
		headerRoles:  rolesHeader,
	}
}

func (p *proxyAuthContext) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(p.headerUser) != "" {
			containerFilter, err := container.ParseContainerFilter(r.Header.Get(p.headerFilter))
			if err != nil {
				log.Warn().Err(err).Str("filter", r.Header.Get(p.headerFilter)).Msg("Failed to parse container filter")
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			userRoles := All
			if strings.TrimSpace(r.Header.Get(p.headerRoles)) != "" {
				userRoles = ParseRole(r.Header.Get(p.headerRoles))
			}
			user := newUser(r.Header.Get(p.headerUser), r.Header.Get(p.headerEmail), r.Header.Get(p.headerName), containerFilter, userRoles)
			ctx := WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

func (p *proxyAuthContext) CreateToken(username, password string) (string, error) {
	log.Fatal().Msg("CreateToken not implemented in proxy auth")
	return "", nil
}
