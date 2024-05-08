package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-chi/jwtauth/v5"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type User struct {
	Username string `json:"username" yaml:"-"`
	Email    string `json:"email" yaml:"email"`
	Name     string `json:"name" yaml:"name"`
	Password string `json:"-" yaml:"password"`
}

func (u User) AvatarURL() string {
	name := u.Name
	if name == "" {
		name = u.Username
	}
	return fmt.Sprintf("https://gravatar.com/avatar/%s?d=https%%3A%%2F%%2Fui-avatars.com%%2Fapi%%2F/%s/128", hashEmail(u.Email), url.QueryEscape(name))
}

func newUser(username, email, name string) User {
	return User{
		Username: username,
		Email:    email,
		Name:     name,
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
		user.Password = sha256sum(user.Password)
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
			log.Fatalf("User %s has no password", username)
		}

		if len(user.Password) != 64 {
			log.Fatalf("User %s has an invalid password hash", username)
		}

		if user.Name == "" {
			user.Name = username
		}
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
		log.Infof("Found changes to %s. Updating users...", u.Path)
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
		log.Errorf("Error reading users file: %s", err)
	}
	user, ok := u.Users[username]
	if !ok {
		return nil
	}
	return user
}

func (u *UserDatabase) FindByPassword(username, password string) *User {
	user := u.Find(username)

	if user == nil {
		return nil
	}

	if user.Password != sha256sum(password) {
		return nil
	}
	return user
}

func sha256sum(s string) string {
	bytes := sha256.Sum256([]byte(s))
	return hex.EncodeToString(bytes[:])
}

func UserFromContext(ctx context.Context) *User {
	if user, ok := ctx.Value(remoteUser).(User); ok {
		return &user
	} else {
		if _, claims, err := jwtauth.FromContext(ctx); err == nil {
			username, ok := claims["username"].(string)
			if !ok {
				return nil
			}
			if username == "" {
				return nil
			}
			email := claims["email"].(string)
			name := claims["name"].(string)
			user := newUser(username, email, name)
			return &user
		}
		return nil
	}
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
