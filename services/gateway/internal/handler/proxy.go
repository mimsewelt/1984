package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

func Proxy(target, stripPrefix string, log *zap.Logger) http.HandlerFunc {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic("invalid proxy target: " + target)
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error("proxy error", zap.String("target", target), zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream unavailable"}`))
	}
	orig := proxy.Director
	proxy.Director = func(r *http.Request) {
		orig(r)
		r.URL.Path = strings.TrimPrefix(r.URL.Path, stripPrefix)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		r.Host = targetURL.Host
	}
	return proxy.ServeHTTP
}
