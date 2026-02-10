package httpapi

import (
	"net/http"

	"goyais/internal/command"
)

func requireRequestContext(w http.ResponseWriter, r *http.Request) (command.RequestContext, bool) {
	return extractRequestContext(w, r)
}
