package auth

import (
	"os"

	"gopkg.in/yaml.v3"
)

type User struct {
	Username string `json:"username"`
	Email    string `json:"email" yaml:"email"`
	Name     string `json:"name" yaml:"name"`
	Avatar   string `json:"avatar,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type Users map[string]*User

func ReadUsersFromFile(path string) (*Users, error) {
	users := make(Users)
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
