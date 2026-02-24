package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const internalForwardedLoginHeader = "X-Goyais-Forwarded-Login"

var proxyHTTPClient = &http.Client{Timeout: 5 * time.Second}

func proxyLoginToTarget(ctx context.Context, hubURL string, input LoginRequest, traceID string, role Role) (LoginResponse, *apiError) {
	targetURL := strings.TrimRight(strings.TrimSpace(hubURL), "/") + "/v1/auth/login"
	body, err := json.Marshal(input)
	if err != nil {
		return LoginResponse{}, &apiError{
			status:  http.StatusInternalServerError,
			code:    "INTERNAL_ENCODING_ERROR",
			message: "Failed to encode proxy login payload",
			details: map[string]any{"reason": err.Error()},
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return LoginResponse{}, &apiError{
			status:  http.StatusBadGateway,
			code:    "UPSTREAM_REQUEST_BUILD_FAILED",
			message: "Failed to create upstream login request",
			details: map[string]any{"reason": err.Error()},
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(internalForwardedLoginHeader, "1")
	if traceID != "" {
		req.Header.Set(TraceHeader, traceID)
	}
	if role != "" {
		req.Header.Set("X-Role", string(role))
	}

	res, err := proxyHTTPClient.Do(req)
	if err != nil {
		return LoginResponse{}, &apiError{
			status:  http.StatusBadGateway,
			code:    "UPSTREAM_REQUEST_FAILED",
			message: "Failed to reach upstream hub",
			details: map[string]any{"reason": err.Error(), "target": targetURL},
		}
	}
	defer res.Body.Close()

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		return LoginResponse{}, &apiError{
			status:  http.StatusBadGateway,
			code:    "UPSTREAM_READ_FAILED",
			message: "Failed to read upstream response",
			details: map[string]any{"reason": err.Error()},
		}
	}

	if res.StatusCode != http.StatusOK {
		upstreamErr := StandardError{}
		if err := json.Unmarshal(payload, &upstreamErr); err == nil && upstreamErr.Code != "" {
			return LoginResponse{}, &apiError{
				status:  res.StatusCode,
				code:    upstreamErr.Code,
				message: upstreamErr.Message,
				details: upstreamErr.Details,
			}
		}
		return LoginResponse{}, &apiError{
			status:  http.StatusBadGateway,
			code:    "UPSTREAM_UNEXPECTED_STATUS",
			message: "Upstream hub returned unexpected status",
			details: map[string]any{"status": res.StatusCode},
		}
	}

	loginResponse := LoginResponse{}
	if err := json.Unmarshal(payload, &loginResponse); err != nil {
		return LoginResponse{}, &apiError{
			status:  http.StatusBadGateway,
			code:    "UPSTREAM_INVALID_PAYLOAD",
			message: "Upstream login payload is invalid",
			details: map[string]any{"reason": err.Error()},
		}
	}

	if strings.TrimSpace(loginResponse.AccessToken) == "" {
		return LoginResponse{}, &apiError{
			status:  http.StatusBadGateway,
			code:    "UPSTREAM_MISSING_TOKEN",
			message: "Upstream login response missing access token",
			details: map[string]any{},
		}
	}

	if strings.TrimSpace(loginResponse.TokenType) == "" {
		loginResponse.TokenType = "bearer"
	}

	return loginResponse, nil
}
