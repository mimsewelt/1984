.PHONY: test-all test-auth test-gateway test-messaging test-posts test-users
.PHONY: migrate-up migrate-down docker-up docker-down docker-build docker-full dev
.PHONY: tidy build lint

test-all:
	@echo "=== Auth Service ==="
	cd services/auth && go test ./... -v -count=1
	@echo ""
	@echo "=== Gateway ==="
	cd services/gateway && go test ./... -v -count=1
	@echo ""
	@echo "=== Messaging ==="
	cd services/messaging && go test ./... -v -count=1
	@echo ""
	@echo "=== Posts ==="
	cd services/posts && go test ./... -v -count=1
	@echo ""
	@echo "=== Users ==="
	cd services/users && go test ./... -v -count=1

test-auth:
	cd services/auth && go test ./... -v -count=1 -race

test-gateway:
	cd services/gateway && go test ./... -v -count=1 -race

test-messaging:
	cd services/messaging && go test ./... -v -count=1 -race

test-posts:
	cd services/posts && go test ./... -v -count=1 -race

test-users:
	cd services/users && go test ./... -v -count=1 -race

test-coverage:
	cd services/auth && go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html

migrate-up:
	@export $(shell cat .env | xargs) && \
	DATABASE_URL=$$DATABASE_URL go run ./tools/migrate/main.go up

migrate-down:
	@export $(shell cat .env | xargs) && \
	DATABASE_URL=$$DATABASE_URL go run ./tools/migrate/main.go down

docker-up:
	docker compose up -d postgres redis minio
	@echo "Waiting for services to be healthy..."
	@sleep 3
	@echo "PostgreSQL, Redis, MinIO are running"

docker-build:
	docker compose build auth gateway

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-full:
	docker compose up -d

dev: docker-up migrate-up
	@echo ""
	@echo "Infrastructure ready!"
	@echo "PostgreSQL : localhost:5432"
	@echo "Redis      : localhost:6379"
	@echo "MinIO      : localhost:9000 (console: localhost:9001)"

tidy:
	cd shared && go mod tidy
	cd services/auth && go mod tidy
	cd services/gateway && go mod tidy
	cd services/messaging && go mod tidy
	cd services/posts && go mod tidy
	cd services/users && go mod tidy
	cd tools/migrate && go mod tidy

build:
	mkdir -p bin
	cd services/gateway && go build -o ../../bin/gateway ./cmd/
	cd services/auth    && go build -o ../../bin/auth    ./cmd/
	cd services/posts   && go build -o ../../bin/posts   ./cmd/
	cd services/users   && go build -o ../../bin/users   ./cmd/

lint:
	golangci-lint run ./services/...
