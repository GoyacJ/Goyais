package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goyais/hub/internal/model"
)

func TestAuthMiddlewareRemoteAuthRequiresBearer(t *testing.T) {
	mw := AuthMiddleware(AuthModeRemoteAuth, func(token string) (*model.AuthUser, error) {
		return model.NewAuthUser("u1", "u1@example.com", "U1"), nil
	}, nil)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d", rr.Code)
	}
}

func TestAuthMiddlewareRemoteAuthInjectsValidatedUser(t *testing.T) {
	mw := AuthMiddleware(AuthModeRemoteAuth, func(token string) (*model.AuthUser, error) {
		if token != "valid-token" {
			return nil, errors.New("invalid")
		}
		return model.NewAuthUser("u1", "u1@example.com", "U1"), nil
	}, nil)

	var gotUserID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromCtx(r.Context())
		if user != nil {
			gotUserID = user.UserID
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if gotUserID != "u1" {
		t.Fatalf("expected injected user u1, got %q", gotUserID)
	}
}

func TestAuthMiddlewareLocalOpenInjectsLocalUserWithoutBearer(t *testing.T) {
	mw := AuthMiddleware(AuthModeLocalOpen, func(token string) (*model.AuthUser, error) {
		return nil, errors.New("should not call token validator in local_open")
	}, func(_ctx context.Context) (*model.AuthUser, error) {
		return model.NewAuthUser("local-user", "local@goyais.local", "Local"), nil
	})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromCtx(r.Context())
		if user == nil || user.UserID != "local-user" {
			t.Fatalf("expected local-user in context, got %+v", user)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}
