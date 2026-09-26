package crud

import "context"

// NoopAuditSink intentionally discards mutation audit events. A host may use
// it only when its audited-operation policy explicitly disables persistence.
// It keeps the Service dependency explicit instead of treating a nil sink as
// an accidental configuration that silently loses audit records.
type NoopAuditSink struct{}

// Append satisfies AuditSink without persisting the event.
func (NoopAuditSink) Append(context.Context, AuditEvent) error { return nil }
