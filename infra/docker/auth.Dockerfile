FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.work and all module files first for better cache
COPY go.work go.work.sum ./
COPY shared/go.mod shared/go.sum ./shared/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/

# Download dependencies
RUN go work sync

# Copy source
COPY shared/ ./shared/
COPY services/auth/ ./services/auth/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /bin/auth \
    ./services/auth/cmd/

# ── Runtime image ─────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates curl

WORKDIR /app
COPY --from=builder /bin/auth .

EXPOSE 9001
CMD ["./auth"]
