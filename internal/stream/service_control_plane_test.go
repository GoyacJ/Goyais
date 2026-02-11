package stream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMediaMTXEnsurePathLegacyAuthConflictIgnored(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v3/config/paths/add/stream-legacy-auth" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":"error","error":"authInternalUsers and legacy credentials (publishUser, publishPass, publishIPs, readUser, readPass, readIPs) cannot be used together"}`))
	}))
	defer server.Close()

	client, err := NewMediaMTXControlPlane(MediaMTXControlPlaneOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.EnsurePath(context.Background(), "stream-legacy-auth", "push", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("ensure path should ignore legacy auth conflict, got=%v", err)
	}
}

func TestTranslateControlPlaneError(t *testing.T) {
	service := &Service{}

	cases := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "bad request maps to invalid request",
			err:  &mediaMTXStatusError{StatusCode: http.StatusBadRequest, Message: "invalid value"},
			want: ErrInvalidRequest,
		},
		{
			name: "legacy auth conflict maps to not implemented",
			err:  &mediaMTXStatusError{StatusCode: http.StatusBadRequest, Message: "authInternalUsers and legacy credentials cannot be used together"},
			want: ErrNotImplemented,
		},
		{
			name: "not found maps to stream not found",
			err:  &mediaMTXStatusError{StatusCode: http.StatusNotFound, Message: "path not found"},
			want: ErrStreamNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := service.translateControlPlaneError(tc.err)
			if got == nil {
				t.Fatalf("expected mapped error")
			}
			if got != tc.want {
				t.Fatalf("unexpected mapped error: got=%v want=%v", got, tc.want)
			}
		})
	}
}
