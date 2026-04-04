FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY shared/go.mod shared/go.sum ./shared/
COPY services/users/go.mod services/users/go.sum ./services/users/

COPY shared/ ./shared/
COPY services/users/ ./services/users/

RUN go work init && \
    go work use ./shared ./services/users && \
    go work sync

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /bin/users \
    ./services/users/cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
WORKDIR /app
COPY --from=builder /bin/users .
EXPOSE 9003
CMD ["./users"]
