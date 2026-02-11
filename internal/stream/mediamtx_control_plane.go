package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultMediaMTXRecordPath = "./recordings/%path/%Y-%m-%d_%H-%M-%S-%f"

type MediaMTXControlPlaneOptions struct {
	BaseURL        string
	APIUser        string
	APIPassword    string
	RequestTimeout time.Duration
}

type MediaMTXControlPlane struct {
	baseURL     string
	apiUser     string
	apiPassword string
	httpClient  *http.Client
}

type mediaMTXErrorResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

func NewMediaMTXControlPlane(options MediaMTXControlPlaneOptions) (*MediaMTXControlPlane, error) {
	baseURL := strings.TrimSpace(options.BaseURL)
	if baseURL == "" {
		return nil, errors.New("mediamtx control plane base url is required")
	}
	if _, err := url.Parse(baseURL); err != nil {
		return nil, fmt.Errorf("parse mediamtx base url: %w", err)
	}
	timeout := options.RequestTimeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &MediaMTXControlPlane{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		apiUser:     strings.TrimSpace(options.APIUser),
		apiPassword: options.APIPassword,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *MediaMTXControlPlane) EnsurePath(ctx context.Context, streamPath string, source string, state json.RawMessage) error {
	normalizedPath, err := normalizeMediaMTXPath(streamPath)
	if err != nil {
		return err
	}

	sourceValue := "publisher"
	if strings.EqualFold(strings.TrimSpace(source), "pull") {
		if pullSource := extractPullSource(state); pullSource != "" {
			sourceValue = pullSource
		}
	}

	payload := map[string]any{
		"source":     sourceValue,
		"record":     false,
		"recordPath": defaultMediaMTXRecordPath,
	}
	if err := c.doJSON(ctx, http.MethodPost, "/v3/config/paths/add/"+url.PathEscape(normalizedPath), payload, nil); err != nil {
		var statusErr *mediaMTXStatusError
		if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusBadRequest {
			if strings.Contains(strings.ToLower(statusErr.Message), "already exists") {
				return c.doJSON(ctx, http.MethodPatch, "/v3/config/paths/patch/"+url.PathEscape(normalizedPath), payload, nil)
			}
		}
		return err
	}
	return nil
}

func (c *MediaMTXControlPlane) PatchPathAuth(ctx context.Context, streamPath string, authRule map[string]any) error {
	normalizedPath, err := normalizeMediaMTXPath(streamPath)
	if err != nil {
		return err
	}
	if len(authRule) == 0 {
		return nil
	}
	return c.doJSON(ctx, http.MethodPatch, "/v3/config/paths/patch/"+url.PathEscape(normalizedPath), authRule, nil)
}

func (c *MediaMTXControlPlane) DeletePath(ctx context.Context, streamPath string) error {
	normalizedPath, err := normalizeMediaMTXPath(streamPath)
	if err != nil {
		return err
	}
	err = c.doJSON(ctx, http.MethodDelete, "/v3/config/paths/delete/"+url.PathEscape(normalizedPath), nil, nil)
	if err == nil {
		return nil
	}
	var statusErr *mediaMTXStatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound {
		// Path was already deleted on the control plane.
		return nil
	}
	return err
}

func (c *MediaMTXControlPlane) KickPath(ctx context.Context, streamPath string) error {
	normalizedPath, err := normalizeMediaMTXPath(streamPath)
	if err != nil {
		return err
	}

	type kickTarget struct {
		listEndpoint string
		kickEndpoint string
	}
	targets := []kickTarget{
		{listEndpoint: "/v3/rtspsessions/list", kickEndpoint: "/v3/rtspsessions/kick/"},
		{listEndpoint: "/v3/rtmpconns/list", kickEndpoint: "/v3/rtmpconns/kick/"},
		{listEndpoint: "/v3/srtconns/list", kickEndpoint: "/v3/srtconns/kick/"},
		{listEndpoint: "/v3/webrtcsessions/list", kickEndpoint: "/v3/webrtcsessions/kick/"},
	}

	for _, target := range targets {
		ids, err := c.listConnectionIDsByPath(ctx, target.listEndpoint, normalizedPath)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if id == "" {
				continue
			}
			if err := c.doJSON(ctx, http.MethodPost, target.kickEndpoint+url.PathEscape(id), map[string]any{}, nil); err != nil {
				var statusErr *mediaMTXStatusError
				if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound {
					continue
				}
				return err
			}
		}
	}
	return nil
}

func (c *MediaMTXControlPlane) listConnectionIDsByPath(ctx context.Context, endpoint string, streamPath string) ([]string, error) {
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &payload); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		pathValue := firstString(item, "path", "pathName", "stream", "streamPath")
		if normalizeMediaMTXPathUnsafe(pathValue) != streamPath {
			continue
		}
		id := firstString(item, "id")
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (c *MediaMTXControlPlane) doJSON(ctx context.Context, method string, endpoint string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal mediamtx request: %w", err)
		}
		bodyReader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("build mediamtx request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.apiUser != "" {
		req.SetBasicAuth(c.apiUser, c.apiPassword)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call mediamtx: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeMediaMTXStatusError(resp.StatusCode, raw)
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode mediamtx response: %w", err)
	}
	return nil
}

type mediaMTXStatusError struct {
	StatusCode int
	Message    string
}

func (e *mediaMTXStatusError) Error() string {
	if e == nil {
		return "mediamtx status error"
	}
	if e.Message == "" {
		return fmt.Sprintf("mediamtx status=%d", e.StatusCode)
	}
	return fmt.Sprintf("mediamtx status=%d: %s", e.StatusCode, e.Message)
}

func decodeMediaMTXStatusError(statusCode int, raw []byte) error {
	msg := strings.TrimSpace(string(raw))
	var payload mediaMTXErrorResponse
	if json.Unmarshal(raw, &payload) == nil && strings.TrimSpace(payload.Error) != "" {
		msg = strings.TrimSpace(payload.Error)
	}
	return &mediaMTXStatusError{
		StatusCode: statusCode,
		Message:    msg,
	}
}

func normalizeMediaMTXPath(raw string) (string, error) {
	path := normalizeMediaMTXPathUnsafe(raw)
	if path == "" {
		return "", ErrInvalidRequest
	}
	return path, nil
}

func normalizeMediaMTXPathUnsafe(raw string) string {
	path := strings.TrimSpace(raw)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	return strings.TrimSpace(path)
}

func extractPullSource(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil {
		return ""
	}
	return firstString(state, "pullUrl", "sourceUrl", "upstreamUrl")
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		value, _ := m[key]
		text, _ := value.(string)
		text = strings.TrimSpace(text)
		if text != "" {
			return text
		}
	}
	return ""
}
