package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/jwtauth/v5"
	"gopkg.in/yaml.v3"
)

type User struct {
	Username string `json:"username"`
	Email    string `json:"email" yaml:"email"`
	Name     string `json:"name" yaml:"name"`
	Avatar   string `json:"avatar,omitempty"`
	Password string `json:"-" yaml:"password"`
}

func newUser(username, email, name string) User {
	avatar := ""
	if email != "" {
		avatar = fmt.Sprintf("https://gravatar.com/avatar/%s?d=https%%3A%%2F%%2Fui-avatars.com%%2Fapi%%2F/%s/128", hashEmail(email), name)
	}
	return User{
		Username: username,
		Email:    email,
		Name:     name,
		Avatar:   avatar,
	}
}

type UserDatabase struct {
	Users map[string]*User `yaml:"users"`
}

func ReadUsersFromFile(path string) (UserDatabase, error) {
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
	}

	return users, nil
}

func (u *UserDatabase) Find(username string) *User {
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
