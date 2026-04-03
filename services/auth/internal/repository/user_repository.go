package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mimsewelt/1984/services/auth/internal/model"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("already exists")

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	q := `INSERT INTO users (id, username, email, password_hash, display_name, created_at, updated_at)
		  VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, q,
		u.ID, u.Username, u.Email, u.PasswordHash,
		u.DisplayName, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	q := `SELECT id, username, email, password_hash, display_name, bio, avatar_url, created_at, updated_at
		  FROM users WHERE email = $1 AND deleted_at IS NULL`
	u := &model.User{}
	err := r.db.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.DisplayName, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	q := `SELECT id, username, email, password_hash, display_name, bio, avatar_url, created_at, updated_at
		  FROM users WHERE id = $1 AND deleted_at IS NULL`
	u := &model.User{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash,
		&u.DisplayName, &u.Bio, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, t *model.RefreshToken) error {
	q := `INSERT INTO refresh_tokens (id, user_id, token_hash, device_id, expires_at, created_at)
		  VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, q, t.ID, t.UserID, t.TokenHash, t.DeviceID, t.ExpiresAt, t.CreatedAt)
	return err
}

func (r *RefreshTokenRepository) FindByUserAndDevice(ctx context.Context, userID, deviceID string) (*model.RefreshToken, error) {
	q := `SELECT id, user_id, token_hash, device_id, expires_at, created_at
		  FROM refresh_tokens WHERE user_id = $1 AND device_id = $2 AND expires_at > NOW()`
	t := &model.RefreshToken{}
	err := r.db.QueryRow(ctx, q, userID, deviceID).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.DeviceID, &t.ExpiresAt, &t.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *RefreshTokenRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE id = $1`, id)
	return err
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < $1`, time.Now())
	return err
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	type pgErr interface{ SQLState() string }
	var pe pgErr
	if errors.As(err, &pe) {
		return pe.SQLState() == "23505"
	}
	return false
}
