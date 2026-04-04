.PHONY: test-all test-auth test-gateway test-messaging test-coverage migrate-up migrate-down tidy build lint

test-all:
	@echo "=== Auth Service ==="
	cd services/auth && go test ./... -v -count=1
	@echo ""
	@echo "=== Gateway ==="
	cd services/gateway && go test ./... -v -count=1
	@echo ""
	@echo "=== Messaging (Signal Protocol crypto) ==="
	cd services/messaging && go test ./... -v -count=1

test-auth:
	cd services/auth && go test ./... -v -count=1 -race

test-gateway:
	cd services/gateway && go test ./... -v -count=1 -race

test-messaging:
	cd services/messaging && go test ./... -v -count=1 -race

test-coverage:
	cd services/auth && go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html

migrate-up:
	DATABASE_URL=$(DATABASE_URL) go run ./tools/migrate/main.go up

migrate-down:
	DATABASE_URL=$(DATABASE_URL) go run ./tools/migrate/main.go down

tidy:
	cd shared && go mod tidy
	cd services/auth && go mod tidy
	cd services/gateway && go mod tidy
	cd services/messaging && go mod tidy
	cd tools/migrate && go mod tidy

build:
	cd services/gateway && go build -o ../../bin/gateway ./cmd/
	cd services/auth    && go build -o ../../bin/auth    ./cmd/

lint:
	golangci-lint run ./services/...
