FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY shared/go.mod shared/go.sum ./shared/
COPY tools/migrate/go.mod tools/migrate/go.sum ./tools/migrate/

RUN go work sync

COPY shared/ ./shared/
COPY tools/migrate/ ./tools/migrate/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /bin/migrate \
    ./tools/migrate/

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /bin/migrate .
COPY migrations/ ./migrations/

CMD ["./migrate", "up"]
