package handler

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

var startTime = time.Now()

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":     "ok",
		"uptime":     time.Since(startTime).Round(time.Second).String(),
		"go_version": runtime.Version(),
	})
}
