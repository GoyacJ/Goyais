package unified

import "context"

type AuditLoggerFunc func(ctx context.Context, event AuditEvent) error

func (f AuditLoggerFunc) Record(ctx context.Context, event AuditEvent) error {
	if f == nil {
		return nil
	}
	return f(ctx, event)
}
