package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mimsewelt/1984/services/users/internal/model"
	"github.com/mimsewelt/1984/services/users/internal/service"
	"github.com/mimsewelt/1984/shared/response"
	"go.uber.org/zap"
)

type UserHandler struct {
	svc *service.UserService
	log *zap.Logger
}

func NewUserHandler(svc *service.UserService, log *zap.Logger) *UserHandler {
	return &UserHandler{svc: svc, log: log}
}

// GET /users/{id}
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	viewerID := r.Header.Get("X-User-Id")

	profile, err := h.svc.GetProfile(r.Context(), targetID, viewerID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, profile)
}

// GET /users/by-username/{username}
func (h *UserHandler) GetProfileByUsername(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	viewerID := r.Header.Get("X-User-Id")

	profile, err := h.svc.GetProfileByUsername(r.Context(), username, viewerID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, profile)
}

// PATCH /users/me
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		response.Unauthorized(w)
		return
	}
	var req model.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	profile, err := h.svc.UpdateProfile(r.Context(), userID, &req)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, profile)
}

// POST /users/{id}/follow
func (h *UserHandler) Follow(w http.ResponseWriter, r *http.Request) {
	followerID := r.Header.Get("X-User-Id")
	followingID := chi.URLParam(r, "id")
	if followerID == "" {
		response.Unauthorized(w)
		return
	}
	err := h.svc.Follow(r.Context(), followerID, followingID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			response.NotFound(w)
		case errors.Is(err, service.ErrCannotFollowSelf):
			response.BadRequest(w, "cannot follow yourself")
		case errors.Is(err, service.ErrAlreadyFollowing):
			response.Error(w, http.StatusConflict, "already following")
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, map[string]string{"message": "followed"})
}

// DELETE /users/{id}/follow
func (h *UserHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	followerID := r.Header.Get("X-User-Id")
	followingID := chi.URLParam(r, "id")
	if followerID == "" {
		response.Unauthorized(w)
		return
	}
	err := h.svc.Unfollow(r.Context(), followerID, followingID)
	if err != nil {
		if errors.Is(err, service.ErrNotFollowing) {
			response.Error(w, http.StatusConflict, "not following")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]string{"message": "unfollowed"})
}

// GET /users/{id}/followers
func (h *UserHandler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	cursor := r.URL.Query().Get("cursor")

	list, err := h.svc.GetFollowers(r.Context(), userID, cursor)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, list)
}

// GET /users/{id}/following
func (h *UserHandler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	cursor := r.URL.Query().Get("cursor")

	list, err := h.svc.GetFollowing(r.Context(), userID, cursor)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, list)
}
