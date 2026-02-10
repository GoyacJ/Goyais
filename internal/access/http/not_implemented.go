package httpapi

import (
	"net/http"

	"goyais/internal/common/errorx"
)

func NewNotImplementedHandler(messageKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errorx.Write(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", messageKey, nil)
	})
}
