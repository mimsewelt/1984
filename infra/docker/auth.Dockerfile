FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY shared/go.mod shared/go.sum ./shared/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/

COPY shared/ ./shared/
COPY services/auth/ ./services/auth/

RUN go work init && \
    go work use ./shared ./services/auth && \
    go work sync

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /bin/auth \
    ./services/auth/cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
WORKDIR /app
COPY --from=builder /bin/auth .
EXPOSE 9001
CMD ["./auth"]
