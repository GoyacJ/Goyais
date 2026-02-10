package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type openAPISpec struct {
	Paths map[string]map[string]any `yaml:"paths"`
}

type apiCase struct {
	Method string
	Path   string
}

var openAPIParamPattern = regexp.MustCompile(`\{([^}]+)\}`)

func TestOpenAPIPathsAreReachable(t *testing.T) {
	cases := mustLoadOpenAPICases(t)

	baseURL, shutdown := newTestServer(t)
	defer shutdown()

	client := &http.Client{Timeout: 10 * time.Second}
	for _, tc := range cases {
		t.Run(tc.Method+" "+tc.Path, func(t *testing.T) {
			targetPath := "/api/v1" + materializeOpenAPIPath(tc.Path)
			headers := headersWithContext("u1")

			var body io.Reader
			switch tc.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				headers.Set("Content-Type", "application/json")
				body = bytes.NewBufferString("{}")
			}

			resp := mustRequest(t, client, tc.Method, baseURL+targetPath, headers, body)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound {
				return
			}

			var payload map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				t.Fatalf("unexpected 404 for %s %s without json error payload: %v", tc.Method, targetPath, err)
			}

			errorPayload, _ := payload["error"].(map[string]any)
			errorCode, _ := errorPayload["code"].(string)
			if errorCode == "API_NOT_FOUND" {
				t.Fatalf("openapi path not mounted: %s %s", tc.Method, targetPath)
			}
		})
	}
}

func mustLoadOpenAPICases(t *testing.T) []apiCase {
	t.Helper()

	specPath := openAPISpecPath(t)
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read openapi spec: %v", err)
	}

	var spec openAPISpec
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse openapi spec: %v", err)
	}

	methodMap := map[string]string{
		"get":    http.MethodGet,
		"post":   http.MethodPost,
		"put":    http.MethodPut,
		"patch":  http.MethodPatch,
		"delete": http.MethodDelete,
	}

	cases := make([]apiCase, 0, len(spec.Paths))
	for path, operations := range spec.Paths {
		for rawMethod := range operations {
			method, ok := methodMap[strings.ToLower(strings.TrimSpace(rawMethod))]
			if !ok {
				continue
			}
			cases = append(cases, apiCase{
				Method: method,
				Path:   path,
			})
		}
	}

	sort.Slice(cases, func(i, j int) bool {
		if cases[i].Path == cases[j].Path {
			return cases[i].Method < cases[j].Method
		}
		return cases[i].Path < cases[j].Path
	})
	return cases
}

func openAPISpecPath(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file path")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	return filepath.Join(repoRoot, "docs", "api", "openapi.yaml")
}

func materializeOpenAPIPath(path string) string {
	return openAPIParamPattern.ReplaceAllStringFunc(path, func(token string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(token, "{"), "}")
		switch name {
		case "commandId":
			return "cmd_1"
		case "shareId":
			return "shr_1"
		case "assetId":
			return "ast_1"
		case "templateId":
			return "tpl_1"
		case "runId":
			return "run_1"
		case "capabilityId":
			return "cap_1"
		case "installId":
			return "ins_1"
		case "streamId":
			return "stream_1"
		default:
			return "id_1"
		}
	})
}
