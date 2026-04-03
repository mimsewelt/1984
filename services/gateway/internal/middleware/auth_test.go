package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mimsewelt/1984/services/gateway/internal/middleware"
)

const testSecret = "test-secret-minimum-32-characters!!"

func makeToken(t *testing.T, userID, username string, exp time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      exp.Unix(),
		"iat":      time.Now().Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to create test token: %v", err)
	}
	return tok
}

func applyMiddleware(secret string, next http.HandlerFunc) http.Handler {
	return middleware.Authenticate(secret)(next)
}

// ── Valid token tests ─────────────────────────────────────────────────────────

func TestAuthenticate_ValidToken_Passes(t *testing.T) {
	called := false
	handler := applyMiddleware(testSecret, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	token := makeToken(t, "user-123", "alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected handler to be called with valid token")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAuthenticate_InjectsUserIDIntoContext(t *testing.T) {
	var capturedUID string
	handler := applyMiddleware(testSecret, func(w http.ResponseWriter, r *http.Request) {
		uid, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			t.Error("user_id not found in context")
		}
		capturedUID = uid
		w.WriteHeader(http.StatusOK)
	})

	token := makeToken(t, "user-abc", "bob", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if capturedUID != "user-abc" {
		t.Errorf("expected user-abc in context, got %q", capturedUID)
	}
}

// ── Missing / malformed token tests ──────────────────────────────────────────

func TestAuthenticate_MissingHeader_Returns401(t *testing.T) {
	handler := applyMiddleware(testSecret, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without a token")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthenticate_BearerPrefixMissing_Returns401(t *testing.T) {
	handler := applyMiddleware(testSecret, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	token := makeToken(t, "u1", "alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", token) // missing "Bearer " prefix
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthenticate_ExpiredToken_Returns401(t *testing.T) {
	handler := applyMiddleware(testSecret, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with expired token")
	})

	token := makeToken(t, "u1", "alice", time.Now().Add(-time.Hour)) // already expired
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthenticate_WrongSecret_Returns401(t *testing.T) {
	handler := applyMiddleware(testSecret, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	// Token signed with a different secret.
	token := makeToken(t, "u1", "alice", time.Now().Add(time.Hour))
	wrongHandler := applyMiddleware("wrong-secret-also-minimum-32-chars!!", func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})
	_ = token

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, "u1", "a", time.Now().Add(time.Hour)))
	rr := httptest.NewRecorder()
	wrongHandler.ServeHTTP(rr, req)

	// Reuse original handler with token signed by wrong secret.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	claims := jwt.MapClaims{"user_id": "u1", "username": "a", "exp": time.Now().Add(time.Hour).Unix()}
	badTok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("different-secret-32-characters!!!"))
	req2.Header.Set("Authorization", "Bearer "+badTok)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong secret, got %d", rr2.Code)
	}
}

func TestAuthenticate_GarbageToken_Returns401(t *testing.T) {
	handler := applyMiddleware(testSecret, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.jwt")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

// ── Algorithm confusion attack ────────────────────────────────────────────────

func TestAuthenticate_AlgNone_Rejected(t *testing.T) {
	// "alg: none" attack — unsigned token claiming to be valid.
	handler := applyMiddleware(testSecret, func(w http.ResponseWriter, r *http.Request) {
		t.Error("alg:none token must never be accepted")
	})

	// Manually craft an unsigned JWT.
	header := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0"               // {"alg":"none","typ":"JWT"}
	payload := "eyJ1c2VyX2lkIjoiYWRtaW4iLCJleHAiOjk5OTk5OTk5OTl9" // {"user_id":"admin","exp":9999999999}
	noneToken := header + "." + payload + "."

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+noneToken)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("alg:none attack succeeded — expected 401, got %d", rr.Code)
	}
}
