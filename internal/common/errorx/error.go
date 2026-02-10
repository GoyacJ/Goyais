package errorx

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	Code       string      `json:"code"`
	MessageKey string      `json:"messageKey"`
	Details    interface{} `json:"details,omitempty"`
}

type Envelope struct {
	Error APIError `json:"error"`
}

func Write(w http.ResponseWriter, status int, code, messageKey string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Error: APIError{Code: code, MessageKey: messageKey, Details: details}})
}
