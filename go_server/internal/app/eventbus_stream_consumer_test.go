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

func TestHandleStreamTriggerMessageSubmitsWorkflowRun(t *testing.T) {
	cases := []struct {
		name               string
		event              map[string]any
		wantIdempotencyKey string
		wantVisibility     string
		wantTrigger        string
		wantRecordingID    string
		wantAssetID        string
	}{
		{
			name: "on publish",
			event: map[string]any{
				"eventType":   "stream.on_publish",
				"eventId":     "evt_1",
				"tenantId":    "t1",
				"workspaceId": "w1",
				"userId":      "u1",
				"traceId":     "trace-1",
				"streamId":    "stream_1",
				"recordingId": "rec_1",
				"templateId":  "tpl_1",
				"visibility":  "WORKSPACE",
			},
			wantIdempotencyKey: "stream-onpublish-rec_1",
			wantVisibility:     "WORKSPACE",
			wantTrigger:        "stream.onPublish",
			wantRecordingID:    "rec_1",
		},
		{
			name: "on read",
			event: map[string]any{
				"eventType":   "stream.on_read",
				"eventId":     "evt_2",
				"tenantId":    "t1",
				"workspaceId": "w1",
				"userId":      "u1",
				"traceId":     "trace-2",
				"streamId":    "stream_2",
				"templateId":  "tpl_2",
				"visibility":  "PRIVATE",
			},
			wantIdempotencyKey: "stream-onread-evt_2",
			wantVisibility:     "PRIVATE",
			wantTrigger:        "stream.onRead",
		},
		{
			name: "on connect",
			event: map[string]any{
				"eventType":   "stream.on_connect",
				"eventId":     "evt_3",
				"tenantId":    "t1",
				"workspaceId": "w1",
				"userId":      "u1",
				"traceId":     "trace-3",
				"streamId":    "stream_3",
				"templateId":  "tpl_3",
				"visibility":  "PRIVATE",
			},
			wantIdempotencyKey: "stream-onconnect-evt_3",
			wantVisibility:     "PRIVATE",
			wantTrigger:        "stream.onConnect",
		},
		{
			name: "on record finish",
			event: map[string]any{
				"eventType":   "stream.on_record_finish",
				"eventId":     "evt_4",
				"tenantId":    "t1",
				"workspaceId": "w1",
				"userId":      "u1",
				"traceId":     "trace-4",
				"streamId":    "stream_4",
				"recordingId": "rec_4",
				"assetId":     "ast_4",
				"templateId":  "tpl_4",
				"visibility":  "WORKSPACE",
			},
			wantIdempotencyKey: "stream-onrecordfinish-rec_4",
			wantVisibility:     "WORKSPACE",
			wantTrigger:        "stream.onRecordFinish",
			wantRecordingID:    "rec_4",
			wantAssetID:        "ast_4",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := json.Marshal(tc.event)
			msg := kafka.Message{
				Key:   []byte("stream-trigger"),
				Value: raw,
			}
			submitter := &fakeSubmitter{}
			if err := handleStreamTriggerMessage(context.Background(), submitter, msg); err != nil {
				t.Fatalf("handle stream event: %v", err)
			}
			if len(submitter.calls) != 1 {
				t.Fatalf("expected one submit call, got=%d", len(submitter.calls))
			}
			call := submitter.calls[0]
			if call.CommandType != "workflow.run" {
				t.Fatalf("unexpected command type: %s", call.CommandType)
			}
			if call.IdempotencyKey != tc.wantIdempotencyKey {
				t.Fatalf("unexpected idempotency key: %s", call.IdempotencyKey)
			}
			if call.Visibility != tc.wantVisibility {
				t.Fatalf("unexpected visibility: %s", call.Visibility)
			}
			var payload map[string]any
			if err := json.Unmarshal(call.Payload, &payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			inputs, _ := payload["inputs"].(map[string]any)
			gotTrigger, _ := inputs["trigger"].(string)
			if gotTrigger != tc.wantTrigger {
				t.Fatalf("unexpected trigger: %s", gotTrigger)
			}
			if tc.wantRecordingID != "" {
				gotRecordingID, _ := inputs["recordingId"].(string)
				if gotRecordingID != tc.wantRecordingID {
					t.Fatalf("unexpected recording id: %s", gotRecordingID)
				}
			}
			if tc.wantAssetID != "" {
				gotAssetID, _ := inputs["assetId"].(string)
				if gotAssetID != tc.wantAssetID {
					t.Fatalf("unexpected asset id: %s", gotAssetID)
				}
			}
		})
	}
}

func TestHandleStreamTriggerMessageIgnoresInvalidOrUnsupportedEvents(t *testing.T) {
	cases := []map[string]any{
		{
			"eventType": "stream.record.stop",
		},
		{
			"eventType":   "stream.on_publish",
			"streamId":    "stream_1",
			"templateId":  "tpl_1",
			"recordingId": "",
		},
		{
			"eventType":   "stream.on_connect",
			"eventId":     "evt_1",
			"streamId":    "stream_1",
			"templateId":  "tpl_1",
			"workspaceId": "w1",
			"userId":      "u1",
		},
	}
	for _, event := range cases {
		raw, _ := json.Marshal(event)
		submitter := &fakeSubmitter{}
		if err := handleStreamTriggerMessage(context.Background(), submitter, kafka.Message{Value: raw}); err != nil {
			t.Fatalf("handle stream event: %v", err)
		}
		if len(submitter.calls) != 0 {
			t.Fatalf("expected no submit call for event=%v", event)
		}
	}
}
