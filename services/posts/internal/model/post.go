package model

import "time"

type Post struct {
	ID            string
	UserID        string
	Caption       string
	MediaURLs     []string
	MediaType     string
	LikesCount    int
	CommentsCount int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Like struct {
	UserID    string
	PostID    string
	CreatedAt time.Time
}

type CreatePostRequest struct {
	Caption   string   `json:"caption"`
	MediaURLs []string `json:"media_urls"`
	MediaType string   `json:"media_type"`
}

type PostResponse struct {
	ID            string   `json:"id"`
	UserID        string   `json:"user_id"`
	Caption       string   `json:"caption"`
	MediaURLs     []string `json:"media_urls"`
	MediaType     string   `json:"media_type"`
	LikesCount    int      `json:"likes_count"`
	CommentsCount int      `json:"comments_count"`
	LikedByMe     bool     `json:"liked_by_me"`
	CreatedAt     string   `json:"created_at"`
}

type FeedResponse struct {
	Posts      []PostResponse `json:"posts"`
	NextCursor string         `json:"next_cursor,omitempty"`
	HasMore    bool           `json:"has_more"`
}
