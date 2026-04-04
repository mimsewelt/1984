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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mimsewelt/1984/services/posts/internal/handler"
	"github.com/mimsewelt/1984/services/posts/internal/repository"
	"github.com/mimsewelt/1984/services/posts/internal/service"
	"github.com/mimsewelt/1984/shared/logger"
	"go.uber.org/zap"
)

func main() {
	log := logger.New()
	defer logger.Sync()

	dbURL := mustEnv("DATABASE_URL")
	port  := getEnv("PORT", "9002")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("db connect failed", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("db ping failed", zap.Error(err))
	}

	postRepo := repository.NewPostRepository(pool)
	likeRepo := repository.NewLikeRepository(pool)
	postSvc  := service.NewPostService(postRepo, likeRepo)
	postH    := handler.NewPostHandler(postSvc, log)

	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Post("/posts",             postH.CreatePost)
	r.Get("/posts/{id}",         postH.GetPost)
	r.Delete("/posts/{id}",      postH.DeletePost)
	r.Post("/posts/{id}/like",   postH.LikePost)
	r.Delete("/posts/{id}/like", postH.UnlikePost)
	r.Get("/feed",               postH.GetFeed)
	r.Get("/users/{id}/posts",   postH.GetUserPosts)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("posts service listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
	log.Info("posts service stopped")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var not set: " + key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
