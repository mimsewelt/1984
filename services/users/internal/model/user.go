package model

import "time"

type User struct {
	ID          string
	Username    string
	Email       string
	DisplayName string
	Bio         string
	AvatarURL   string
	IsVerified  bool
	IsPrivate   bool
	CreatedAt   time.Time
}

type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	IsPrivate   bool   `json:"is_private"`
}

type ProfileResponse struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	DisplayName    string `json:"display_name"`
	Bio            string `json:"bio"`
	AvatarURL      string `json:"avatar_url"`
	IsVerified     bool   `json:"is_verified"`
	IsPrivate      bool   `json:"is_private"`
	FollowersCount int    `json:"followers_count"`
	FollowingCount int    `json:"following_count"`
	IsFollowedByMe bool   `json:"is_followed_by_me"`
	IsMe           bool   `json:"is_me"`
	CreatedAt      string `json:"created_at"`
}

type FollowUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	IsVerified  bool   `json:"is_verified"`
}

type FollowListResponse struct {
	Users      []FollowUser `json:"users"`
	NextCursor string       `json:"next_cursor,omitempty"`
	HasMore    bool         `json:"has_more"`
}

