package auth

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	email TEXT UNIQUE,
	phone TEXT UNIQUE,
	password_hash TEXT NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);
`

type store struct {
	db *sql.DB
}

func openStore(path string) (*store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &store{db: db}, nil
}

func (s *store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

type dbUser struct {
	ID           string
	Username     string
	Email        sql.NullString
	Phone        sql.NullString
	PasswordHash string
	CreatedAt    int64
}

func (s *store) createUser(ctx context.Context, username, email, phone, hash string) (*dbUser, error) {
	id := uuid.New().String()
	now := time.Now().Unix()
	var em any
	var ph any
	if email != "" {
		em = email
	} else {
		em = nil
	}
	if phone != "" {
		ph = phone
	} else {
		ph = nil
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, username, email, phone, password_hash, created_at) VALUES (?,?,?,?,?,?)`,
		id, username, em, ph, hash, now,
	)
	if err != nil {
		return nil, err
	}
	u := &dbUser{ID: id, Username: username, PasswordHash: hash, CreatedAt: now}
	if email != "" {
		u.Email = sql.NullString{String: email, Valid: true}
	}
	if phone != "" {
		u.Phone = sql.NullString{String: phone, Valid: true}
	}
	return u, nil
}

func (s *store) getByLogin(ctx context.Context, login string) (*dbUser, error) {
	login = strings.TrimSpace(login)
	loginLower := strings.ToLower(login)
	phoneKey := digitsOnly(login)
	var u dbUser
	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, email, phone, password_hash, created_at FROM users
		WHERE username = ?
		   OR (email IS NOT NULL AND lower(email) = ?)
		   OR (phone IS NOT NULL AND phone = ? AND ? != '')
	`, login, loginLower, phoneKey, phoneKey).Scan(
		&u.ID, &u.Username, &u.Email, &u.Phone, &u.PasswordHash, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *store) getByID(ctx context.Context, id string) (*dbUser, error) {
	var u dbUser
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, phone, password_hash, created_at FROM users WHERE id = ?`,
		strings.TrimSpace(id),
	).Scan(&u.ID, &u.Username, &u.Email, &u.Phone, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *store) usernameTaken(ctx context.Context, username string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users WHERE username = ?`, username).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *store) emailTaken(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, nil
	}
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users WHERE lower(email) = ?`, strings.ToLower(email)).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *store) phoneTaken(ctx context.Context, phone string) (bool, error) {
	if phone == "" {
		return false, nil
	}
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users WHERE phone = ?`, phone).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func dbUserToPublic(u *dbUser) User {
	out := User{
		ID:    u.ID,
		Name:  u.Username,
		Role:  "user",
		Email: "",
	}
	if u.Email.Valid {
		out.Email = u.Email.String
	} else if u.Phone.Valid {
		out.Email = u.Phone.String
	}
	return out
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint")
}
