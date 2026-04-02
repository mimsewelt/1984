package handler

import (
 "encoding/json"
 "net/http"
 "runtime"
 "time"
)

var startTime = time.Now()

type healthResponse struct {
 Status    string json:"status"
 Uptime    string json:"uptime"
 GoVersion string json:"go_version"
}

func Health(w http.ResponseWriter, r *http.Request) {
 w.Header().Set("Content-Type", "application/json")
 w.WriteHeader(http.StatusOK)
 _ = json.NewEncoder(w).Encode(healthResponse{
  Status:    "ok",
  Uptime:    time.Since(startTime).Round(time.Second).String(),
  GoVersion: runtime.Version(),
 })
}