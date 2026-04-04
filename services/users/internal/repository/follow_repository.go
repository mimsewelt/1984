package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAlreadyFollowing = errors.New("already following")
var ErrNotFollowing     = errors.New("not following")
var ErrCannotFollowSelf = errors.New("cannot follow yourself")

type FollowRepository struct {
	db *pgxpool.Pool
}

func NewFollowRepository(db *pgxpool.Pool) *FollowRepository {
	return &FollowRepository{db: db}
}

func (r *FollowRepository) Follow(ctx context.Context, followerID, followingID string) error {
	if followerID == followingID {
		return ErrCannotFollowSelf
	}
	_, err := r.db.Exec(ctx,
		`INSERT INTO follows (follower_id, following_id) VALUES ($1, $2)`,
		followerID, followingID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyFollowing
		}
		return err
	}
	return nil
}

func (r *FollowRepository) Unfollow(ctx context.Context, followerID, followingID string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM follows WHERE follower_id = $1 AND following_id = $2`,
		followerID, followingID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFollowing
	}
	return nil
}

func (r *FollowRepository) IsFollowing(ctx context.Context, followerID, followingID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)`,
		followerID, followingID,
	).Scan(&exists)
	return exists, err
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
