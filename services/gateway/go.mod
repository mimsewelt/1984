module github.com/yourorg/instagram-clone/services/gateway

go 1.23

require (
 github.com/go-chi/chi/v5 v5.1.0
 github.com/go-chi/httprate v0.14.1
 github.com/golang-jwt/jwt/v5 v5.2.1
 github.com/yourorg/instagram-clone/shared v0.0.0
 go.uber.org/zap v1.27.0
)

replace github.com/yourorg/instagram-clone/shared => ../../shared