package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	defaultBucket = "instagram-media"
	presignExpiry = 7 * 24 * time.Hour
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOStorage(endpoint, accessKey, secretKey string, useSSL bool) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio connect: %w", err)
	}
	return &MinIOStorage{client: client, bucket: defaultBucket}, nil
}

// EnsureBucket creates the bucket if it doesn't exist.
func (s *MinIOStorage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("bucket exists check: %w", err)
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

// Upload stores a file and returns its object key.
func (s *MinIOStorage) Upload(ctx context.Context, r io.Reader, size int64, contentType, folder string) (string, error) {
	ext := extensionFromContentType(contentType)
	key := fmt.Sprintf("%s/%s%s", folder, uuid.NewString(), ext)

	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("minio upload: %w", err)
	}
	return key, nil
}

// PresignedURL returns a time-limited URL to access an object.
func (s *MinIOStorage) PresignedURL(ctx context.Context, key string) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, presignExpiry, nil)
	if err != nil {
		return "", fmt.Errorf("presign: %w", err)
	}
	return u.String(), nil
}

// PresignedUploadURL returns a presigned PUT URL so clients upload directly to MinIO.
func (s *MinIOStorage) PresignedUploadURL(ctx context.Context, folder, contentType string) (string, string, error) {
	ext := extensionFromContentType(contentType)
	key := fmt.Sprintf("%s/%s%s", folder, uuid.NewString(), ext)

	u, err := s.client.PresignedPutObject(ctx, s.bucket, key, presignExpiry)
	if err != nil {
		return "", "", fmt.Errorf("presign upload: %w", err)
	}
	return key, u.String(), nil
}

// Delete removes an object.
func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func extensionFromContentType(ct string) string {
	switch ct {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	default:
		return ""
	}
}
