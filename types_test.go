package crud

import "testing"

func TestPublicActionValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		action Action
		want   string
	}{
		"create": {action: ActionCreate, want: "create"},
		"read":   {action: ActionRead, want: "read"},
		"update": {action: ActionUpdate, want: "update"},
		"delete": {action: ActionDelete, want: "delete"},
		"help":   {action: ActionHelp, want: "help"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := string(test.action); got != test.want {
				t.Fatalf("action = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCapabilitiesHas(t *testing.T) {
	t.Parallel()

	capabilities := Capabilities{
		CapabilityAtomicVersion: {},
		CapabilityUnitOfWork:    {},
	}

	if !capabilities.Has(CapabilityAtomicVersion) {
		t.Fatal("atomic version capability must be present")
	}
	if capabilities.Has(CapabilityHardDelete) {
		t.Fatal("hard delete capability must not be present")
	}
}
