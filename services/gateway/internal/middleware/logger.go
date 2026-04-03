package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func RequestLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := uuid.NewString()
			w.Header().Set("X-Request-Id", reqID)
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rec, r)
			log.Info("request",
				zap.String("request_id", reqID),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rec.status),
				zap.Duration("latency", time.Since(start)),
			)
		})
	}
}
