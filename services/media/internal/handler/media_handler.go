package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mimsewelt/1984/services/media/internal/service"
	"github.com/mimsewelt/1984/shared/response"
	"go.uber.org/zap"
)

type MediaHandler struct {
	svc *service.MediaService
	log *zap.Logger
}

func NewMediaHandler(svc *service.MediaService, log *zap.Logger) *MediaHandler {
	return &MediaHandler{svc: svc, log: log}
}

// POST /media/upload
// Direct upload — client sends file as multipart form.
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		response.Unauthorized(w)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.BadRequest(w, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "file field required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	result, err := h.svc.Upload(r.Context(), file, header.Size, contentType, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidType):
			response.BadRequest(w, "unsupported file type")
		case errors.Is(err, service.ErrFileTooLarge):
			response.Error(w, http.StatusRequestEntityTooLarge, "file too large")
		default:
			h.log.Error("upload error", zap.Error(err))
			response.InternalError(w)
		}
		return
	}

	response.Created(w, result)
}

// POST /media/presign
// Returns a presigned URL for direct client-to-MinIO upload.
// Body: {"content_type": "image/jpeg"}
func (h *MediaHandler) RequestPresignedUpload(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		response.Unauthorized(w)
		return
	}

	var body struct {
		ContentType string `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ContentType == "" {
		response.BadRequest(w, "content_type required")
		return
	}

	result, err := h.svc.RequestPresignedUpload(r.Context(), body.ContentType, userID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidType) {
			response.BadRequest(w, "unsupported file type")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, result)
}

// GET /media/url/{key}
// Returns a fresh presigned download URL.
func (h *MediaHandler) GetURL(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	if key == "" {
		response.BadRequest(w, "key required")
		return
	}

	url, err := h.svc.GetURL(r.Context(), key)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"url": url})
}

// DELETE /media/{key}
func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		response.Unauthorized(w)
		return
	}

	key := chi.URLParam(r, "*")
	if err := h.svc.Delete(r.Context(), key, userID); err != nil {
		if err.Error() == "forbidden: key does not belong to user" {
			response.Forbidden(w)
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"message": "deleted"})
}

// GET /media/health — includes MinIO connectivity check via service
func (h *MediaHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// keep strconv import used by potential future handlers
var _ = strconv.Itoa
