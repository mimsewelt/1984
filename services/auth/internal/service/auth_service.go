package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mimsewelt/1984/services/auth/internal/model"
	"github.com/mimsewelt/1984/services/auth/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrTokenExpired       = errors.New("token expired or invalid")
)

const (
	bcryptCost           = 12
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 30 * 24 * time.Hour
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type AuthService struct {
	users     *repository.UserRepository
	tokens    *repository.RefreshTokenRepository
	jwtSecret []byte
}

func NewAuthService(
	users *repository.UserRepository,
	tokens *repository.RefreshTokenRepository,
	jwtSecret string,
) *AuthService {
	return &AuthService{users: users, tokens: tokens, jwtSecret: []byte(jwtSecret)}
}

func (s *AuthService) Register(ctx context.Context, req *model.RegisterRequest) (*model.AuthResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	user := &model.User{
		ID:           uuid.NewString(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		DisplayName:  req.DisplayName,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return s.issueTokenPair(ctx, user, "web")
}

func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.AuthResponse, error) {
	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = "web"
	}
	return s.issueTokenPair(ctx, user, deviceID)
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken, deviceID string) (*model.AuthResponse, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(rawRefreshToken, claims, func(t *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		return nil, ErrTokenExpired
	}
	stored, err := s.tokens.FindByUserAndDevice(ctx, claims.UserID, deviceID)
	if err != nil {
		return nil, ErrTokenExpired
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.TokenHash), []byte(rawRefreshToken)); err != nil {
		return nil, ErrTokenExpired
	}
	_ = s.tokens.Delete(ctx, stored.ID)
	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	return s.issueTokenPair(ctx, user, deviceID)
}

func (s *AuthService) issueTokenPair(ctx context.Context, user *model.User, deviceID string) (*model.AuthResponse, error) {
	now := time.Now().UTC()
	accessClaims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenDuration)),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}
	rawRefresh, err := generateSecureToken(48)
	if err != nil {
		return nil, err
	}
	refreshHash, err := bcrypt.GenerateFromPassword([]byte(rawRefresh), bcryptCost)
	if err != nil {
		return nil, err
	}
	rt := &model.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: string(refreshHash),
		DeviceID:  deviceID,
		ExpiresAt: now.Add(refreshTokenDuration),
		CreatedAt: now,
	}
	if err := s.tokens.Save(ctx, rt); err != nil {
		return nil, err
	}
	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(accessTokenDuration.Seconds()),
		User: model.UserDTO{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
		},
	}, nil
}

func generateSecureToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
