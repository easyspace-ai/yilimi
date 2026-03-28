package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var ErrUnauthorized = errors.New("invalid credentials")

// Service handles registration and login backed by SQLite.
type Service struct {
	st *store
}

// NewService opens the auth database and runs migrations.
func NewService(dbPath string, jwtSecret string) (*Service, error) {
	SetJWTSecret(jwtSecret)
	st, err := openStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("auth db: %w", err)
	}
	return &Service{st: st}, nil
}

// Close releases the database handle.
func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	return s.st.Close()
}

// Register creates a user and returns a JWT.
func (s *Service) Register(ctx context.Context, username, contact, password string) (User, string, error) {
	if err := validateUsername(username); err != nil {
		return User{}, "", err
	}
	if err := validatePassword(password); err != nil {
		return User{}, "", err
	}
	email, phone, err := splitContact(contact)
	if err != nil {
		return User{}, "", err
	}
	if ok, err := s.st.usernameTaken(ctx, username); err != nil {
		return User{}, "", err
	} else if ok {
		return User{}, "", ErrUsernameTaken
	}
	if email != "" {
		if ok, err := s.st.emailTaken(ctx, email); err != nil {
			return User{}, "", err
		} else if ok {
			return User{}, "", ErrDuplicateContact
		}
	}
	if phone != "" {
		if ok, err := s.st.phoneTaken(ctx, phone); err != nil {
			return User{}, "", err
		} else if ok {
			return User{}, "", ErrDuplicateContact
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, "", err
	}
	urow, err := s.st.createUser(ctx, username, email, phone, string(hash))
	if err != nil {
		if isUniqueConstraint(err) {
			// Race on unique username/email/phone
			return User{}, "", ErrDuplicateContact
		}
		return User{}, "", err
	}
	u := dbUserToPublic(urow)
	tok, err := s.signToken(u)
	if err != nil {
		return User{}, "", err
	}
	return u, tok, nil
}

// Login validates credentials and returns a JWT.
func (s *Service) Login(ctx context.Context, login, password string) (User, string, error) {
	urow, err := s.st.getByLogin(ctx, login)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, "", ErrUnauthorized
	}
	if err != nil {
		return User{}, "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(urow.PasswordHash), []byte(password)); err != nil {
		return User{}, "", ErrUnauthorized
	}
	u := dbUserToPublic(urow)
	tok, err := s.signToken(u)
	if err != nil {
		return User{}, "", err
	}
	return u, tok, nil
}

// Me loads the user row for a user id (e.g. after token subject validation).
func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	urow, err := s.st.getByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUnauthorized
	}
	if err != nil {
		return User{}, err
	}
	return dbUserToPublic(urow), nil
}
