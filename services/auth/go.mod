module github.com/yourorg/instagram-clone/services/auth

go 1.23

require (
 github.com/go-chi/chi/v5 v5.1.0
 github.com/golang-jwt/jwt/v5 v5.2.1
 github.com/google/uuid v1.6.0
 github.com/jackc/pgx/v5 v5.6.0
 github.com/yourorg/instagram-clone/shared v0.0.0
 go.uber.org/zap v1.27.0
 golang.org/x/crypto v0.24.0
)

replace github.com/yourorg/instagram-clone/shared => ../../shared