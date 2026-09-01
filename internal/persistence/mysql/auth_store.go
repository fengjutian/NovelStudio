package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"novelstudio/internal/auth"
)

type AuthStore struct{ DB *sql.DB }

func (s AuthStore) CreateUser(ctx context.Context, user auth.User, hash string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO users(id,name,email,password_hash,created_at) VALUES(?,?,?,?,?)`, user.ID, user.Name, auth.NormalizeEmail(user.Email), hash, user.CreatedAt)
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return auth.ErrEmailTaken
	}
	return err
}
func (s AuthStore) UserByEmail(ctx context.Context, email string) (auth.User, string, error) {
	var u auth.User
	var hash string
	err := s.DB.QueryRowContext(ctx, `SELECT id,name,email,password_hash,created_at FROM users WHERE email=?`, auth.NormalizeEmail(email)).Scan(&u.ID, &u.Name, &u.Email, &hash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = auth.ErrNotFound
	}
	return u, hash, err
}
func (s AuthStore) CreateSession(ctx context.Context, hash, userID string, expires time.Time) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO user_sessions(token_hash,user_id,expires_at,created_at) VALUES(?,?,?,?)`, hash, userID, expires, time.Now().UTC())
	return err
}
func (s AuthStore) UserBySession(ctx context.Context, hash string, now time.Time) (auth.User, error) {
	var u auth.User
	err := s.DB.QueryRowContext(ctx, `SELECT u.id,u.name,u.email,u.created_at FROM user_sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=? AND s.expires_at>?`, hash, now).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		err = auth.ErrNotFound
	}
	return u, err
}
func (s AuthStore) DeleteSession(ctx context.Context, hash string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM user_sessions WHERE token_hash=?`, hash)
	return err
}
