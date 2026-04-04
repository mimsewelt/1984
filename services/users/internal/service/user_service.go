package service

import (
	"context"
	"errors"
	"time"

	"github.com/mimsewelt/1984/services/users/internal/model"
	"github.com/mimsewelt/1984/services/users/internal/repository"
)

const defaultListLimit = 20

var (
	ErrNotFound        = errors.New("user not found")
	ErrAlreadyFollowing = errors.New("already following")
	ErrNotFollowing    = errors.New("not following")
	ErrCannotFollowSelf = errors.New("cannot follow yourself")
	ErrForbidden       = errors.New("forbidden")
)

type UserRepo interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	UpdateProfile(ctx context.Context, id string, req *model.UpdateProfileRequest) error
	FollowersCount(ctx context.Context, userID string) (int, error)
	FollowingCount(ctx context.Context, userID string) (int, error)
	Followers(ctx context.Context, userID, cursor string, limit int) ([]*model.FollowUser, error)
	Following(ctx context.Context, userID, cursor string, limit int) ([]*model.FollowUser, error)
}

type FollowRepo interface {
	Follow(ctx context.Context, followerID, followingID string) error
	Unfollow(ctx context.Context, followerID, followingID string) error
	IsFollowing(ctx context.Context, followerID, followingID string) (bool, error)
}

type UserService struct {
	users   UserRepo
	follows FollowRepo
}

func NewUserService(users UserRepo, follows FollowRepo) *UserService {
	return &UserService{users: users, follows: follows}
}

func (s *UserService) GetProfile(ctx context.Context, targetID, viewerID string) (*model.ProfileResponse, error) {
	user, err := s.users.FindByID(ctx, targetID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.buildProfile(ctx, user, viewerID)
}

func (s *UserService) GetProfileByUsername(ctx context.Context, username, viewerID string) (*model.ProfileResponse, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.buildProfile(ctx, user, viewerID)
}

func (s *UserService) UpdateProfile(ctx context.Context, userID string, req *model.UpdateProfileRequest) (*model.ProfileResponse, error) {
	if err := s.users.UpdateProfile(ctx, userID, req); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.GetProfile(ctx, userID, userID)
}

func (s *UserService) Follow(ctx context.Context, followerID, followingID string) error {
	if followerID == followingID {
		return ErrCannotFollowSelf
	}
	// Verify target user exists.
	if _, err := s.users.FindByID(ctx, followingID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	err := s.follows.Follow(ctx, followerID, followingID)
	if errors.Is(err, repository.ErrAlreadyFollowing) {
		return ErrAlreadyFollowing
	}
	return err
}

func (s *UserService) Unfollow(ctx context.Context, followerID, followingID string) error {
	err := s.follows.Unfollow(ctx, followerID, followingID)
	if errors.Is(err, repository.ErrNotFollowing) {
		return ErrNotFollowing
	}
	return err
}

func (s *UserService) GetFollowers(ctx context.Context, userID, cursor string) (*model.FollowListResponse, error) {
	users, err := s.users.Followers(ctx, userID, cursor, defaultListLimit+1)
	if err != nil {
		return nil, err
	}
	return buildFollowList(users), nil
}

func (s *UserService) GetFollowing(ctx context.Context, userID, cursor string) (*model.FollowListResponse, error) {
	users, err := s.users.Following(ctx, userID, cursor, defaultListLimit+1)
	if err != nil {
		return nil, err
	}
	return buildFollowList(users), nil
}

func (s *UserService) buildProfile(ctx context.Context, user *model.User, viewerID string) (*model.ProfileResponse, error) {
	followersCount, _ := s.users.FollowersCount(ctx, user.ID)
	followingCount, _ := s.users.FollowingCount(ctx, user.ID)
	isFollowed, _ := s.follows.IsFollowing(ctx, viewerID, user.ID)

	return &model.ProfileResponse{
		ID:             user.ID,
		Username:       user.Username,
		DisplayName:    user.DisplayName,
		Bio:            user.Bio,
		AvatarURL:      user.AvatarURL,
		IsVerified:     user.IsVerified,
		IsPrivate:      user.IsPrivate,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		IsFollowedByMe: isFollowed,
		IsMe:           user.ID == viewerID,
		CreatedAt:      user.CreatedAt.Format(time.RFC3339),
	}, nil
}

func buildFollowList(users []*model.FollowUser) *model.FollowListResponse {
	hasMore := len(users) > defaultListLimit
	if hasMore {
		users = users[:defaultListLimit]
	}
	items := make([]model.FollowUser, 0, len(users))
	for _, u := range users {
		items = append(items, *u)
	}
	return &model.FollowListResponse{
		Users:   items,
		HasMore: hasMore,
	}
}
