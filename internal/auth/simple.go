package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"maps"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/rs/zerolog/log"
)

type simpleAuthContext struct {
	UserDatabase UserDatabase
	tokenAuth    *jwtauth.JWTAuth
	ttl          time.Duration
	// UserDatabase.Find reloads users.yml in place, and the middleware now calls it
	// on every request, so the reload has to be serialized.
	mu sync.Mutex
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func NewSimpleAuth(userDatabase UserDatabase, ttl time.Duration) *simpleAuthContext {
	// Hash the users in a stable order. Ranging over the map directly makes the
	// digest depend on Go's randomized map iteration order, so any users.yml with
	// more than one user derives a different signing key on every start and
	// silently invalidates every session on restart.
	//
	// The linked OAuth accounts are part of the digest for the same reason the
	// password is: repointing a user at a different GitHub login has to rotate
	// sessions, or the sessions minted under the old link outlive it.
	h := sha256.New()
	for _, username := range slices.Sorted(maps.Keys(userDatabase.Users)) {
		user := userDatabase.Users[username]
		h.Write([]byte(user.Password))
		h.Write([]byte(user.RolesConfigured))
		h.Write([]byte(user.Email))
		h.Write([]byte(user.Github))
	}

	tokenAuth := jwtauth.New("HS256", h.Sum(nil), nil)

	return &simpleAuthContext{
		UserDatabase: userDatabase,
		tokenAuth:    tokenAuth,
		ttl:          ttl,
	}
}

// find returns the user by value. UserDatabase.Find hands back a pointer into
// UserDatabase.Users and reloads users.yml as it goes, so dereferencing it under
// the lock keeps a reload from racing whoever is reading the user.
func (a *simpleAuthContext) find(username string) (User, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	user := a.UserDatabase.Find(username)
	if user == nil {
		return User{}, false
	}

	return *user, true
}

// PasswordLoginEnabled reports whether the password form can succeed for anyone.
// When it cannot, the login page drops the username and password fields instead
// of offering a form with no possible answer.
func (a *simpleAuthContext) PasswordLoginEnabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.UserDatabase.AnyPassword()
}

// findByGithub resolves a GitHub login to a user, by value and under the same
// lock as find, so the users.yml reload cannot race the read.
func (a *simpleAuthContext) findByGithub(login string) (User, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	user := a.UserDatabase.FindByGithub(login)
	if user == nil {
		return User{}, false
	}

	return *user, true
}

// findByEmail is findByGithub for a verified email address.
func (a *simpleAuthContext) findByEmail(email string) (User, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	user := a.UserDatabase.FindByEmail(email)
	if user == nil {
		return User{}, false
	}

	return *user, true
}

func (a *simpleAuthContext) CreateToken(username, password string) (string, error) {
	user, ok := a.find(username)
	// An account linked only to an OAuth provider carries no hash to compare,
	// and CompareHashAndPassword would log a bogus "invalid hash length" for
	// every attempt against one.
	if !ok || user.Password == "" || !CompareHashAndPassword(user.Password, password) {
		return "", ErrInvalidCredentials
	}

	return a.issueToken(user)
}

// issueToken mints the session JWT. Password login and the OAuth callback both
// land here, so a session is indistinguishable however the user proved who they
// are, and everything downstream of the token stays untouched.
func (a *simpleAuthContext) issueToken(user User) (string, error) {
	// Identity only. Everything else about the user is read from users.yml per
	// request, so anything baked in here would just be a copy that goes stale.
	claims := map[string]any{"username": user.Username}
	jwtauth.SetIssuedNow(claims)

	if a.ttl > 0 {
		jwtauth.SetExpiryIn(claims, a.ttl)
	}

	_, tokenString, err := a.tokenAuth.Encode(claims)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// TTL is the configured session lifetime, exposed so the OAuth callback can set
// the same cookie expiry the password path does.
func (a *simpleAuthContext) TTL() time.Duration {
	return a.ttl
}

func (a *simpleAuthContext) AuthMiddleware(next http.Handler) http.Handler {
	return jwtauth.Verifier(a.tokenAuth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The roles claim is a bitmask frozen at login, so it goes stale the moment
		// the set of roles grows: a session minted before the cloud role existed
		// carries a mask without that bit and quietly loses the feature until the
		// user happens to log out. The same is true of a roles or filter edit in
		// users.yml. Resolve both from the database per request and let the token
		// prove only who the user is.
		if user := a.userFromToken(r.Context()); user != nil {
			r = r.WithContext(WithUser(r.Context(), *user))
		}

		next.ServeHTTP(w, r)
	}))
}

// userFromToken resolves the verified token's subject against users.yml. It returns
// nil for a missing or invalid token, and for a user who is no longer configured, so
// the request falls through to RequireAuthentication as unauthenticated.
func (a *simpleAuthContext) userFromToken(ctx context.Context) *User {
	_, claims, err := jwtauth.FromContext(ctx)
	if err != nil {
		return nil
	}

	username, ok := claims["username"].(string)
	if !ok || username == "" {
		return nil
	}

	user, ok := a.find(username)
	if !ok {
		log.Debug().Str("username", username).Msg("Token is valid but user is no longer in the user database")
		return nil
	}

	user.Password = ""

	return &user
}
