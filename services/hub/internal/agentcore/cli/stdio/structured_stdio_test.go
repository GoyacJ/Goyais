package stdio

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStructuredStdioRoutesUserAndControlRequests(t *testing.T) {
	input := bytes.NewBufferString(strings.Join([]string{
		`{"type":"control_request","request_id":"r1","request":{"subtype":"interrupt"}}`,
		`{"type":"control_request","request_id":"r2","request":{"subtype":"set_model","model":"gpt-5"}}`,
		`{"type":"user","uuid":"u1","message":{"role":"user","content":"hello"}}`,
		"",
	}, "\n"))
	output := &bytes.Buffer{}

	var interrupted atomic.Bool
	handler := NewStructuredStdio(input, output, HandlerOptions{
		OnInterrupt: func() {
			interrupted.Store(true)
		},
		OnControlRequest: func(req ControlRequest) (any, error) {
			if req.Request["subtype"] == "set_model" {
				return map[string]any{"ok": true}, nil
			}
			return nil, nil
		},
	})
	handler.Start()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	user, err := handler.NextUserMessage(ctx)
	if err != nil {
		t.Fatalf("expected user message, got error: %v", err)
	}
	if user.UUID != "u1" {
		t.Fatalf("expected uuid u1, got %q", user.UUID)
	}
	if !interrupted.Load() {
		t.Fatal("expected interrupt callback to be invoked")
	}

	responses := decodeJSONLines(t, output.String())
	if len(responses) < 2 {
		t.Fatalf("expected control responses, got %d entries: %s", len(responses), output.String())
	}

	var seenInterruptSuccess bool
	var seenSetModelSuccess bool
	for _, line := range responses {
		if line["type"] != "control_response" {
			continue
		}
		response, _ := line["response"].(map[string]any)
		if response["subtype"] != "success" {
			continue
		}
		switch response["request_id"] {
		case "r1":
			seenInterruptSuccess = true
		case "r2":
			seenSetModelSuccess = true
		}
	}
	if !seenInterruptSuccess || !seenSetModelSuccess {
		t.Fatalf("expected both control requests to succeed, got %s", output.String())
	}
}

func TestStructuredStdioUnsupportedControlRequestReturnsError(t *testing.T) {
	input := bytes.NewBufferString(strings.Join([]string{
		`{"type":"control_request","request_id":"r3","request":{"subtype":"unknown_subtype"}}`,
		"",
	}, "\n"))
	output := &bytes.Buffer{}
	handler := NewStructuredStdio(input, output, HandlerOptions{})
	handler.Start()

	time.Sleep(25 * time.Millisecond)
	lines := decodeJSONLines(t, output.String())
	if len(lines) == 0 {
		t.Fatalf("expected at least one output line, got none")
	}
	response, _ := lines[0]["response"].(map[string]any)
	if response["subtype"] != "error" {
		t.Fatalf("expected error response subtype, got %v", response["subtype"])
	}
	errText, _ := response["error"].(string)
	if !strings.Contains(errText, "Unsupported control request subtype") {
		t.Fatalf("unexpected error text: %q", errText)
	}
}

func decodeJSONLines(t *testing.T, raw string) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	out := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry := map[string]any{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("failed to parse JSON line %q: %v", line, err)
		}
		out = append(out, entry)
	}
	return out
}
