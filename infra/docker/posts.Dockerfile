FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY shared/go.mod shared/go.sum ./shared/
COPY services/posts/go.mod services/posts/go.sum ./services/posts/

COPY shared/ ./shared/
COPY services/posts/ ./services/posts/

RUN go work init && \
    go work use ./shared ./services/posts && \
    go work sync

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /bin/posts \
    ./services/posts/cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
WORKDIR /app
COPY --from=builder /bin/posts .
EXPOSE 9002
CMD ["./posts"]
