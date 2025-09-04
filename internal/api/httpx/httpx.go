package httpx

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	Error   string      `json:"error"`
	Code    string      `json:"code"`
	Details interface{} `json:"details,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, code, msg string, details interface{}) {
	WriteJSON(w, status, APIError{
		Error:   msg,
		Code:    code,
		Details: details,
	})
}
