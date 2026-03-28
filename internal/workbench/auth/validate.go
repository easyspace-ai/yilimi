package auth

import (
	"errors"
	"strings"
	"unicode"
)

var (
	ErrInvalidUsername  = errors.New("invalid username")
	ErrUsernameTaken    = errors.New("username already taken")
	ErrInvalidContact   = errors.New("invalid email or phone")
	ErrInvalidPassword  = errors.New("password must be at least 8 characters with letters and digits")
	ErrDuplicateContact = errors.New("username, email or phone already registered")
)

func validateUsername(username string) error {
	s := strings.TrimSpace(username)
	if len(s) < 3 || len(s) > 32 {
		return ErrInvalidUsername
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			continue
		}
		return ErrInvalidUsername
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrInvalidPassword
	}
	var letter, digit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			letter = true
		}
		if unicode.IsDigit(r) {
			digit = true
		}
	}
	if !letter || !digit {
		return ErrInvalidPassword
	}
	return nil
}

// splitContact returns normalized email (lower, trim) or phone (digits only).
func splitContact(contact string) (email string, phone string, err error) {
	s := strings.TrimSpace(contact)
	if s == "" {
		return "", "", ErrInvalidContact
	}
	if strings.Contains(s, "@") {
		parts := strings.SplitN(s, "@", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", ErrInvalidContact
		}
		return strings.ToLower(s), "", nil
	}
	d := digitsOnly(s)
	if len(d) == 11 && d[0] == '1' {
		return "", d, nil
	}
	return "", "", ErrInvalidContact
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
