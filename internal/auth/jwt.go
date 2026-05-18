package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type Service struct {
	username string
	password string
	secret   []byte
	ttl      time.Duration
}

func NewService(username, password, secret string, ttl time.Duration) *Service {
	return &Service{username: username, password: password, secret: []byte(secret), ttl: ttl}
}

func (s *Service) Authenticate(username, password string) bool {
	return username == s.username && password == s.password
}

func (s *Service) IssueToken(username string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   username,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *Service) Validate(tokenValue string) error {
	token, err := jwt.ParseWithClaims(tokenValue, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil || token == nil || !token.Valid {
		return ErrInvalidToken
	}
	return nil
}
