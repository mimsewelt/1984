package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mimsewelt/1984/services/auth/internal/model"
	"github.com/mimsewelt/1984/services/auth/internal/repository"
	"github.com/mimsewelt/1984/services/auth/internal/service"
	"golang.org/x/crypto/bcrypt"
)

// ── In-memory fakes (no DB required) ────────────────────────────────────────

type fakeUserRepo struct {
	users map[string]*model.User // keyed by email
	byID  map[string]*model.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users: make(map[string]*model.User),
		byID:  make(map[string]*model.User),
	}
}

func (f *fakeUserRepo) Create(_ context.Context, u *model.User) error {
	if _, exists := f.users[u.Email]; exists {
		return repository.ErrConflict
	}
	f.users[u.Email] = u
	f.byID[u.ID] = u
	return nil
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (*model.User, error) {
	u, ok := f.users[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) FindByID(_ context.Context, id string) (*model.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

type fakeTokenRepo struct {
	tokens map[string]*model.RefreshToken // key: userID+":"+deviceID
}

func newFakeTokenRepo() *fakeTokenRepo {
	return &fakeTokenRepo{tokens: make(map[string]*model.RefreshToken)}
}

func (f *fakeTokenRepo) Save(_ context.Context, t *model.RefreshToken) error {
	f.tokens[t.UserID+":"+t.DeviceID] = t
	return nil
}

func (f *fakeTokenRepo) FindByUserAndDevice(_ context.Context, userID, deviceID string) (*model.RefreshToken, error) {
	t, ok := f.tokens[userID+":"+deviceID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	if t.ExpiresAt.Before(time.Now()) {
		return nil, repository.ErrNotFound
	}
	return t, nil
}

func (f *fakeTokenRepo) Delete(_ context.Context, id string) error {
	for k, t := range f.tokens {
		if t.ID == id {
			delete(f.tokens, k)
			return nil
		}
	}
	return nil
}

func (f *fakeTokenRepo) DeleteExpired(_ context.Context) error { return nil }

// ── Helpers ──────────────────────────────────────────────────────────────────

func newTestService() *service.AuthService {
	return service.NewAuthService(
		newFakeUserRepo(),
		newFakeTokenRepo(),
		"test-secret-minimum-32-characters!!",
	)
}

func validRegisterReq() *model.RegisterRequest {
	return &model.RegisterRequest{
		Username:    "testuser",
		Email:       "test@example.com",
		Password:    "SuperSecret123!",
		DisplayName: "Test User",
	}
}

// ── Register tests ────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	svc := newTestService()
	resp, err := svc.Register(context.Background(), validRegisterReq())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if resp.User.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", resp.User.Email)
	}
	if resp.ExpiresIn <= 0 {
		t.Error("expected positive expires_in")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newTestService()
	req := validRegisterReq()

	if _, err := svc.Register(context.Background(), req); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	_, err := svc.Register(context.Background(), req)
	if !errors.Is(err, service.ErrUserExists) {
		t.Errorf("expected ErrUserExists, got %v", err)
	}
}

// ── Login tests ───────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	svc := newTestService()
	req := validRegisterReq()

	if _, err := svc.Register(context.Background(), req); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	resp, err := svc.Login(context.Background(), &model.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
		DeviceID: "iphone-14",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := newTestService()
	if _, err := svc.Register(context.Background(), validRegisterReq()); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Login(context.Background(), &model.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	})
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := newTestService()
	_, err := svc.Login(context.Background(), &model.LoginRequest{
		Email:    "nobody@example.com",
		Password: "anything",
	})
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

// ── Refresh token tests ───────────────────────────────────────────────────────

func TestRefresh_RotatesToken(t *testing.T) {
	svc := newTestService()
	if _, err := svc.Register(context.Background(), validRegisterReq()); err != nil {
		t.Fatal(err)
	}

	loginResp, err := svc.Login(context.Background(), &model.LoginRequest{
		Email:    "test@example.com",
		Password: "SuperSecret123!",
		DeviceID: "device-abc",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	newResp, err := svc.Refresh(context.Background(), loginResp.RefreshToken, "device-abc")
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if newResp.AccessToken == loginResp.AccessToken {
		t.Error("expected a new access token after refresh")
	}
	if newResp.RefreshToken == loginResp.RefreshToken {
		t.Error("expected a new refresh token after rotation")
	}
}

func TestRefresh_OldTokenRejectedAfterRotation(t *testing.T) {
	svc := newTestService()
	if _, err := svc.Register(context.Background(), validRegisterReq()); err != nil {
		t.Fatal(err)
	}

	loginResp, _ := svc.Login(context.Background(), &model.LoginRequest{
		Email:    "test@example.com",
		Password: "SuperSecret123!",
		DeviceID: "device-abc",
	})

	// First refresh consumes the token.
	if _, err := svc.Refresh(context.Background(), loginResp.RefreshToken, "device-abc"); err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}

	// Second refresh with the same (now rotated-out) token must fail.
	_, err := svc.Refresh(context.Background(), loginResp.RefreshToken, "device-abc")
	if !errors.Is(err, service.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired on reuse, got %v", err)
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc := newTestService()
	_, err := svc.Refresh(context.Background(), "not-a-valid-token", "device-x")
	if !errors.Is(err, service.ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

// ── Password hashing tests ────────────────────────────────────────────────────

func TestPasswordHash_NotStoredPlaintext(t *testing.T) {
	userRepo := newFakeUserRepo()
	tokenRepo := newFakeTokenRepo()
	svc := service.NewAuthService(userRepo, tokenRepo, "test-secret-minimum-32-characters!!")

	req := validRegisterReq()
	if _, err := svc.Register(context.Background(), req); err != nil {
		t.Fatal(err)
	}

	stored, err := userRepo.FindByEmail(context.Background(), req.Email)
	if err != nil {
		t.Fatal(err)
	}
	if stored.PasswordHash == req.Password {
		t.Error("password stored as plaintext — this is a critical security bug")
	}
	// Verify it is a valid bcrypt hash.
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(req.Password)); err != nil {
		t.Errorf("stored hash does not match original password: %v", err)
	}
}

func TestUserID_IsUUID(t *testing.T) {
	userRepo := newFakeUserRepo()
	tokenRepo := newFakeTokenRepo()
	svc := service.NewAuthService(userRepo, tokenRepo, "test-secret-minimum-32-characters!!")

	resp, err := svc.Register(context.Background(), validRegisterReq())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(resp.User.ID); err != nil {
		t.Errorf("user ID is not a valid UUID: %s", resp.User.ID)
	}
}
