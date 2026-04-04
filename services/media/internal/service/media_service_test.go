package service_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/mimsewelt/1984/services/media/internal/service"
)

// ── Fake storage ──────────────────────────────────────────────────────────────

type fakeStorage struct {
	uploaded map[string][]byte
	urls     map[string]string
	deleted  []string
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{
		uploaded: make(map[string][]byte),
		urls:     make(map[string]string),
	}
}

func (f *fakeStorage) Upload(_ context.Context, r io.Reader, size int64, contentType, folder string) (string, error) {
	data, _ := io.ReadAll(r)
	key := folder + "/test-file.jpg"
	f.uploaded[key] = data
	f.urls[key] = "https://minio.example.com/" + key
	return key, nil
}

func (f *fakeStorage) PresignedURL(_ context.Context, key string) (string, error) {
	url, ok := f.urls[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return url, nil
}

func (f *fakeStorage) PresignedUploadURL(_ context.Context, folder, contentType string) (string, string, error) {
	key := folder + "/presigned-file.jpg"
	return key, "https://minio.example.com/upload/" + key, nil
}

func (f *fakeStorage) Delete(_ context.Context, key string) error {
	f.deleted = append(f.deleted, key)
	return nil
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func newTestService() *service.MediaService {
	return service.NewMediaService(newFakeStorage())
}

func TestUpload_Image_Success(t *testing.T) {
	svc := newTestService()
	data := bytes.NewReader([]byte("fake image data"))

	result, err := svc.Upload(context.Background(), data, int64(data.Len()), "image/jpeg", "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Key == "" {
		t.Error("expected non-empty key")
	}
	if result.URL == "" {
		t.Error("expected non-empty URL")
	}
	if result.ContentType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", result.ContentType)
	}
}

func TestUpload_InvalidType_Rejected(t *testing.T) {
	svc := newTestService()
	data := bytes.NewReader([]byte("fake pdf"))

	_, err := svc.Upload(context.Background(), data, int64(data.Len()), "application/pdf", "user-1")
	if !errors.Is(err, service.ErrInvalidType) {
		t.Errorf("expected ErrInvalidType, got %v", err)
	}
}

func TestUpload_ImageTooLarge_Rejected(t *testing.T) {
	svc := newTestService()
	data := bytes.NewReader([]byte("x"))
	tooBig := int64(11 << 20) // 11 MB > 10 MB limit

	_, err := svc.Upload(context.Background(), data, tooBig, "image/jpeg", "user-1")
	if !errors.Is(err, service.ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestUpload_VideoTooLarge_Rejected(t *testing.T) {
	svc := newTestService()
	data := bytes.NewReader([]byte("x"))
	tooBig := int64(101 << 20) // 101 MB > 100 MB limit

	_, err := svc.Upload(context.Background(), data, tooBig, "video/mp4", "user-1")
	if !errors.Is(err, service.ErrFileTooLarge) {
		t.Errorf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestUpload_AllowedTypes(t *testing.T) {
	svc := newTestService()
	types := []string{"image/jpeg", "image/png", "image/webp", "image/gif", "video/mp4", "video/webm"}

	for _, ct := range types {
		data := bytes.NewReader([]byte("data"))
		_, err := svc.Upload(context.Background(), data, int64(data.Len()), ct, "user-1")
		if err != nil {
			t.Errorf("expected %s to be allowed, got %v", ct, err)
		}
	}
}

func TestUpload_KeyContainsUserID(t *testing.T) {
	svc := newTestService()
	data := bytes.NewReader([]byte("data"))

	result, err := svc.Upload(context.Background(), data, int64(data.Len()), "image/jpeg", "user-abc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Key, "user-abc") {
		t.Errorf("expected key to contain user-abc, got %s", result.Key)
	}
}

func TestRequestPresignedUpload_Success(t *testing.T) {
	svc := newTestService()

	result, err := svc.RequestPresignedUpload(context.Background(), "image/jpeg", "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Key == "" {
		t.Error("expected non-empty key")
	}
	if result.UploadURL == "" {
		t.Error("expected non-empty upload URL")
	}
}

func TestRequestPresignedUpload_InvalidType(t *testing.T) {
	svc := newTestService()
	_, err := svc.RequestPresignedUpload(context.Background(), "application/pdf", "user-1")
	if !errors.Is(err, service.ErrInvalidType) {
		t.Errorf("expected ErrInvalidType, got %v", err)
	}
}

func TestDelete_OwnFile_Success(t *testing.T) {
	svc := newTestService()
	data := bytes.NewReader([]byte("data"))

	result, _ := svc.Upload(context.Background(), data, int64(data.Len()), "image/jpeg", "user-1")
	err := svc.Delete(context.Background(), result.Key, "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDelete_OtherUserFile_Forbidden(t *testing.T) {
	svc := newTestService()
	data := bytes.NewReader([]byte("data"))

	result, _ := svc.Upload(context.Background(), data, int64(data.Len()), "image/jpeg", "user-1")
	err := svc.Delete(context.Background(), result.Key, "user-2")
	if err == nil {
		t.Error("expected error when deleting another user's file")
	}
}

func TestVideoUpload_StoredInVideosFolder(t *testing.T) {
	svc := newTestService()
	data := bytes.NewReader([]byte("video data"))

	result, err := svc.Upload(context.Background(), data, int64(data.Len()), "video/mp4", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Key, "videos") {
		t.Errorf("expected video key to contain 'videos', got %s", result.Key)
	}
}

func TestImageUpload_StoredInImagesFolder(t *testing.T) {
	svc := newTestService()
	data := bytes.NewReader([]byte("image data"))

	result, err := svc.Upload(context.Background(), data, int64(data.Len()), "image/jpeg", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Key, "images") {
		t.Errorf("expected image key to contain 'images', got %s", result.Key)
	}
}
