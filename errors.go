package crud

import "fmt"

// ErrorCode classifies a public, safe failure.
type ErrorCode string

const (
	ErrorInvalidRequest         ErrorCode = "invalid_request"
	ErrorUnauthenticated        ErrorCode = "unauthenticated"
	ErrorForbidden              ErrorCode = "forbidden"
	ErrorNotFound               ErrorCode = "not_found"
	ErrorValidationFailed       ErrorCode = "validation_failed"
	ErrorConflict               ErrorCode = "conflict"
	ErrorDeleteRestricted       ErrorCode = "delete_restricted"
	ErrorRateLimited            ErrorCode = "rate_limited"
	ErrorTemporarilyUnavailable ErrorCode = "temporarily_unavailable"
)

// Error is a sanitized error that can cross a transport boundary.
// Cause is intentionally omitted from JSON and Error() so driver, database,
// stack, and infrastructure details never reach an operator.
type Error struct {
	Code          ErrorCode                `json:"code"`
	Message       MessageCode              `json:"message"`
	Fields        map[FieldKey]MessageCode `json:"fields,omitempty"`
	CorrelationID string                   `json:"correlationId,omitempty"`
	Cause         error                    `json:"-"`
}

// Error implements error without exposing the internal cause or message.
func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("crud: %s", err.Code)
}

// Unwrap exposes the internal cause for protected diagnostics only.
func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// Is compares two clear.crud errors by their public code.
func (err *Error) Is(target error) bool {
	targetError, ok := target.(*Error)
	return ok && err != nil && targetError != nil && err.Code != "" && err.Code == targetError.Code
}
