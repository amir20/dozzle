package auth

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/jwtauth/v5"
)

type simpleAuthContext struct {
	UserDatabase UserDatabase
	tokenAuth    *jwtauth.JWTAuth
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func NewSimpleAuth(userDatabase UserDatabase) *simpleAuthContext {
	h := sha256.New()
	for _, user := range userDatabase.Users {
		h.Write([]byte(user.Password))
	}

	tokenAuth := jwtauth.New("HS256", h.Sum(nil), nil)

	return &simpleAuthContext{
		UserDatabase: userDatabase,
		tokenAuth:    tokenAuth,
	}
}

func (a *simpleAuthContext) CreateToken(username, password string) (string, error) {
	user := a.UserDatabase.FindByPassword(username, password)
	if user == nil {
		return "", ErrInvalidCredentials
	}

	_, tokenString, err := a.tokenAuth.Encode(map[string]interface{}{"username": user.Username, "email": user.Email, "name": user.Name, "timestamp": time.Now()})
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (a *simpleAuthContext) AuthMiddleware(next http.Handler) http.Handler {
	return jwtauth.Verifier(a.tokenAuth)(next)
}
