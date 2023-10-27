package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"os"

	"gopkg.in/yaml.v3"

	log "github.com/sirupsen/logrus"
)

type User struct {
	Username string `json:"username"`
	Email    string `json:"email" yaml:"email"`
	Name     string `json:"name" yaml:"name"`
	Avatar   string `json:"avatar,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type UserDatabase struct {
	Users map[string]*User `yaml:"users"`
}

func ReadUsersFromFile(path string) (*UserDatabase, error) {
	users := UserDatabase{}
	file, err := os.Open(path)
	if err != nil {
		return &users, err
	}
	defer file.Close()

	if err := yaml.NewDecoder(file).Decode(&users); err != nil {
		return &users, err
	}

	return &users, nil
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

	log.Infof("Password %s with %s", user.Password, sha256sum(password))

	if user.Password != sha256sum(password) {
		return nil
	}
	return user
}

func sha256sum(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
