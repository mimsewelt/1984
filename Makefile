.PHONY: test test-auth test-gateway test-messaging test-all lint

## Run all tests across all services
test-all:
	@echo "=== Auth Service ==="
	cd services/auth && go test ./... -v -count=1
	@echo ""
	@echo "=== Gateway ==="
	cd services/gateway && go test ./... -v -count=1
	@echo ""
	@echo "=== Messaging (Signal Protocol crypto) ==="
	cd services/messaging && go test ./... -v -count=1

## Individual service tests
test-auth:
	cd services/auth && go test ./... -v -count=1 -race

test-gateway:
	cd services/gateway && go test ./... -v -count=1 -race

test-messaging:
	cd services/messaging && go test ./... -v -count=1 -race

## Run with coverage report
test-coverage:
	cd services/auth && go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: services/auth/coverage.html"

## Lint (requires golangci-lint)
lint:
	golangci-lint run ./services/...

## Tidy all modules
tidy:
	cd shared && go mod tidy
	cd services/auth && go mod tidy
	cd services/gateway && go mod tidy
	cd services/messaging && go mod tidy

## Build all services
build:
	cd services/gateway && go build -o ../../bin/gateway ./cmd/
	cd services/auth    && go build -o ../../bin/auth    ./cmd/
	@echo "Binaries in ./bin/"