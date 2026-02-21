package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/goyais/hub/internal/model"
)

type contextKey string

const (
	ctxUser contextKey = "user"
)

// AuthMiddleware validates Bearer token and injects user into context.
// tokenValidator is a func injected at wire-up time (avoids circular deps).
func AuthMiddleware(validateToken func(token string) (*model.AuthUser, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":{"code":"E_UNAUTHORIZED","message":"missing token"}}`, http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")
			user, err := validateToken(token)
			if err != nil {
				http.Error(w, `{"error":{"code":"E_UNAUTHORIZED","message":"invalid or expired token"}}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromCtx extracts the authenticated user from context.
func UserFromCtx(ctx context.Context) *model.AuthUser {
	u, _ := ctx.Value(ctxUser).(*model.AuthUser)
	return u
}
