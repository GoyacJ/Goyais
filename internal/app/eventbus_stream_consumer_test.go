package app

import (
	"context"
	"encoding/json"
	"testing"

	"goyais/internal/command"

	"github.com/segmentio/kafka-go"
)

type fakeSubmitter struct {
	calls []submitCall
}

type submitCall struct {
	RequestContext command.RequestContext
	CommandType    string
	Payload        json.RawMessage
	IdempotencyKey string
	Visibility     string
}

func (f *fakeSubmitter) Submit(
	_ context.Context,
	reqCtx command.RequestContext,
	commandType string,
	payload json.RawMessage,
	idempotencyKey string,
	requestedVisibility string,
) (command.Command, error) {
	f.calls = append(f.calls, submitCall{
		RequestContext: reqCtx,
		CommandType:    commandType,
		Payload:        payload,
		IdempotencyKey: idempotencyKey,
		Visibility:     requestedVisibility,
	})
	return command.Command{ID: "cmd_test", CommandType: commandType}, nil
}

func TestHandleStreamOnPublishMessageSubmitsWorkflowRun(t *testing.T) {
	event := map[string]any{
		"eventType":   "stream.on_publish",
		"tenantId":    "t1",
		"workspaceId": "w1",
		"userId":      "u1",
		"traceId":     "trace-1",
		"streamId":    "stream_1",
		"recordingId": "rec_1",
		"templateId":  "tpl_1",
		"visibility":  "WORKSPACE",
	}
	raw, _ := json.Marshal(event)
	msg := kafka.Message{
		Key:   []byte("stream_1:rec_1"),
		Value: raw,
	}
	submitter := &fakeSubmitter{}
	if err := handleStreamOnPublishMessage(context.Background(), submitter, msg); err != nil {
		t.Fatalf("handle stream event: %v", err)
	}
	if len(submitter.calls) != 1 {
		t.Fatalf("expected one submit call, got=%d", len(submitter.calls))
	}
	call := submitter.calls[0]
	if call.CommandType != "workflow.run" {
		t.Fatalf("unexpected command type: %s", call.CommandType)
	}
	if call.IdempotencyKey != "stream-onpublish-rec_1" {
		t.Fatalf("unexpected idempotency key: %s", call.IdempotencyKey)
	}
	if call.Visibility != "WORKSPACE" {
		t.Fatalf("unexpected visibility: %s", call.Visibility)
	}
}

func TestHandleStreamOnPublishMessageIgnoresOtherEvents(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"eventType": "stream.record.stop",
	})
	submitter := &fakeSubmitter{}
	if err := handleStreamOnPublishMessage(context.Background(), submitter, kafka.Message{Value: raw}); err != nil {
		t.Fatalf("handle stream event: %v", err)
	}
	if len(submitter.calls) != 0 {
		t.Fatalf("expected no submit call for non on_publish event")
	}
}
