package printmode

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type fakeExecutor struct {
	results []TurnResult
	err     error
	reqs    []TurnRequest
}

func (f *fakeExecutor) ExecuteTurn(_ context.Context, req TurnRequest) (TurnResult, error) {
	f.reqs = append(f.reqs, req)
	if f.err != nil {
		return TurnResult{}, f.err
	}
	if len(f.results) == 0 {
		return TurnResult{
			SessionID: "sess_default",
			RunID:     "run_default",
			Output:    req.Prompt,
		}, nil
	}
	result := f.results[0]
	f.results = f.results[1:]
	return result, nil
}

func TestRunnerRejectsPromptWhenStreamJSONInput(t *testing.T) {
	executor := &fakeExecutor{}
	runner := Runner{
		Input:    bytes.NewBufferString(""),
		Output:   &bytes.Buffer{},
		Executor: executor,
	}
	err := runner.Run(context.Background(), RunRequest{
		Prompt:       "hello",
		InputFormat:  "stream-json",
		OutputFormat: "stream-json",
		Verbose:      true,
	})
	if err == nil {
		t.Fatal("expected stream-json prompt argument to fail")
	}
	if !strings.Contains(err.Error(), "cannot be used with a prompt argument") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunnerJSONOutputWithSchemaSuccess(t *testing.T) {
	output := &bytes.Buffer{}
	executor := &fakeExecutor{
		results: []TurnResult{
			{
				SessionID: "sess_1",
				RunID:     "run_1",
				Output:    `{"name":"Ada"}`,
			},
		},
	}
	runner := Runner{
		Output:   output,
		Executor: executor,
	}

	err := runner.Run(context.Background(), RunRequest{
		Prompt:       "return json",
		OutputFormat: "json",
		JSONSchema: `{
			"type":"object",
			"properties":{"name":{"type":"string"}},
			"required":["name"]
		}`,
	})
	if err != nil {
		t.Fatalf("expected json output to succeed: %v", err)
	}

	var payload map[string]any
	if unmarshalErr := json.Unmarshal(output.Bytes(), &payload); unmarshalErr != nil {
		t.Fatalf("expected JSON output, got error: %v\noutput=%s", unmarshalErr, output.String())
	}
	if payload["subtype"] != "success" {
		t.Fatalf("expected success result subtype, got %v", payload["subtype"])
	}
	if payload["is_error"] != false {
		t.Fatalf("expected non-error result, got %v", payload["is_error"])
	}
}

func TestRunnerJSONOutputSchemaFailure(t *testing.T) {
	output := &bytes.Buffer{}
	executor := &fakeExecutor{
		results: []TurnResult{
			{
				SessionID: "sess_1",
				RunID:     "run_1",
				Output:    `{"name":123}`,
			},
		},
	}
	runner := Runner{
		Output:   output,
		Executor: executor,
	}

	err := runner.Run(context.Background(), RunRequest{
		Prompt:       "return json",
		OutputFormat: "json",
		JSONSchema: `{
			"type":"object",
			"properties":{"name":{"type":"string"}},
			"required":["name"]
		}`,
	})
	if err != nil {
		t.Fatalf("expected schema failure to be returned as result payload, got error: %v", err)
	}

	var payload map[string]any
	if unmarshalErr := json.Unmarshal(output.Bytes(), &payload); unmarshalErr != nil {
		t.Fatalf("expected JSON output, got error: %v\noutput=%s", unmarshalErr, output.String())
	}
	if payload["subtype"] != "error_during_execution" {
		t.Fatalf("expected error_during_execution subtype, got %v", payload["subtype"])
	}
	resultText, _ := payload["result"].(string)
	if !strings.Contains(resultText, "Structured output failed JSON schema validation") {
		t.Fatalf("expected schema validation failure text, got %q", resultText)
	}
}

func TestRunnerStreamJSONRoundTripWithControlRequest(t *testing.T) {
	input := bytes.NewBufferString(strings.Join([]string{
		`{"type":"control_request","request_id":"req-init","request":{"subtype":"initialize"}}`,
		`{"type":"control_request","request_id":"req-model","request":{"subtype":"set_model","model":"gpt-5-mini"}}`,
		`{"type":"user","uuid":"u1","message":{"role":"user","content":"hello stream"}}`,
		"",
	}, "\n"))
	output := &bytes.Buffer{}
	executor := &fakeExecutor{
		results: []TurnResult{
			{
				SessionID:    "sess_2",
				RunID:        "run_2",
				Output:       "hello stream",
				OutputChunks: []string{"hello ", "stream"},
			},
		},
	}
	runner := Runner{
		Input:    input,
		Output:   output,
		Executor: executor,
	}

	err := runner.Run(context.Background(), RunRequest{
		InputFormat:          "stream-json",
		OutputFormat:         "stream-json",
		Verbose:              true,
		ReplayUserMessages:   true,
		IncludePartial:       true,
		PermissionPromptTool: "stdio",
		CWD:                  "/tmp/work",
	})
	if err != nil {
		t.Fatalf("expected stream-json run to succeed: %v", err)
	}

	lines := decodeJSONLines(t, output.String())
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 JSON lines, got %d\n%s", len(lines), output.String())
	}

	var hasInit bool
	var hasControlSuccess bool
	var hasUserReplay bool
	var hasResult bool
	for _, line := range lines {
		switch line["type"] {
		case "system":
			if line["subtype"] == "init" {
				hasInit = true
			}
		case "control_response":
			response, _ := line["response"].(map[string]any)
			if response["subtype"] == "success" {
				hasControlSuccess = true
			}
		case "user":
			hasUserReplay = true
		case "result":
			hasResult = true
		}
	}

	if !hasInit {
		t.Fatalf("expected init system line, output=%s", output.String())
	}
	if !hasControlSuccess {
		t.Fatalf("expected control_response success, output=%s", output.String())
	}
	if !hasUserReplay {
		t.Fatalf("expected replayed user line, output=%s", output.String())
	}
	if !hasResult {
		t.Fatalf("expected result line, output=%s", output.String())
	}
	if len(executor.reqs) != 1 {
		t.Fatalf("expected exactly one executed turn, got %d", len(executor.reqs))
	}
	if executor.reqs[0].Model != "gpt-5-mini" {
		t.Fatalf("expected set_model control request to override model, got %q", executor.reqs[0].Model)
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
			t.Fatalf("failed to decode JSON line %q: %v", line, err)
		}
		out = append(out, entry)
	}
	return out
}
