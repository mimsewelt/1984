package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yourorg/instagram-clone/services/auth/internal/handler"
	"github.com/yourorg/instagram-clone/services/auth/internal/model"
	"github.com/yourorg/instagram-clone/services/auth/internal/repository"
	"github.com/yourorg/instagram-clone/services/auth/internal/service"
	"github.com/yourorg/instagram-clone/shared/logger"
)

// ── Fakes (same pattern as service tests) ────────────────────────────────────

type fakeUserRepo struct {
	users map[string]*model.User
	byID  map[string]*model.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users: make(map[string]*model.User),
		byID:  make(map[string]*model.User),
	}
}

func (f *fakeUserRepo) Create(_ context.Context, u *model.User) error {
	if _, ok := f.users[u.Email]; ok {
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
	tokens map[string]*model.RefreshToken
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
	return t, nil
}

func (f *fakeTokenRepo) Delete(_ context.Context, id string) error {
	for k, t := range f.tokens {
		if t.ID == id {
			delete(f.tokens, k)
		}
	}
	return nil
}

func (f *fakeTokenRepo) DeleteExpired(_ context.Context) error { return nil }

// ── Test setup ────────────────────────────────────────────────────────────────

func newTestHandler() *handler.AuthHandler {
	svc := service.NewAuthService(
		newFakeUserRepo(),
		newFakeTokenRepo(),
		"test-secret-minimum-32-characters!!",
	)
	return handler.NewAuthHandler(svc, logger.New())
}

func postJSON(t *testing.T, h http.HandlerFunc, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return out
}

// ── Register handler tests ────────────────────────────────────────────────────

func TestRegisterHandler_Returns201(t *testing.T) {
	h := newTestHandler()
	rr := postJSON(t, h.Register, "/register", map[string]string{
		"username":     "alice",
		"email":        "alice@example.com",
		"password":     "Password123!",
		"display_name": "Alice",
	})
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestRegisterHandler_MissingFields_Returns400(t *testing.T) {
	h := newTestHandler()
	cases := []map[string]string{
		{"username": "a", "email": "x@x.com", "password": "short"},  // password too short
		{"username": "ab", "email": "x@x.com", "password": "LongEnough1!"},  // username too short
		{"username": "valid", "email": "", "password": "LongEnough1!"},       // missing email
	}
	for _, body := range cases {
		rr := postJSON(t, h.Register, "/register", body)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for body %v, got %d", body, rr.Code)
		}
	}
}

func TestRegisterHandler_DuplicateEmail_Returns409(t *testing.T) {
	h := newTestHandler()
	body := map[string]string{
		"username": "alice", "email": "alice@example.com",
		"password": "Password123!", "display_name": "Alice",
	}
	postJSON(t, h.Register, "/register", body)
	rr := postJSON(t, h.Register, "/register", body)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
}

func TestRegisterHandler_ResponseContainsTokens(t *testing.T) {
	h := newTestHandler()
	rr := postJSON(t, h.Register, "/register", map[string]string{
		"username": "bob", "email": "bob@example.com",
		"password": "Password123!", "display_name": "Bob",
	})
	body := decodeBody(t, rr)
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %v", body)
	}
	if data["access_token"] == "" || data["access_token"] == nil {
		t.Error("expected access_token in response")
	}
	if data["refresh_token"] == "" || data["refresh_token"] == nil {
		t.Error("expected refresh_token in response")
	}
}

// ── Login handler tests ───────────────────────────────────────────────────────

func TestLoginHandler_Success_Returns200(t *testing.T) {
	h := newTestHandler()
	postJSON(t, h.Register, "/register", map[string]string{
		"username": "charlie", "email": "charlie@example.com",
		"password": "Password123!", "display_name": "Charlie",
	})

	rr := postJSON(t, h.Login, "/login", map[string]string{
		"email": "charlie@example.com", "password": "Password123!",
	})
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}
}

func TestLoginHandler_WrongPassword_Returns401(t *testing.T) {
	h := newTestHandler()
	postJSON(t, h.Register, "/register", map[string]string{
		"username": "dave", "email": "dave@example.com",
		"password": "CorrectPassword1!", "display_name": "Dave",
	})

	rr := postJSON(t, h.Login, "/login", map[string]string{
		"email": "dave@example.com", "password": "WrongPassword!",
	})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestLoginHandler_UnknownUser_Returns401Not404(t *testing.T) {
	// Must return 401, not 404, to prevent user enumeration.
	h := newTestHandler()
	rr := postJSON(t, h.Login, "/login", map[string]string{
		"email": "ghost@example.com", "password": "whatever",
	})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 (not 404) for unknown user, got %d", rr.Code)
	}
}

func TestLoginHandler_ErrorMessageIsGeneric(t *testing.T) {
	// Error messages for wrong password and unknown user must be identical.
	h := newTestHandler()
	postJSON(t, h.Register, "/register", map[string]string{
		"username": "eve", "email": "eve@example.com",
		"password": "CorrectPass1!", "display_name": "Eve",
	})

	r1 := postJSON(t, h.Login, "/login", map[string]string{
		"email": "eve@example.com", "password": "wrong",
	})
	r2 := postJSON(t, h.Login, "/login", map[string]string{
		"email": "nobody@example.com", "password": "wrong",
	})

	b1 := decodeBody(t, r1)
	b2 := decodeBody(t, r2)

	if b1["error"] != b2["error"] {
		t.Errorf("user enumeration possible: wrong-pass=%q unknown=%q", b1["error"], b2["error"])
	}
}

// ── Content-Type tests ────────────────────────────────────────────────────────

func TestHandler_InvalidJSON_Returns400(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader([]byte("not-json{")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", rr.Code)
	}
}