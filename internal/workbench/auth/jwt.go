package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenTTL = 7 * 24 * time.Hour

var jwtSecretKey []byte

// SetJWTSecret configures signing/verification. NewService calls this automatically.
func SetJWTSecret(secret string) {
	jwtSecretKey = []byte(strings.TrimSpace(secret))
}

type tokenClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func (s *Service) signToken(u User) (string, error) {
	if len(jwtSecretKey) == 0 {
		return "", errors.New("jwt secret not configured")
	}
	now := time.Now()
	claims := tokenClaims{
		Email: u.Email,
		Name:  u.Name,
		Role:  u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(jwtSecretKey)
}

// UserFromBearer parses Authorization: Bearer <token> and returns the user embedded in JWT.
func UserFromBearer(header string) User {
	if len(jwtSecretKey) == 0 {
		return guestUser
	}
	raw := strings.TrimSpace(header)
	if !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return guestUser
	}
	tok := strings.TrimSpace(raw[7:])
	if tok == "" {
		return guestUser
	}
	t, err := jwt.ParseWithClaims(tok, &tokenClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecretKey, nil
	})
	if err != nil {
		return guestUser
	}
	claims, ok := t.Claims.(*tokenClaims)
	if !ok || !t.Valid || claims.Subject == "" {
		return guestUser
	}
	return User{
		ID:    claims.Subject,
		Email: claims.Email,
		Name:  claims.Name,
		Role:  claims.Role,
	}
}
