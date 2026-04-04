package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mimsewelt/1984/services/posts/internal/model"
	"github.com/mimsewelt/1984/services/posts/internal/service"
	"github.com/mimsewelt/1984/shared/response"
	"go.uber.org/zap"
)

type PostHandler struct {
	svc *service.PostService
	log *zap.Logger
}

func NewPostHandler(svc *service.PostService, log *zap.Logger) *PostHandler {
	return &PostHandler{svc: svc, log: log}
}

// POST /posts
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		response.Unauthorized(w)
		return
	}
	var req model.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	resp, err := h.svc.CreatePost(r.Context(), userID, &req)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.Created(w, resp)
}

// GET /posts/{id}
func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")
	viewerID := r.Header.Get("X-User-Id")

	resp, err := h.svc.GetPost(r.Context(), postID, viewerID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, resp)
}

// DELETE /posts/{id}
func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		response.Unauthorized(w)
		return
	}
	if err := h.svc.DeletePost(r.Context(), postID, userID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.NotFound(w)
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]string{"message": "deleted"})
}

// GET /feed
func (h *PostHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		response.Unauthorized(w)
		return
	}
	cursor := r.URL.Query().Get("cursor")
	feed, err := h.svc.GetFeed(r.Context(), userID, cursor)
	if err != nil {
		h.log.Error("get feed error", zap.Error(err))
		response.InternalError(w)
		return
	}
	response.OK(w, feed)
}

// GET /users/{id}/posts
func (h *PostHandler) GetUserPosts(w http.ResponseWriter, r *http.Request) {
	targetUserID := chi.URLParam(r, "id")
	viewerID := r.Header.Get("X-User-Id")
	cursor := r.URL.Query().Get("cursor")

	feed, err := h.svc.GetUserPosts(r.Context(), targetUserID, viewerID, cursor)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, feed)
}

// POST /posts/{id}/like
func (h *PostHandler) LikePost(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		response.Unauthorized(w)
		return
	}
	if err := h.svc.LikePost(r.Context(), postID, userID); err != nil {
		if errors.Is(err, service.ErrAlreadyLiked) {
			response.Error(w, http.StatusConflict, "already liked")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]string{"message": "liked"})
}

// DELETE /posts/{id}/like
func (h *PostHandler) UnlikePost(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		response.Unauthorized(w)
		return
	}
	if err := h.svc.UnlikePost(r.Context(), postID, userID); err != nil {
		if errors.Is(err, service.ErrNotLiked) {
			response.Error(w, http.StatusConflict, "not liked")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, map[string]string{"message": "unliked"})
}
