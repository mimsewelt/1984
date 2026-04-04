package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mimsewelt/1984/services/users/internal/model"
	"github.com/mimsewelt/1984/services/users/internal/repository"
	"github.com/mimsewelt/1984/services/users/internal/service"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeUserRepo struct {
	users map[string]*model.User
	byUsername map[string]*model.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users:      make(map[string]*model.User),
		byUsername: make(map[string]*model.User),
	}
}

func (f *fakeUserRepo) add(u *model.User) {
	f.users[u.ID] = u
	f.byUsername[u.Username] = u
}

func (f *fakeUserRepo) FindByID(_ context.Context, id string) (*model.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) FindByUsername(_ context.Context, username string) (*model.User, error) {
	u, ok := f.byUsername[username]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) UpdateProfile(_ context.Context, id string, req *model.UpdateProfileRequest) error {
	u, ok := f.users[id]
	if !ok {
		return repository.ErrNotFound
	}
	u.DisplayName = req.DisplayName
	u.Bio = req.Bio
	u.AvatarURL = req.AvatarURL
	u.IsPrivate = req.IsPrivate
	return nil
}

func (f *fakeUserRepo) FollowersCount(_ context.Context, userID string) (int, error) {
	return 0, nil
}

func (f *fakeUserRepo) FollowingCount(_ context.Context, userID string) (int, error) {
	return 0, nil
}

func (f *fakeUserRepo) Followers(_ context.Context, userID, cursor string, limit int) ([]*model.FollowUser, error) {
	return nil, nil
}

func (f *fakeUserRepo) Following(_ context.Context, userID, cursor string, limit int) ([]*model.FollowUser, error) {
	return nil, nil
}

type fakeFollowRepo struct {
	follows map[string]map[string]bool
}

func newFakeFollowRepo() *fakeFollowRepo {
	return &fakeFollowRepo{follows: make(map[string]map[string]bool)}
}

func (f *fakeFollowRepo) Follow(_ context.Context, followerID, followingID string) error {
	if followerID == followingID {
		return repository.ErrCannotFollowSelf
	}
	if f.follows[followerID] == nil {
		f.follows[followerID] = make(map[string]bool)
	}
	if f.follows[followerID][followingID] {
		return repository.ErrAlreadyFollowing
	}
	f.follows[followerID][followingID] = true
	return nil
}

func (f *fakeFollowRepo) Unfollow(_ context.Context, followerID, followingID string) error {
	if !f.follows[followerID][followingID] {
		return repository.ErrNotFollowing
	}
	f.follows[followerID][followingID] = false
	return nil
}

func (f *fakeFollowRepo) IsFollowing(_ context.Context, followerID, followingID string) (bool, error) {
	return f.follows[followerID][followingID], nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newTestService() (*service.UserService, *fakeUserRepo, *fakeFollowRepo) {
	userRepo   := newFakeUserRepo()
	followRepo := newFakeFollowRepo()
	svc        := service.NewUserService(userRepo, followRepo)
	return svc, userRepo, followRepo
}

func testUser(id, username string) *model.User {
	return &model.User{
		ID:          id,
		Username:    username,
		Email:       username + "@example.com",
		DisplayName: username,
		CreatedAt:   time.Now(),
	}
}

// ── GetProfile tests ──────────────────────────────────────────────────────────

func TestGetProfile_Success(t *testing.T) {
	svc, userRepo, _ := newTestService()
	userRepo.add(testUser("user-1", "alice"))

	profile, err := svc.GetProfile(context.Background(), "user-1", "user-2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if profile.Username != "alice" {
		t.Errorf("expected alice, got %s", profile.Username)
	}
	if profile.IsMe {
		t.Error("expected IsMe=false when viewer != target")
	}
}

func TestGetProfile_IsMe_WhenViewerIsTarget(t *testing.T) {
	svc, userRepo, _ := newTestService()
	userRepo.add(testUser("user-1", "alice"))

	profile, err := svc.GetProfile(context.Background(), "user-1", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !profile.IsMe {
		t.Error("expected IsMe=true when viewer == target")
	}
}

func TestGetProfile_NotFound(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.GetProfile(context.Background(), "nonexistent", "viewer")
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetProfileByUsername_Success(t *testing.T) {
	svc, userRepo, _ := newTestService()
	userRepo.add(testUser("user-1", "alice"))

	profile, err := svc.GetProfileByUsername(context.Background(), "alice", "user-2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if profile.ID != "user-1" {
		t.Errorf("expected user-1, got %s", profile.ID)
	}
}

// ── UpdateProfile tests ───────────────────────────────────────────────────────

func TestUpdateProfile_Success(t *testing.T) {
	svc, userRepo, _ := newTestService()
	userRepo.add(testUser("user-1", "alice"))

	profile, err := svc.UpdateProfile(context.Background(), "user-1", &model.UpdateProfileRequest{
		DisplayName: "Alice Updated",
		Bio:         "My new bio",
		IsPrivate:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if profile.DisplayName != "Alice Updated" {
		t.Errorf("expected updated display name, got %s", profile.DisplayName)
	}
	if profile.Bio != "My new bio" {
		t.Errorf("expected updated bio, got %s", profile.Bio)
	}
	if !profile.IsPrivate {
		t.Error("expected IsPrivate=true")
	}
}

func TestUpdateProfile_NotFound(t *testing.T) {
	svc, _, _ := newTestService()
	_, err := svc.UpdateProfile(context.Background(), "nonexistent", &model.UpdateProfileRequest{})
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ── Follow tests ──────────────────────────────────────────────────────────────

func TestFollow_Success(t *testing.T) {
	svc, userRepo, followRepo := newTestService()
	userRepo.add(testUser("user-1", "alice"))
	userRepo.add(testUser("user-2", "bob"))

	err := svc.Follow(context.Background(), "user-1", "user-2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	isFollowing, _ := followRepo.IsFollowing(context.Background(), "user-1", "user-2")
	if !isFollowing {
		t.Error("expected user-1 to follow user-2")
	}
}

func TestFollow_CannotFollowSelf(t *testing.T) {
	svc, userRepo, _ := newTestService()
	userRepo.add(testUser("user-1", "alice"))

	err := svc.Follow(context.Background(), "user-1", "user-1")
	if !errors.Is(err, service.ErrCannotFollowSelf) {
		t.Errorf("expected ErrCannotFollowSelf, got %v", err)
	}
}

func TestFollow_CannotFollowTwice(t *testing.T) {
	svc, userRepo, _ := newTestService()
	userRepo.add(testUser("user-1", "alice"))
	userRepo.add(testUser("user-2", "bob"))

	_ = svc.Follow(context.Background(), "user-1", "user-2")
	err := svc.Follow(context.Background(), "user-1", "user-2")
	if !errors.Is(err, service.ErrAlreadyFollowing) {
		t.Errorf("expected ErrAlreadyFollowing, got %v", err)
	}
}

func TestFollow_TargetNotFound(t *testing.T) {
	svc, userRepo, _ := newTestService()
	userRepo.add(testUser("user-1", "alice"))

	err := svc.Follow(context.Background(), "user-1", "nonexistent")
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUnfollow_Success(t *testing.T) {
	svc, userRepo, _ := newTestService()
	userRepo.add(testUser("user-1", "alice"))
	userRepo.add(testUser("user-2", "bob"))

	_ = svc.Follow(context.Background(), "user-1", "user-2")
	err := svc.Unfollow(context.Background(), "user-1", "user-2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUnfollow_CannotUnfollowIfNotFollowing(t *testing.T) {
	svc, _, _ := newTestService()
	err := svc.Unfollow(context.Background(), "user-1", "user-2")
	if !errors.Is(err, service.ErrNotFollowing) {
		t.Errorf("expected ErrNotFollowing, got %v", err)
	}
}

func TestIsFollowedByMe_ReflectedInProfile(t *testing.T) {
	svc, userRepo, _ := newTestService()
	userRepo.add(testUser("user-1", "alice"))
	userRepo.add(testUser("user-2", "bob"))

	_ = svc.Follow(context.Background(), "user-1", "user-2")

	profile, err := svc.GetProfile(context.Background(), "user-2", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !profile.IsFollowedByMe {
		t.Error("expected IsFollowedByMe=true after following")
	}
}
