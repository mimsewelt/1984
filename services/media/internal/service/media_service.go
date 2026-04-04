package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	maxImageSize = 10 << 20 // 10 MB
	maxVideoSize = 100 << 20 // 100 MB
)

var (
	ErrFileTooLarge    = errors.New("file too large")
	ErrInvalidType     = errors.New("invalid file type")
)

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

var allowedVideoTypes = map[string]bool{
	"video/mp4":  true,
	"video/webm": true,
}

type Storage interface {
	Upload(ctx context.Context, r io.Reader, size int64, contentType, folder string) (string, error)
	PresignedURL(ctx context.Context, key string) (string, error)
	PresignedUploadURL(ctx context.Context, folder, contentType string) (string, string, error)
	Delete(ctx context.Context, key string) error
}

type UploadResult struct {
	Key         string `json:"key"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type PresignResult struct {
	Key       string `json:"key"`
	UploadURL string `json:"upload_url"`
}

type MediaService struct {
	storage Storage
}

func NewMediaService(storage Storage) *MediaService {
	return &MediaService{storage: storage}
}

// Upload handles a direct file upload from the client.
func (s *MediaService) Upload(ctx context.Context, r io.Reader, size int64, contentType, userID string) (*UploadResult, error) {
	if err := validateType(contentType); err != nil {
		return nil, err
	}
	if err := validateSize(contentType, size); err != nil {
		return nil, err
	}

	folder := folderForUser(userID, contentType)
	key, err := s.storage.Upload(ctx, r, size, contentType, folder)
	if err != nil {
		return nil, fmt.Errorf("upload: %w", err)
	}

	url, err := s.storage.PresignedURL(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("presign after upload: %w", err)
	}

	return &UploadResult{
		Key:         key,
		URL:         url,
		ContentType: contentType,
		Size:        size,
	}, nil
}

// RequestPresignedUpload returns a presigned URL for direct client-to-MinIO upload.
// This is the preferred flow — binary data never passes through our service.
func (s *MediaService) RequestPresignedUpload(ctx context.Context, contentType, userID string) (*PresignResult, error) {
	if err := validateType(contentType); err != nil {
		return nil, err
	}

	folder := folderForUser(userID, contentType)
	key, uploadURL, err := s.storage.PresignedUploadURL(ctx, folder, contentType)
	if err != nil {
		return nil, fmt.Errorf("presign upload: %w", err)
	}

	return &PresignResult{
		Key:       key,
		UploadURL: uploadURL,
	}, nil
}

// GetURL returns a presigned download URL for a given key.
func (s *MediaService) GetURL(ctx context.Context, key string) (string, error) {
	return s.storage.PresignedURL(ctx, key)
}

// Delete removes a media object — only the owner should call this.
func (s *MediaService) Delete(ctx context.Context, key, userID string) error {
	// Verify the key belongs to this user.
	if !strings.HasPrefix(key, fmt.Sprintf("users/%s/", userID)) {
		return errors.New("forbidden: key does not belong to user")
	}
	return s.storage.Delete(ctx, key)
}

func validateType(ct string) error {
	if allowedImageTypes[ct] || allowedVideoTypes[ct] {
		return nil
	}
	return ErrInvalidType
}

func validateSize(ct string, size int64) error {
	if allowedImageTypes[ct] && size > maxImageSize {
		return ErrFileTooLarge
	}
	if allowedVideoTypes[ct] && size > maxVideoSize {
		return ErrFileTooLarge
	}
	return nil
}

func folderForUser(userID, contentType string) string {
	if allowedVideoTypes[contentType] {
		return fmt.Sprintf("users/%s/videos", userID)
	}
	return fmt.Sprintf("users/%s/images", userID)
}
