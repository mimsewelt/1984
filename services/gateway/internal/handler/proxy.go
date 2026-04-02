package handler

import (
 "net/http"
 "net/http/httputil"
 "net/url"
 "strings"

 "go.uber.org/zap"
)

// Proxy creates a reverse proxy handler for a given target base URL.
// It strips the given stripPrefix from the request path before forwarding.
func Proxy(target, stripPrefix string, log *zap.Logger) http.HandlerFunc {
 targetURL, err := url.Parse(target)
 if err != nil {
  panic("invalid proxy target: " + target)
 }

 proxy := httputil.NewSingleHostReverseProxy(targetURL)

 // Custom error handler so upstream errors return structured JSON.
 proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
  log.Error("proxy error",
   zap.String("target", target),
   zap.String("path", r.URL.Path),
   zap.Error(err),
  )
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusBadGateway)
  _, _ = w.Write([]byte(`{"error":"upstream unavailable"}`))
 }

 // Rewrite adds X-Forwarded-* headers and forward user context.
 originalDirector := proxy.Director
 proxy.Director = func(r *http.Request) {
  originalDirector(r)
  r.URL.Path = strings.TrimPrefix(r.URL.Path, stripPrefix)
  if r.URL.Path == "" {
   r.URL.Path = "/"
  }
  // Forward authenticated user id to downstream services.
  if uid := r.Header.Get("X-User-Id"); uid != "" {
   r.Header.Set("X-User-Id", uid)
  }
  r.Header.Set("X-Forwarded-Host", r.Host)
  r.Host = targetURL.Host
 }

 return proxy.ServeHTTP
}