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
 "github.com/yourorg/instagram-clone/services/gateway/internal/config"
 "github.com/yourorg/instagram-clone/services/gateway/internal/handler"
 "github.com/yourorg/instagram-clone/services/gateway/internal/middleware"
 "github.com/yourorg/instagram-clone/shared/logger"
 "go.uber.org/zap"
)

func main() {
 log := logger.New()
 defer logger.Sync()

 cfg := config.Load()

 r := chi.NewRouter()

 // ── Global middleware ────────────────────────────────────────────────────
 r.Use(chimw.RealIP)
 r.Use(chimw.Recoverer)
 r.Use(middleware.RequestLogger(log))
 r.Use(chimw.StripSlashes)

 // Global rate limit: 100 req / 60s per IP (overridable via env)
 r.Use(httprate.LimitByIP(cfg.RateLimitRequests, cfg.RateLimitWindow))

 // ── Public routes (no auth) ──────────────────────────────────────────────
 r.Get("/health", handler.Health)

 // Auth service — register, login, refresh (no JWT required)
 r.Mount("/api/v1/auth", handler.Proxy(cfg.AuthServiceURL, "/api/v1/auth", log))

 // ── Protected routes ─────────────────────────────────────────────────────
 r.Group(func(r chi.Router) {
  r.Use(middleware.Authenticate(cfg.JWTSecret))

  // Inject user_id header so downstream services trust it without re-parsing JWT.
  r.Use(func(next http.Handler) http.Handler {
   return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if uid, ok := middleware.UserIDFromContext(r.Context()); ok {
     r.Header.Set("X-User-Id", uid)
    }
    next.ServeHTTP(w, r)
   })
  })

  r.Mount("/api/v1/posts",     handler.Proxy(cfg.PostsServiceURL, "/api/v1/posts", log))
  r.Mount("/api/v1/users",     handler.Proxy(cfg.UsersServiceURL, "/api/v1/users", log))
  r.Mount("/api/v1/media",     handler.Proxy(cfg.MediaServiceURL, "/api/v1/media", log))
  r.Mount("/api/v1/messages",  handler.Proxy(cfg.MsgServiceURL,   "/api/v1/messages", log))
 })

 // ── HTTP server with graceful shutdown ───────────────────────────────────
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

 log.Info("shutting down gracefully…")
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()

 if err := srv.Shutdown(ctx); err != nil {
  log.Error("shutdown error", zap.Error(err))
 }
 log.Info("gateway stopped")
}