package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/mimsewelt/1984/services/posts/internal/model"
	"github.com/mimsewelt/1984/services/posts/internal/repository"
)

const defaultFeedLimit = 20

var (
	ErrNotFound     = errors.New("post not found")
	ErrForbidden    = errors.New("forbidden")
	ErrAlreadyLiked = errors.New("already liked")
	ErrNotLiked     = errors.New("not liked")
)

type PostRepo interface {
	Create(ctx context.Context, p *model.Post) error
	FindByID(ctx context.Context, id string) (*model.Post, error)
	Delete(ctx context.Context, id, userID string) error
	Feed(ctx context.Context, userID, cursor string, limit int) ([]*model.Post, error)
	UserPosts(ctx context.Context, userID, cursor string, limit int) ([]*model.Post, error)
	IsLikedBy(ctx context.Context, postID, userID string) (bool, error)
}

type LikeRepo interface {
	Like(ctx context.Context, postID, userID string) error
	Unlike(ctx context.Context, postID, userID string) error
}

type PostService struct {
	posts PostRepo
	likes LikeRepo
}

func NewPostService(posts PostRepo, likes LikeRepo) *PostService {
	return &PostService{posts: posts, likes: likes}
}

func (s *PostService) CreatePost(ctx context.Context, userID string, req *model.CreatePostRequest) (*model.PostResponse, error) {
	if len(req.MediaURLs) == 0 {
		return nil, errors.New("at least one media URL required")
	}
	mediaType := req.MediaType
	if mediaType == "" {
		mediaType = "image"
	}

	now := time.Now().UTC()
	post := &model.Post{
		ID:        uuid.NewString(),
		UserID:    userID,
		Caption:   req.Caption,
		MediaURLs: req.MediaURLs,
		MediaType: mediaType,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.posts.Create(ctx, post); err != nil {
		return nil, err
	}
	return toResponse(post, false), nil
}

func (s *PostService) GetPost(ctx context.Context, postID, viewerID string) (*model.PostResponse, error) {
	post, err := s.posts.FindByID(ctx, postID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	liked, _ := s.posts.IsLikedBy(ctx, postID, viewerID)
	return toResponse(post, liked), nil
}

func (s *PostService) DeletePost(ctx context.Context, postID, userID string) error {
	err := s.posts.Delete(ctx, postID, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *PostService) GetFeed(ctx context.Context, userID, cursor string) (*model.FeedResponse, error) {
	limit := defaultFeedLimit + 1
	posts, err := s.posts.Feed(ctx, userID, cursor, limit)
	if err != nil {
		return nil, err
	}
	return s.buildFeedResponse(ctx, userID, posts), nil
}

func (s *PostService) GetUserPosts(ctx context.Context, userID, viewerID, cursor string) (*model.FeedResponse, error) {
	limit := defaultFeedLimit + 1
	posts, err := s.posts.UserPosts(ctx, userID, cursor, limit)
	if err != nil {
		return nil, err
	}
	return s.buildFeedResponse(ctx, viewerID, posts), nil
}

func (s *PostService) LikePost(ctx context.Context, postID, userID string) error {
	err := s.likes.Like(ctx, postID, userID)
	if errors.Is(err, repository.ErrAlreadyLiked) {
		return ErrAlreadyLiked
	}
	return err
}

func (s *PostService) UnlikePost(ctx context.Context, postID, userID string) error {
	err := s.likes.Unlike(ctx, postID, userID)
	if errors.Is(err, repository.ErrNotLiked) {
		return ErrNotLiked
	}
	return err
}

func (s *PostService) buildFeedResponse(ctx context.Context, viewerID string, posts []*model.Post) *model.FeedResponse {
	hasMore := len(posts) > defaultFeedLimit
	if hasMore {
		posts = posts[:defaultFeedLimit]
	}

	items := make([]model.PostResponse, 0, len(posts))
	var nextCursor string

	for _, p := range posts {
		liked, _ := s.posts.IsLikedBy(ctx, p.ID, viewerID)
		items = append(items, *toResponse(p, liked))
		nextCursor = p.CreatedAt.Format(time.RFC3339Nano)
	}

	if !hasMore {
		nextCursor = ""
	}

	return &model.FeedResponse{
		Posts:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}

func toResponse(p *model.Post, likedByMe bool) *model.PostResponse {
	return &model.PostResponse{
		ID:            p.ID,
		UserID:        p.UserID,
		Caption:       p.Caption,
		MediaURLs:     p.MediaURLs,
		MediaType:     p.MediaType,
		LikesCount:    p.LikesCount,
		CommentsCount: p.CommentsCount,
		LikedByMe:     likedByMe,
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
	}
}
