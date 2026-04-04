FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY shared/go.mod shared/go.sum ./shared/
COPY services/users/go.mod services/users/go.sum ./services/users/

RUN go work sync

COPY shared/ ./shared/
COPY services/users/ ./services/users/

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
