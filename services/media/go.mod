module github.com/mimsewelt/1984/services/media

go 1.23

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/minio/minio-go/v7 v7.0.70
	github.com/mimsewelt/1984/shared v0.0.0
	go.uber.org/zap v1.27.0
)

replace github.com/mimsewelt/1984/shared => ../../shared
