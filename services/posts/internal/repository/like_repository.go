package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAlreadyLiked = errors.New("already liked")
var ErrNotLiked     = errors.New("not liked")

type LikeRepository struct {
	db *pgxpool.Pool
}

func NewLikeRepository(db *pgxpool.Pool) *LikeRepository {
	return &LikeRepository{db: db}
}

func (r *LikeRepository) Like(ctx context.Context, postID, userID string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO likes (post_id, user_id) VALUES ($1, $2)`,
		postID, userID,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyLiked
		}
		return err
	}
	return nil
}

func (r *LikeRepository) Unlike(ctx context.Context, postID, userID string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM likes WHERE post_id = $1 AND user_id = $2`,
		postID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotLiked
	}
	return nil
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

// suppress unused import
var _ = pgx.ErrNoRows
