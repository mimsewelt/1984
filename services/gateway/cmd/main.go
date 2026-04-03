package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/mimsewelt/1984/services/gateway/internal/config"
	"github.com/mimsewelt/1984/services/gateway/internal/handler"
	"github.com/mimsewelt/1984/services/gateway/internal/middleware"
	"github.com/mimsewelt/1984/shared/logger"
	"go.uber.org/zap"
)

func main() {
	log := logger.New()
	defer logger.Sync()

	cfg := config.Load()
	r := chi.NewRouter()

	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.RequestLogger(log))
	r.Use(chimw.StripSlashes)
	r.Use(httprate.LimitByIP(cfg.RateLimitRequests, cfg.RateLimitWindow))

	r.Get("/health", handler.Health)
	r.Mount("/api/v1/auth", handler.Proxy(cfg.AuthServiceURL, "/api/v1/auth", log))

	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(cfg.JWTSecret))
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
					r.Header.Set("X-User-Id", uid)
				}
				next.ServeHTTP(w, r)
			})
		})
		r.Mount("/api/v1/posts",    handler.Proxy(cfg.PostsServiceURL, "/api/v1/posts", log))
		r.Mount("/api/v1/users",    handler.Proxy(cfg.UsersServiceURL, "/api/v1/users", log))
		r.Mount("/api/v1/media",    handler.Proxy(cfg.MediaServiceURL, "/api/v1/media", log))
		r.Mount("/api/v1/messages", handler.Proxy(cfg.MsgServiceURL, "/api/v1/messages", log))
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		log.Info("gateway listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("gateway stopped")
}
