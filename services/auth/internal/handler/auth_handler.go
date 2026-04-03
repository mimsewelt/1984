package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mimsewelt/1984/services/auth/internal/model"
	"github.com/mimsewelt/1984/services/auth/internal/service"
	"github.com/mimsewelt/1984/shared/response"
	"go.uber.org/zap"
)

type AuthHandler struct {
	svc *service.AuthService
	log *zap.Logger
}

func NewAuthHandler(svc *service.AuthService, log *zap.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, log: log}
}

// POST /register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := validateRegister(&req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	resp, err := h.svc.Register(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserExists):
			response.Error(w, http.StatusConflict, "username or email already taken")
		default:
			h.log.Error("register error", zap.Error(err))
			response.InternalError(w)
		}
		return
	}

	response.Created(w, resp)
}

// POST /login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	resp, err := h.svc.Login(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			// Always return the same message to prevent user enumeration.
			response.Error(w, http.StatusUnauthorized, "invalid email or password")
		default:
			h.log.Error("login error", zap.Error(err))
			response.InternalError(w)
		}
		return
	}

	response.OK(w, resp)
}

// POST /refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	deviceID := r.Header.Get("X-Device-Id")
	if deviceID == "" {
		deviceID = "web"
	}

	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken, deviceID)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	response.OK(w, resp)
}

// POST /logout — client simply discards tokens; server can optionally invalidate.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// In a full implementation: parse refresh token, delete from DB.
	response.OK(w, map[string]string{"message": "logged out"})
}

func validateRegister(req *model.RegisterRequest) error {
	if len(req.Username) < 3 || len(req.Username) > 30 {
		return errors.New("username must be 3–30 characters")
	}
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if req.Email == "" {
		return errors.New("email is required")
	}
	return nil
}
