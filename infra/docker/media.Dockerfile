FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY shared/go.mod shared/go.sum ./shared/
COPY services/media/go.mod services/media/go.sum ./services/media/

COPY shared/ ./shared/
COPY services/media/ ./services/media/

RUN go work init && \
    go work use ./shared ./services/media && \
    go work sync

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /bin/media \
    ./services/media/cmd/

FROM alpine:3.19
RUN apk --no-cache add ca-certificates curl
WORKDIR /app
COPY --from=builder /bin/media .
EXPOSE 9004
CMD ["./media"]
