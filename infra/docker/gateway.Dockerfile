FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY shared/go.mod shared/go.sum ./shared/
COPY services/gateway/go.mod services/gateway/go.sum ./services/gateway/

RUN go work sync

COPY shared/ ./shared/
COPY services/gateway/ ./services/gateway/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /bin/gateway \
    ./services/gateway/cmd/

FROM alpine:3.19

RUN apk --no-cache add ca-certificates curl

WORKDIR /app
COPY --from=builder /bin/gateway .

EXPOSE 8080
CMD ["./gateway"]
