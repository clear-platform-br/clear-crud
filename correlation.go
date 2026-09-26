package crud

import "context"

type correlationIDKey struct{}

// WithCorrelationID attaches a host-generated opaque correlation ID to the
// operation context. It is intended for transport adapters and audit sinks.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, correlationIDKey{}, id)
}

// CorrelationID returns the host-generated correlation ID, when present.
func CorrelationID(ctx context.Context) string {
	id, _ := ctx.Value(correlationIDKey{}).(string)
	return id
}
