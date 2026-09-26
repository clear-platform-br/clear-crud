package crud

import (
	"context"
	"testing"
)

func TestNoopAuditSink(t *testing.T) {
	if err := (NoopAuditSink{}).Append(context.Background(), AuditEvent{}); err != nil {
		t.Fatalf("Append() = %v", err)
	}
}
