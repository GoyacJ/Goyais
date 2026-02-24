package httpapi

import "net/http"

const (
	corsAllowOriginHeader  = "*"
	corsAllowMethodsHeader = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	corsAllowHeadersHeader = "Authorization, Content-Type, X-Trace-Id"
	corsExposeHeaders      = "X-Trace-Id"
	corsMaxAgeHeader       = "600"
)

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsAllowOriginHeader)
		w.Header().Set("Access-Control-Allow-Methods", corsAllowMethodsHeader)
		w.Header().Set("Access-Control-Allow-Headers", corsAllowHeadersHeader)
		w.Header().Set("Access-Control-Expose-Headers", corsExposeHeaders)
		w.Header().Set("Access-Control-Max-Age", corsMaxAgeHeader)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
