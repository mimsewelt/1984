package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mimsewelt/1984/services/users/internal/model"
)

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	q := `SELECT id, username, email, display_name, bio, avatar_url,
		         is_verified, is_private, created_at
		  FROM users WHERE id = $1 AND deleted_at IS NULL`
	u := &model.User{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.DisplayName,
		&u.Bio, &u.AvatarURL, &u.IsVerified, &u.IsPrivate, &u.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	q := `SELECT id, username, email, display_name, bio, avatar_url,
		         is_verified, is_private, created_at
		  FROM users WHERE username = $1 AND deleted_at IS NULL`
	u := &model.User{}
	err := r.db.QueryRow(ctx, q, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.DisplayName,
		&u.Bio, &u.AvatarURL, &u.IsVerified, &u.IsPrivate, &u.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id string, req *model.UpdateProfileRequest) error {
	q := `UPDATE users SET display_name = $1, bio = $2, avatar_url = $3,
		  is_private = $4, updated_at = NOW()
		  WHERE id = $5 AND deleted_at IS NULL`
	tag, err := r.db.Exec(ctx, q, req.DisplayName, req.Bio, req.AvatarURL, req.IsPrivate, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) FollowersCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM follows WHERE following_id = $1`, userID,
	).Scan(&count)
	return count, err
}

func (r *UserRepository) FollowingCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM follows WHERE follower_id = $1`, userID,
	).Scan(&count)
	return count, err
}

func (r *UserRepository) Followers(ctx context.Context, userID, cursor string, limit int) ([]*model.FollowUser, error) {
	var q string
	var args []any
	if cursor == "" {
		q = `SELECT u.id, u.username, u.display_name, u.avatar_url, u.is_verified
			 FROM users u INNER JOIN follows f ON f.follower_id = u.id
			 WHERE f.following_id = $1 AND u.deleted_at IS NULL
			 ORDER BY f.created_at DESC LIMIT $2`
		args = []any{userID, limit}
	} else {
		q = `SELECT u.id, u.username, u.display_name, u.avatar_url, u.is_verified
			 FROM users u INNER JOIN follows f ON f.follower_id = u.id
			 WHERE f.following_id = $1 AND u.deleted_at IS NULL AND f.created_at < $2
			 ORDER BY f.created_at DESC LIMIT $3`
		args = []any{userID, cursor, limit}
	}
	return r.queryFollowUsers(ctx, q, args...)
}

func (r *UserRepository) Following(ctx context.Context, userID, cursor string, limit int) ([]*model.FollowUser, error) {
	var q string
	var args []any
	if cursor == "" {
		q = `SELECT u.id, u.username, u.display_name, u.avatar_url, u.is_verified
			 FROM users u INNER JOIN follows f ON f.following_id = u.id
			 WHERE f.follower_id = $1 AND u.deleted_at IS NULL
			 ORDER BY f.created_at DESC LIMIT $2`
		args = []any{userID, limit}
	} else {
		q = `SELECT u.id, u.username, u.display_name, u.avatar_url, u.is_verified
			 FROM users u INNER JOIN follows f ON f.following_id = u.id
			 WHERE f.follower_id = $1 AND u.deleted_at IS NULL AND f.created_at < $2
			 ORDER BY f.created_at DESC LIMIT $3`
		args = []any{userID, cursor, limit}
	}
	return r.queryFollowUsers(ctx, q, args...)
}

func (r *UserRepository) queryFollowUsers(ctx context.Context, q string, args ...any) ([]*model.FollowUser, error) {
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*model.FollowUser
	for rows.Next() {
		u := &model.FollowUser{}
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL, &u.IsVerified); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
