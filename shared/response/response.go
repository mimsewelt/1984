package response

import (
 "encoding/json"
 "net/http"
)

type Envelope struct {
 Data  any    json:"data,omitempty"
 Error string json:"error,omitempty"
}

func JSON(w http.ResponseWriter, status int, data any) {
 w.Header().Set("Content-Type", "application/json")
 w.WriteHeader(status)
 _ = json.NewEncoder(w).Encode(Envelope{Data: data})
}

func Error(w http.ResponseWriter, status int, msg string) {
 w.Header().Set("Content-Type", "application/json")
 w.WriteHeader(status)
 _ = json.NewEncoder(w).Encode(Envelope{Error: msg})
}

func OK(w http.ResponseWriter, data any) {
 JSON(w, http.StatusOK, data)
}

func Created(w http.ResponseWriter, data any) {
 JSON(w, http.StatusCreated, data)
}

func BadRequest(w http.ResponseWriter, msg string) {
 Error(w, http.StatusBadRequest, msg)
}

func Unauthorized(w http.ResponseWriter) {
 Error(w, http.StatusUnauthorized, "unauthorized")
}

func Forbidden(w http.ResponseWriter) {
 Error(w, http.StatusForbidden, "forbidden")
}

func NotFound(w http.ResponseWriter) {
 Error(w, http.StatusNotFound, "not found")
}

func InternalError(w http.ResponseWriter) {
 Error(w, http.StatusInternalServerError, "internal server error")
}