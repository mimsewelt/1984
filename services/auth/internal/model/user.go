package model

import "time"

type User struct {
 ID           string    db:"id"
 Username     string    db:"username"
 Email        string    db:"email"
 PasswordHash string    db:"password_hash"
 DisplayName  string    db:"display_name"
 Bio          string    db:"bio"
 AvatarURL    string    db:"avatar_url"
 CreatedAt    time.Time db:"created_at"
 UpdatedAt    time.Time db:"updated_at"
}

// RefreshToken stores a hashed refresh token with device metadata.
type RefreshToken struct {
 ID        string    db:"id"
 UserID    string    db:"user_id"
 TokenHash string    db:"token_hash" // bcrypt hash — never store raw
 DeviceID  string    db:"device_id"
 ExpiresAt time.Time db:"expires_at"
 CreatedAt time.Time db:"created_at"
}

// --- Request / Response DTOs ---

type RegisterRequest struct {
 Username    string json:"username"
 Email       string json:"email"
 Password    string json:"password"
 DisplayName string json:"display_name"
}

type LoginRequest struct {
 Email    string json:"email"
 Password string json:"password"
 DeviceID string json:"device_id" // optional, for refresh token tracking
}

type RefreshRequest struct {
 RefreshToken string json:"refresh_token"
}

type AuthResponse struct {
 AccessToken  string json:"access_token"
 RefreshToken string json:"refresh_token"
 ExpiresIn    int    json:"expires_in" // seconds
 User         UserDTO json:"user"
}

type UserDTO struct {
 ID          string json:"id"
 Username    string json:"username"
 Email       string json:"email"
 DisplayName string json:"display_name"
 AvatarURL   string json:"avatar_url"
}