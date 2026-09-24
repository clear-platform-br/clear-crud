package crud

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestErrorSanitizesCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("sqlite: no such table: residents")
	err := &Error{
		Code:    ErrorTemporarilyUnavailable,
		Message: "crud.unavailable",
		Cause:   cause,
	}

	if got := err.Error(); got != "crud: temporarily_unavailable" {
		t.Fatalf("Error() = %q", got)
	}
	if strings.Contains(err.Error(), "sqlite") {
		t.Fatalf("Error() leaked cause: %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is must reach the protected cause")
	}

	encoded, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("marshal error: %v", marshalErr)
	}
	if strings.Contains(string(encoded), "sqlite") {
		t.Fatalf("JSON leaked cause: %s", encoded)
	}
}

func TestErrorMatchesPublicCode(t *testing.T) {
	t.Parallel()

	err := &Error{Code: ErrorConflict}
	if !errors.Is(err, &Error{Code: ErrorConflict}) {
		t.Fatal("same public error code must match")
	}
	if errors.Is(err, &Error{Code: ErrorNotFound}) {
		t.Fatal("different public error codes must not match")
	}
}

func TestNilError(t *testing.T) {
	t.Parallel()

	var err *Error
	if got := err.Error(); got != "" {
		t.Fatalf("nil Error() = %q, want empty string", got)
	}
	if err.Unwrap() != nil {
		t.Fatal("nil Unwrap() must return nil")
	}
}
