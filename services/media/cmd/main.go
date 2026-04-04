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
	"github.com/mimsewelt/1984/services/media/internal/handler"
	"github.com/mimsewelt/1984/services/media/internal/service"
	"github.com/mimsewelt/1984/services/media/internal/storage"
	"github.com/mimsewelt/1984/shared/logger"
	"go.uber.org/zap"
)

func main() {
	log := logger.New()
	defer logger.Sync()

	minioEndpoint  := getEnv("MINIO_ENDPOINT",   "minio:9000")
	minioAccessKey := mustEnv("MINIO_ACCESS_KEY")
	minioSecretKey := mustEnv("MINIO_SECRET_KEY")
	minioSSL       := getEnv("MINIO_USE_SSL", "false") == "true"
	port           := getEnv("PORT", "9004")

	store, err := storage.NewMinIOStorage(minioEndpoint, minioAccessKey, minioSecretKey, minioSSL)
	if err != nil {
		log.Fatal("minio connect failed", zap.Error(err))
	}

	ctx := context.Background()
	if err := store.EnsureBucket(ctx); err != nil {
		log.Fatal("bucket setup failed", zap.Error(err))
	}
	log.Info("minio ready", zap.String("endpoint", minioEndpoint))

	mediaSvc := service.NewMediaService(store)
	mediaH   := handler.NewMediaHandler(mediaSvc, log)

	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))

	r.Get("/health",          mediaH.Health)
	r.Post("/media/upload",   mediaH.Upload)
	r.Post("/media/presign",  mediaH.RequestPresignedUpload)
	r.Get("/media/url/*",     mediaH.GetURL)
	r.Delete("/media/*",      mediaH.Delete)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		log.Info("media service listening", zap.String("addr", srv.Addr))
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
	log.Info("media service stopped")
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
