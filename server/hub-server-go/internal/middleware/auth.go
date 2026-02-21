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

const (
	AuthModeLocalOpen  = "local_open"
	AuthModeRemoteAuth = "remote_auth"
)

type LocalUserResolver func(ctx context.Context) (*model.AuthUser, error)

// AuthMiddleware injects the current principal into request context.
// - remote_auth: requires Authorization: Bearer <token>
// - local_open: accepts unauthenticated requests and injects a local principal
func AuthMiddleware(
	authMode string,
	validateToken func(token string) (*model.AuthUser, error),
	resolveLocalUser LocalUserResolver,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authMode == AuthModeLocalOpen {
				if resolveLocalUser == nil {
					http.Error(w, `{"error":{"code":"E_INTERNAL","message":"local principal resolver is not configured"}}`, http.StatusInternalServerError)
					return
				}
				user, err := resolveLocalUser(r.Context())
				if err != nil {
					http.Error(w, `{"error":{"code":"E_INTERNAL","message":"failed to resolve local principal"}}`, http.StatusInternalServerError)
					return
				}
				ctx := context.WithValue(r.Context(), ctxUser, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

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
