package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mimsewelt/1984/services/posts/internal/model"
	"github.com/mimsewelt/1984/services/posts/internal/repository"
	"github.com/mimsewelt/1984/services/posts/internal/service"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakePostRepo struct {
	posts  map[string]*model.Post
	likes  map[string]map[string]bool // postID → userID → liked
	feed   []*model.Post
}

func newFakePostRepo() *fakePostRepo {
	return &fakePostRepo{
		posts: make(map[string]*model.Post),
		likes: make(map[string]map[string]bool),
	}
}

func (f *fakePostRepo) Create(_ context.Context, p *model.Post) error {
	f.posts[p.ID] = p
	return nil
}

func (f *fakePostRepo) FindByID(_ context.Context, id string) (*model.Post, error) {
	p, ok := f.posts[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return p, nil
}

func (f *fakePostRepo) Delete(_ context.Context, id, userID string) error {
	p, ok := f.posts[id]
	if !ok || p.UserID != userID {
		return repository.ErrNotFound
	}
	delete(f.posts, id)
	return nil
}

func (f *fakePostRepo) Feed(_ context.Context, userID, cursor string, limit int) ([]*model.Post, error) {
	return f.feed, nil
}

func (f *fakePostRepo) UserPosts(_ context.Context, userID, cursor string, limit int) ([]*model.Post, error) {
	var out []*model.Post
	for _, p := range f.posts {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakePostRepo) IsLikedBy(_ context.Context, postID, userID string) (bool, error) {
	return f.likes[postID][userID], nil
}

type fakeLikeRepo struct {
	likes map[string]map[string]bool
}

func newFakeLikeRepo() *fakeLikeRepo {
	return &fakeLikeRepo{likes: make(map[string]map[string]bool)}
}

func (f *fakeLikeRepo) Like(_ context.Context, postID, userID string) error {
	if f.likes[postID] == nil {
		f.likes[postID] = make(map[string]bool)
	}
	if f.likes[postID][userID] {
		return repository.ErrAlreadyLiked
	}
	f.likes[postID][userID] = true
	return nil
}

func (f *fakeLikeRepo) Unlike(_ context.Context, postID, userID string) error {
	if !f.likes[postID][userID] {
		return repository.ErrNotLiked
	}
	f.likes[postID][userID] = false
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newTestService() (*service.PostService, *fakePostRepo, *fakeLikeRepo) {
	postRepo := newFakePostRepo()
	likeRepo := newFakeLikeRepo()
	svc := service.NewPostService(postRepo, likeRepo)
	return svc, postRepo, likeRepo
}

func validCreateReq() *model.CreatePostRequest {
	return &model.CreatePostRequest{
		Caption:   "Hello world",
		MediaURLs: []string{"https://cdn.example.com/photo.jpg"},
		MediaType: "image",
	}
}

// ── CreatePost tests ──────────────────────────────────────────────────────────

func TestCreatePost_Success(t *testing.T) {
	svc, _, _ := newTestService()
	resp, err := svc.CreatePost(context.Background(), "user-1", validCreateReq())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty post ID")
	}
	if resp.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", resp.UserID)
	}
	if resp.Caption != "Hello world" {
		t.Errorf("expected caption 'Hello world', got %s", resp.Caption)
	}
	if len(resp.MediaURLs) != 1 {
		t.Errorf("expected 1 media URL, got %d", len(resp.MediaURLs))
	}
}

func TestCreatePost_NoMediaURL_Fails(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.CreatePost(context.Background(), "user-1", &model.CreatePostRequest{
		Caption:   "no media",
		MediaURLs: []string{},
	})
	if err == nil {
		t.Error("expected error for missing media URL")
	}
}

func TestCreatePost_DefaultsMediaTypeToImage(t *testing.T) {
	svc, _, _ := newTestService()
	resp, err := svc.CreatePost(context.Background(), "user-1", &model.CreatePostRequest{
		MediaURLs: []string{"https://cdn.example.com/photo.jpg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.MediaType != "image" {
		t.Errorf("expected media_type 'image', got %s", resp.MediaType)
	}
}

// ── GetPost tests ─────────────────────────────────────────────────────────────

func TestGetPost_Success(t *testing.T) {
	svc, _, _ := newTestService()
	created, _ := svc.CreatePost(context.Background(), "user-1", validCreateReq())

	got, err := svc.GetPost(context.Background(), created.ID, "user-2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected post ID %s, got %s", created.ID, got.ID)
	}
}

func TestGetPost_NotFound(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.GetPost(context.Background(), "nonexistent", "user-1")
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── DeletePost tests ──────────────────────────────────────────────────────────

func TestDeletePost_OwnerCanDelete(t *testing.T) {
	svc, _, _ := newTestService()
	created, _ := svc.CreatePost(context.Background(), "user-1", validCreateReq())

	err := svc.DeletePost(context.Background(), created.ID, "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeletePost_NonOwnerCannotDelete(t *testing.T) {
	svc, _, _ := newTestService()
	created, _ := svc.CreatePost(context.Background(), "user-1", validCreateReq())

	err := svc.DeletePost(context.Background(), created.ID, "user-2")
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("expected ErrNotFound for non-owner delete, got %v", err)
	}
}

// ── Like tests ────────────────────────────────────────────────────────────────

func TestLikePost_Success(t *testing.T) {
	svc, _, _ := newTestService()
	created, _ := svc.CreatePost(context.Background(), "user-1", validCreateReq())

	err := svc.LikePost(context.Background(), created.ID, "user-2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLikePost_CannotLikeTwice(t *testing.T) {
	svc, _, _ := newTestService()
	created, _ := svc.CreatePost(context.Background(), "user-1", validCreateReq())

	_ = svc.LikePost(context.Background(), created.ID, "user-2")
	err := svc.LikePost(context.Background(), created.ID, "user-2")
	if !errors.Is(err, service.ErrAlreadyLiked) {
		t.Errorf("expected ErrAlreadyLiked, got %v", err)
	}
}

func TestUnlikePost_Success(t *testing.T) {
	svc, _, _ := newTestService()
	created, _ := svc.CreatePost(context.Background(), "user-1", validCreateReq())

	_ = svc.LikePost(context.Background(), created.ID, "user-2")
	err := svc.UnlikePost(context.Background(), created.ID, "user-2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUnlikePost_CannotUnlikeIfNotLiked(t *testing.T) {
	svc, _, _ := newTestService()
	created, _ := svc.CreatePost(context.Background(), "user-1", validCreateReq())

	err := svc.UnlikePost(context.Background(), created.ID, "user-2")
	if !errors.Is(err, service.ErrNotLiked) {
		t.Errorf("expected ErrNotLiked, got %v", err)
	}
}

// ── Feed tests ────────────────────────────────────────────────────────────────

func TestGetFeed_Empty(t *testing.T) {
	svc, _, _ := newTestService()
	feed, err := svc.GetFeed(context.Background(), "user-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Posts) != 0 {
		t.Errorf("expected empty feed, got %d posts", len(feed.Posts))
	}
	if feed.HasMore {
		t.Error("expected HasMore=false for empty feed")
	}
}

func TestGetUserPosts_ReturnsOnlyUserPosts(t *testing.T) {
	svc, _, _ := newTestService()
	svc.CreatePost(context.Background(), "user-1", validCreateReq())
	svc.CreatePost(context.Background(), "user-1", validCreateReq())
	svc.CreatePost(context.Background(), "user-2", validCreateReq())

	feed, err := svc.GetUserPosts(context.Background(), "user-1", "user-1", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Posts) != 2 {
		t.Errorf("expected 2 posts for user-1, got %d", len(feed.Posts))
	}
	for _, p := range feed.Posts {
		if p.UserID != "user-1" {
			t.Errorf("expected only user-1 posts, got post from %s", p.UserID)
		}
	}
}
