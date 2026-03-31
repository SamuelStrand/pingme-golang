package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"pingme-golang/internal/models"
)

var ErrEmailTaken = errors.New("email already exists")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrSessionNotFound = errors.New("session not found")

type Repository struct {
	DB *sqlx.DB
}

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash string) (models.User, error) {
	var u models.User
	err := r.DB.QueryRowxContext(ctx, `
		insert into users (email, password)
		values ($1, $2)
		returning id, user_tg, email, password, created_at
	`, email, passwordHash).Scan(&u.ID, &u.UserTG, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == "23505" {
			return models.User{}, ErrEmailTaken
		}
		return models.User{}, err
	}
	return u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, userID string) (models.User, error) {
	var u models.User
	err := r.DB.QueryRowxContext(ctx, `
		select id, user_tg, email, password, created_at
		from users
		where id = $1
	`, userID).Scan(&u.ID, &u.UserTG, &u.Email, &u.Password, &u.CreatedAt)
	return u, err
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	var u models.User
	err := r.DB.QueryRowxContext(ctx, `
		select id, user_tg, email, password, created_at
		from users
		where email = $1
	`, email).Scan(&u.ID, &u.UserTG, &u.Email, &u.Password, &u.CreatedAt)
	return u, err
}

func (r *Repository) CreateSession(ctx context.Context, userID string, expiresAt time.Time) (string, error) {
	var sessionID string
	err := r.DB.QueryRowxContext(ctx, `
		insert into auth_sessions (user_id, expires_at)
		values ($1, $2)
		returning id
	`, userID, expiresAt).Scan(&sessionID)
	return sessionID, err
}

func (r *Repository) GetSession(ctx context.Context, sessionID string) (userID string, expiresAt time.Time, revokedAt *time.Time, err error) {
	err = r.DB.QueryRowxContext(ctx, `
		select user_id, expires_at, revoked_at
		from auth_sessions
		where id = $1
	`, sessionID).Scan(&userID, &expiresAt, &revokedAt)
	return userID, expiresAt, revokedAt, err
}

func (r *Repository) RevokeSession(ctx context.Context, sessionID string) error {
	res, err := r.DB.ExecContext(ctx, `
		update auth_sessions
		set revoked_at = now()
		where id = $1 and revoked_at is null
	`, sessionID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrSessionNotFound
	}
	return nil
}
