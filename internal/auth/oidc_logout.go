package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/utils"
	"github.com/go-chi/jwtauth/v5"
	"github.com/rs/zerolog/log"
)

// idTokenRetention is how long a stored ID token is kept when --auth-ttl is
// session and the cookie carries no expiry of its own. Most people never click
// logout, so something has to bound what piles up.
const idTokenRetention = 30 * 24 * time.Hour

// idTokenStore keeps the raw ID token of each session on disk, so logout can
// hand it back to the issuer as id_token_hint.
//
// It cannot ride in the session cookie: the token is signed by the issuer and
// has to go back byte for byte, and some issuers put the avatar itself in it as
// a data: URL, several KB past what a browser keeps in a cookie. The cookie
// carries only the session id that names the file.
//
// Everything here is best effort. A token that was never written, or is gone
// after the container was recreated without a volume, costs the issuer's
// confirmation prompt at logout and nothing else.
type idTokenStore struct {
	// dir is the data directory. Empty keeps nothing, and every logout asks.
	dir       string
	retention time.Duration
}

func newIDTokenStore(dir string, ttl time.Duration) idTokenStore {
	retention := idTokenRetention
	if ttl > 0 {
		retention = ttl
	}

	return idTokenStore{dir: dir, retention: retention}
}

// path is <data>/<profile key>/sessions/<session id>. Both parts come out of a
// session JWT Dozzle signed, but are checked anyway since they name a file.
func (s idTokenStore) path(user, session string) (string, bool) {
	if s.dir == "" || !isSafeSegment(user) || !isSessionID(session) {
		return "", false
	}

	return filepath.Join(s.dir, user, "sessions", session), true
}

func (s idTokenStore) save(user, session, token string) error {
	if s.dir == "" {
		return nil
	}

	path, ok := s.path(user, session)
	if !ok {
		return os.ErrInvalid
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	// WriteFile goes through os.Create, which keeps an existing file's mode, so
	// creating it first is what keeps someone's ID token from other local users.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	f.Close()

	return utils.WriteFile(path, func(w io.Writer) error {
		_, err := io.WriteString(w, token)
		return err
	})
}

// take reads the token for a session and deletes it: a logout is the last time
// it is needed.
func (s idTokenStore) take(user, session string) string {
	path, ok := s.path(user, session)
	if !ok {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Debug().Err(err).Str("user", user).Msg("No stored ID token for this session; the issuer will ask to confirm logout")
		return ""
	}

	if err := os.Remove(path); err != nil {
		log.Debug().Err(err).Str("path", path).Msg("Could not remove the stored ID token")
	}

	return string(data)
}

// prune drops a user's tokens older than the retention, run at each login of
// that user so abandoned sessions do not accumulate.
func (s idTokenStore) prune(user string) {
	if s.dir == "" || !isSafeSegment(user) {
		return
	}

	dir := filepath.Join(s.dir, user, "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-s.retention)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			log.Debug().Err(err).Str("file", entry.Name()).Msg("Could not prune a stored ID token")
		}
	}
}

func newSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func isSessionID(id string) bool {
	if len(id) != 32 {
		return false
	}
	_, err := hex.DecodeString(id)

	return err == nil
}

func isSafeSegment(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name
}

// LogoutTarget is where the browser goes once Dozzle's cookie is cleared. With
// Params it is submitted as a form POST: an ID token can run past the 8KB
// request line a reverse proxy in front of the issuer accepts (nginx's
// default), and the spec requires the end_session_endpoint to take POST too.
type LogoutTarget struct {
	URL    string            `json:"url"`
	Params map[string]string `json:"params,omitempty"`
}

// LogoutRedirect forgets the session's stored ID token and returns where the
// browser goes next, or nil to just reload.
//
// When the issuer publishes an end_session_endpoint, that is OpenID Connect
// RP-Initiated Logout: with id_token_hint the issuer signs the user out without
// asking, and post_logout_redirect_uri brings them back to Dozzle's login page.
//
// --auth-logout-url still wins when it points somewhere else, for an issuer
// whose logout lives outside discovery. Set to the end_session_endpoint itself,
// which is what the docs used to recommend, it is treated as unset so those
// deployments pick up the hint without a config change.
func (a *oidcAuthContext) LogoutRedirect(r *http.Request) *LogoutTarget {
	idToken := ""
	// Only a browser session names the stored ID token; an MCP token presented
	// here must not consume the one its user's browser still needs.
	if _, claims, err := jwtauth.FromContext(r.Context()); err == nil && isSessionClaims(claims) {
		sub, _ := claims["sub"].(string)
		session, _ := claims["session"].(string)
		if sub != "" && session != "" {
			idToken = a.idTokens.take(profileKey(sub), session)
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	endSession := ""
	if discovery, err := a.provider.discover(ctx); err == nil {
		endSession = discovery.EndSessionURL
	} else {
		log.Warn().Err(err).Msg("Could not reach the OIDC issuer for logout; only the Dozzle session was cleared")
	}

	if a.logoutURL != "" && !sameEndpoint(a.logoutURL, endSession) {
		return &LogoutTarget{URL: a.logoutURL}
	}

	if endSession == "" {
		return nil
	}

	params := map[string]string{
		"client_id":                a.provider.clientID,
		"post_logout_redirect_uri": a.absoluteURL(r, "/login"),
	}
	if idToken != "" {
		params["id_token_hint"] = idToken
	}

	return &LogoutTarget{URL: endSession, Params: params}
}

// sameEndpoint compares two URLs ignoring their query, so a configured logout
// URL that already carries parameters still matches discovery.
func sameEndpoint(a, b string) bool {
	if a == "" || b == "" {
		return false
	}

	pa, errA := url.Parse(a)
	pb, errB := url.Parse(b)
	if errA != nil || errB != nil {
		return false
	}

	return strings.EqualFold(pa.Scheme, pb.Scheme) &&
		strings.EqualFold(pa.Host, pb.Host) &&
		strings.TrimSuffix(pa.Path, "/") == strings.TrimSuffix(pb.Path, "/")
}
