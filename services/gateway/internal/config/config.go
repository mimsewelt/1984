package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	JWTSecret       string
	AuthServiceURL  string
	PostsServiceURL string
	UsersServiceURL string
	MediaServiceURL string
	MsgServiceURL   string
	RateLimitRequests int
	RateLimitWindow   time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "8080"),
		JWTSecret:         mustEnv("JWT_SECRET"),
		AuthServiceURL:    getEnv("AUTH_SERVICE_URL", "http://auth:9001"),
		PostsServiceURL:   getEnv("POSTS_SERVICE_URL", "http://posts:9002"),
		UsersServiceURL:   getEnv("USERS_SERVICE_URL", "http://users:9003"),
		MediaServiceURL:   getEnv("MEDIA_SERVICE_URL", "http://media:9004"),
		MsgServiceURL:     getEnv("MSG_SERVICE_URL", "http://messaging:9005"),
		RateLimitRequests: getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:   time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SEC", 60)) * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required env var not set: " + key)
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
