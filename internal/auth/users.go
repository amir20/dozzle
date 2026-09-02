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

type User struct {
	Username        string                    `json:"username" yaml:"-"`
	Email           string                    `json:"email" yaml:"email"`
	Name            string                    `json:"name" yaml:"name"`
	Password        string                    `json:"-" yaml:"password"`
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

	for username, user := range users.Users {
		user.Username = username
		if user.Password == "" {
			log.Fatal().Msgf("User %s has an empty password", username)
		}

		if !(len(user.Password) == 64 || len(user.Password) == 60) {
			log.Fatal().Str("password", user.Password).Str("user", username).Msg("Invalid password for user")
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
	}

	return users, nil
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

func CompareHashAndPassword(hash, password string) bool {
	if len(hash) == 64 {
		log.Fatal().Msg("sha256 passwords are no longer supported. Please use bcrypt. See https://github.com/amir20/dozzle/security/advisories/GHSA-w7qr-q9fh-fj35 for more details.")
	}

	if len(hash) == 60 {
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		return err == nil
	}

	log.Error().Str("hash", hash).Msg("Invalid hash length. Expecting 64 or 60 characters.")

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
