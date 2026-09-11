package auth

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// sha256AdvisoryURL explains why sha256 hashes stopped being accepted, and is
// the one thing an operator with a pre-bcrypt users.yml needs to read.
const sha256AdvisoryURL = "https://github.com/amir20/dozzle/security/advisories/GHSA-w7qr-q9fh-fj35"

type User struct {
	Username        string                    `json:"username" yaml:"-"`
	Email           string                    `json:"email" yaml:"email"`
	Name            string                    `json:"name" yaml:"name"`
	Password        string                    `json:"-" yaml:"password"`
	Github          string                    `json:"-" yaml:"github"`
	Filter          string                    `json:"-" yaml:"filter"`
	RolesConfigured string                    `json:"-" yaml:"roles"`
	ContainerLabels container.ContainerLabels `json:"-" yaml:"-"`
	Roles           Role                      `json:"-" yaml:"-"`
}

func (u User) AvatarURL() string {
	name := u.Name
	if name == "" {
		name = u.Username
	}
	return fmt.Sprintf("https://gravatar.com/avatar/%s?d=https%%3A%%2F%%2Fui-avatars.com%%2Fapi%%2F/%s/128", hashEmail(u.Email), url.QueryEscape(name))
}

func newUser(username, email, name string, labels container.ContainerLabels, roles Role) User {
	return User{
		Username:        username,
		Email:           email,
		Name:            name,
		ContainerLabels: labels,
		Roles:           roles,
	}
}

type UserDatabase struct {
	Users    map[string]*User `yaml:"users"`
	LastRead time.Time        `yaml:"-"`
	Path     string           `yaml:"-"`

	// Secondary indexes over Users. An OAuth login arrives holding an external
	// identity rather than a Dozzle username, so it needs a lookup keyed by the
	// linked account. Rebuilt by decodeUsersFromFile so they cannot drift from
	// Users across the mtime reload in readFileIfChanged.
	byEmail  map[string]*User
	byGithub map[string]*User
}

func ReadUsersFromFile(path string) (UserDatabase, error) {
	users, err := decodeUsersFromFile(path)
	if err != nil {
		return users, err
	}

	users.LastRead = time.Now()
	users.Path = path

	return users, nil
}

func GenerateUsers(user User, hashPassword bool) *bytes.Buffer {
	buffer := &bytes.Buffer{}

	if hashPassword {
		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 11)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to hash password")
		}
		user.Password = string(hash)
	}

	users := UserDatabase{
		Users: map[string]*User{
			user.Username: &user,
		},
	}

	yaml.NewEncoder(buffer).Encode(users)

	return buffer
}

func decodeUsersFromFile(path string) (UserDatabase, error) {
	users := UserDatabase{}
	file, err := os.Open(path)
	if err != nil {
		return users, err
	}
	defer file.Close()

	if err := yaml.NewDecoder(file).Decode(&users); err != nil {
		return users, err
	}

	users.byEmail = make(map[string]*User, len(users.Users))
	users.byGithub = make(map[string]*User, len(users.Users))

	for username, user := range users.Users {
		user.Username = username

		// A password is optional now that an account can be proven by OAuth
		// instead, but an entry with no way at all to sign in is a typo, not a
		// configuration.
		if user.Password == "" && user.Github == "" && user.Email == "" {
			return users, fmt.Errorf("user %s has no password, github, or email, so it can never sign in", username)
		}

		// A sha256 hash is 64 characters. It used to be a supported format, so this
		// check used to accept one, but CompareHashAndPassword has refused to compare
		// one since v10.0.1. Accepting it here means the file loads clean, the login
		// page renders a password form, and the refusal lands on the first login
		// attempt instead of at startup. Reject it where main.go can name the file.
		if len(user.Password) == 64 {
			return users, fmt.Errorf("user %s has a sha256 password hash, which is no longer supported: regenerate it with `dozzle generate` to get a bcrypt hash, see %s", username, sha256AdvisoryURL)
		}

		if user.Password != "" && len(user.Password) != 60 {
			return users, fmt.Errorf("user %s has an invalid password hash: expected 60 characters, got %d", username, len(user.Password))
		}

		if user.Name == "" {
			user.Name = username
		}

		if strings.TrimSpace(user.RolesConfigured) == "" {
			user.RolesConfigured = "all"
		}

		user.Roles = ParseRole(user.RolesConfigured)

		labels, err := container.ParseContainerFilter(user.Filter)
		if err != nil {
			return users, fmt.Errorf("user %s has an invalid filter %q: %w", username, user.Filter, err)
		}
		user.ContainerLabels = labels

		// Two users sharing a linked account would make the login they share
		// resolve to whichever one the map happened to hold, so reject it at
		// load rather than authenticate the wrong user later.
		if email := normalizeEmail(user.Email); email != "" {
			if existing, ok := users.byEmail[email]; ok {
				return users, fmt.Errorf("users %s and %s share the email %q", existing.Username, username, user.Email)
			}
			users.byEmail[email] = user
		}

		if github := normalizeGithub(user.Github); github != "" {
			if existing, ok := users.byGithub[github]; ok {
				return users, fmt.Errorf("users %s and %s share the github login %q", existing.Username, username, user.Github)
			}
			users.byGithub[github] = user
		}
	}

	return users, nil
}

// normalizeEmail and normalizeGithub key the secondary indexes.
//
// GitHub logins are unique case-insensitively — you cannot register "Amir20"
// once "amir20" exists — and the API hands back the canonical casing, which is
// not necessarily the casing an operator typed into users.yml. Folding both
// sides therefore cannot introduce an ambiguity, and not folding them turns a
// capitalization difference into a silent failed login.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeGithub(login string) string {
	return strings.ToLower(strings.TrimSpace(login))
}

func (u *UserDatabase) readFileIfChanged() error {
	if u.Path == "" {
		return nil
	}
	info, err := os.Stat(u.Path)
	if err != nil {
		return err
	}

	if info.ModTime().After(u.LastRead) {
		log.Info().Msg("Reloading user database")
		users, err := decodeUsersFromFile(u.Path)
		if err != nil {
			return err
		}
		u.Users = users.Users
		u.byEmail = users.byEmail
		u.byGithub = users.byGithub
		u.LastRead = time.Now()
	}

	return nil
}

func (u *UserDatabase) Find(username string) *User {
	if err := u.readFileIfChanged(); err != nil {
		log.Error().Err(err).Msg("Failed to read user database")
		return nil
	}
	user, ok := u.Users[username]
	if !ok {
		return nil
	}
	return user
}

// AnyPassword reports whether any configured user can sign in with a password.
//
// It fails open: on a read error the login page still offers the password form,
// because hiding it would strand an operator with no way in over what may be a
// transient problem.
func (u *UserDatabase) AnyPassword() bool {
	if err := u.readFileIfChanged(); err != nil {
		log.Error().Err(err).Msg("Failed to read user database")
		return true
	}

	for _, user := range u.Users {
		if user.Password != "" {
			return true
		}
	}

	return false
}

// FindByGithub resolves a GitHub login to the user that linked it in users.yml.
// users.yml is the allowlist: a GitHub account nobody linked has no match here,
// which is what makes the OAuth flow fail closed.
func (u *UserDatabase) FindByGithub(login string) *User {
	return u.findIndexed(normalizeGithub(login), func() map[string]*User { return u.byGithub })
}

// FindByEmail resolves a verified email to the user that claims it. Unused by
// the GitHub flow, which matches on the login, and there for generic OIDC.
func (u *UserDatabase) FindByEmail(email string) *User {
	return u.findIndexed(normalizeEmail(email), func() map[string]*User { return u.byEmail })
}

// findIndexed takes a selector rather than a map because readFileIfChanged
// replaces the index maps wholesale. Resolving the field after the reload is
// what keeps a lookup from reading the map the reload just discarded.
func (u *UserDatabase) findIndexed(key string, index func() map[string]*User) *User {
	if key == "" {
		return nil
	}

	if err := u.readFileIfChanged(); err != nil {
		log.Error().Err(err).Msg("Failed to read user database")
		return nil
	}

	return index()[key]
}

func CompareHashAndPassword(hash, password string) bool {
	// decodeUsersFromFile rejects a sha256 hash, so this is unreachable from a
	// file that loaded. It stays as a guard because this runs on an
	// unauthenticated request: exiting here turns "someone guessed a username"
	// into a process kill, and the users.yml reload can put a hash in front of
	// this long after startup.
	if len(hash) == 64 {
		log.Error().Msgf("sha256 passwords are no longer supported. Please use bcrypt. See %s for more details.", sha256AdvisoryURL)
		return false
	}

	if len(hash) == 60 {
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		return err == nil
	}

	log.Error().Int("length", len(hash)).Msg("Invalid hash length. Expecting 60 characters.")

	return false
}

// UserFromContext returns the user an authentication middleware resolved for this
// request. Both providers resolve the user themselves: proxy auth from the request
// headers, simple auth from users.yml keyed by the verified token's username. Roles
// deliberately are not read back out of the JWT, because a bitmask frozen at login
// goes stale the moment the role set grows or users.yml changes.
func UserFromContext(ctx context.Context) *User {
	if user, ok := ctx.Value(remoteUser).(User); ok {
		return &user
	}

	return nil
}

func RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		if user != nil {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}
	})
}
