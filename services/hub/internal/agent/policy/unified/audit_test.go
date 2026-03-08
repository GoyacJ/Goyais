package unified

import (
	"context"
	"testing"
)

func TestAuditLoggerFuncRecordsEvent(t *testing.T) {
	var recorded AuditEvent
	logger := AuditLoggerFunc(func(_ context.Context, event AuditEvent) error {
		recorded = event
		return nil
	})

	if err := logger.Record(context.Background(), AuditEvent{
		WorkspaceID: "ws_test",
		Action:      "session.read",
		Result:      "success",
	}); err != nil {
		t.Fatalf("record audit event failed: %v", err)
	}

	if recorded.WorkspaceID != "ws_test" || recorded.Action != "session.read" || recorded.Result != "success" {
		t.Fatalf("unexpected recorded audit event: %#v", recorded)
	}
}
