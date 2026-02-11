package stream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMediaMTXEnsurePathAddsPath(t *testing.T) {
	var (
		mu      sync.Mutex
		called  bool
		gotPath string
		gotBody map[string]any
		gotAuth string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/v3/config/paths/add/") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		mu.Lock()
		called = true
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client, err := NewMediaMTXControlPlane(MediaMTXControlPlaneOptions{
		BaseURL:        server.URL,
		APIUser:        "api-user",
		APIPassword:    "api-pass",
		RequestTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.EnsurePath(context.Background(), "/camera/main", "push", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("ensure path: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !called {
		t.Fatalf("expected add endpoint called")
	}
	if gotPath != "/v3/config/paths/add/camera/main" {
		t.Fatalf("unexpected add path: %s", gotPath)
	}
	if gotAuth == "" {
		t.Fatalf("expected auth header")
	}
	if source, _ := gotBody["source"].(string); source != "publisher" {
		t.Fatalf("unexpected source: %v", source)
	}
	if recordPath, _ := gotBody["recordPath"].(string); recordPath == "" {
		t.Fatalf("expected recordPath")
	}
}

func TestMediaMTXEnsurePathAlreadyExistsFallsBackToPatch(t *testing.T) {
	var (
		mu          sync.Mutex
		addCalled   bool
		patchCalled bool
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v3/config/paths/add/"):
			addCalled = true
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"status":"error","error":"path already exists"}`))
		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/v3/config/paths/patch/"):
			patchCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewMediaMTXControlPlane(MediaMTXControlPlaneOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.EnsurePath(context.Background(), "stream-a", "push", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("ensure path: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !addCalled || !patchCalled {
		t.Fatalf("expected add then patch, got add=%v patch=%v", addCalled, patchCalled)
	}
}

func TestMediaMTXDeletePathIgnoresNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":"error","error":"path not found"}`))
	}))
	defer server.Close()

	client, err := NewMediaMTXControlPlane(MediaMTXControlPlaneOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.DeletePath(context.Background(), "/stream/not-found"); err != nil {
		t.Fatalf("delete path should ignore not found, got=%v", err)
	}
}

func TestMediaMTXKickPathFiltersByPath(t *testing.T) {
	var (
		mu        sync.Mutex
		kickedIDs []string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/rtspsessions/list", "/v3/rtmpconns/list", "/v3/srtconns/list", "/v3/webrtcsessions/list":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"items":[{"id":"id-1","path":"cam/live"},{"id":"id-2","path":"other/path"}]}`))
			return
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/kick/") {
			mu.Lock()
			kickedIDs = append(kickedIDs, strings.TrimPrefix(r.URL.Path[strings.LastIndex(r.URL.Path, "/"):], "/"))
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client, err := NewMediaMTXControlPlane(MediaMTXControlPlaneOptions{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.KickPath(context.Background(), "/cam/live"); err != nil {
		t.Fatalf("kick path: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(kickedIDs) != 4 {
		t.Fatalf("expected one kick per endpoint, got=%d ids=%v", len(kickedIDs), kickedIDs)
	}
}
