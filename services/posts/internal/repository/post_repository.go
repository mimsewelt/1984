package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mimsewelt/1984/services/posts/internal/model"
)

var ErrNotFound = errors.New("not found")

type PostRepository struct {
	db *pgxpool.Pool
}

func NewPostRepository(db *pgxpool.Pool) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) Create(ctx context.Context, p *model.Post) error {
	q := `INSERT INTO posts (id, user_id, caption, media_urls, media_type, created_at, updated_at)
		  VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, q,
		p.ID, p.UserID, p.Caption, p.MediaURLs,
		p.MediaType, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *PostRepository) FindByID(ctx context.Context, id string) (*model.Post, error) {
	q := `SELECT id, user_id, caption, media_urls, media_type,
		         likes_count, comments_count, created_at, updated_at
		  FROM posts WHERE id = $1 AND deleted_at IS NULL`
	p := &model.Post{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.UserID, &p.Caption, &p.MediaURLs, &p.MediaType,
		&p.LikesCount, &p.CommentsCount, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *PostRepository) Delete(ctx context.Context, id, userID string) error {
	q := `UPDATE posts SET deleted_at = NOW() WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`
	tag, err := r.db.Exec(ctx, q, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Feed returns posts from users that userID follows, ordered by created_at DESC.
// cursor is the created_at timestamp of the last post seen (for pagination).
func (r *PostRepository) Feed(ctx context.Context, userID, cursor string, limit int) ([]*model.Post, error) {
	var q string
	var args []any

	if cursor == "" {
		q = `SELECT p.id, p.user_id, p.caption, p.media_urls, p.media_type,
			        p.likes_count, p.comments_count, p.created_at, p.updated_at
			 FROM posts p
			 INNER JOIN follows f ON f.following_id = p.user_id
			 WHERE f.follower_id = $1 AND p.deleted_at IS NULL
			 ORDER BY p.created_at DESC
			 LIMIT $2`
		args = []any{userID, limit}
	} else {
		q = `SELECT p.id, p.user_id, p.caption, p.media_urls, p.media_type,
			        p.likes_count, p.comments_count, p.created_at, p.updated_at
			 FROM posts p
			 INNER JOIN follows f ON f.following_id = p.user_id
			 WHERE f.follower_id = $1 AND p.deleted_at IS NULL
			   AND p.created_at < $2
			 ORDER BY p.created_at DESC
			 LIMIT $3`
		args = []any{userID, cursor, limit}
	}

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		p := &model.Post{}
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Caption, &p.MediaURLs, &p.MediaType,
			&p.LikesCount, &p.CommentsCount, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func (r *PostRepository) UserPosts(ctx context.Context, userID, cursor string, limit int) ([]*model.Post, error) {
	var q string
	var args []any

	if cursor == "" {
		q = `SELECT id, user_id, caption, media_urls, media_type,
			        likes_count, comments_count, created_at, updated_at
			 FROM posts
			 WHERE user_id = $1 AND deleted_at IS NULL
			 ORDER BY created_at DESC LIMIT $2`
		args = []any{userID, limit}
	} else {
		q = `SELECT id, user_id, caption, media_urls, media_type,
			        likes_count, comments_count, created_at, updated_at
			 FROM posts
			 WHERE user_id = $1 AND deleted_at IS NULL AND created_at < $2
			 ORDER BY created_at DESC LIMIT $3`
		args = []any{userID, cursor, limit}
	}

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		p := &model.Post{}
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Caption, &p.MediaURLs, &p.MediaType,
			&p.LikesCount, &p.CommentsCount, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func (r *PostRepository) IsLikedBy(ctx context.Context, postID, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM likes WHERE post_id = $1 AND user_id = $2)`,
		postID, userID,
	).Scan(&exists)
	return exists, err
}
