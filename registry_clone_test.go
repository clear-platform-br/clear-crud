package crud

import (
	"context"
	"testing"
)

func TestRegistryClonesLookupAndEnumDetails(t *testing.T) {
	t.Parallel()
	definition := validDefinition("contact_categories")
	definition.Fields = append(definition.Fields,
		Field{Key: "kind", Label: "crud.kind", Type: FieldEnum, Visible: true, Enum: []Option{{Value: "standard", Label: "crud.kind.standard"}}},
		lookupField(),
	)
	definition.Source = fakeSource{capabilities: capabilitiesWithLookup()}

	registry := NewRegistry()
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	definition.Fields[2].Enum[0].Label = "changed"
	definition.Fields[3].Lookup.Dependencies[0] = "name"

	stored, ok := registry.Get(definition.Key)
	if !ok {
		t.Fatal("Get() did not return definition")
	}
	if stored.Fields[2].Enum[0].Label != "crud.kind.standard" || stored.Fields[3].Lookup.Dependencies[0] != "active" {
		t.Fatal("registry did not preserve the original nested values")
	}
	stored.Fields[2].Enum[0].Label = "returned-value-mutated"
	stored.Fields[3].Lookup.Dependencies[0] = "name"
	again, _ := registry.Get(definition.Key)
	if again.Fields[2].Enum[0].Label != "crud.kind.standard" || again.Fields[3].Lookup.Dependencies[0] != "active" {
		t.Fatal("Get() must clone enum and lookup details")
	}
}

func TestRegistryClonesNumericDisplayMetadata(t *testing.T) {
	t.Parallel()
	definition := validDefinition("formatted_values")
	definition.Fields[0].Type = FieldInteger
	definition.Fields[0].NumericDisplay = &NumericDisplay{Locale: "pt-BR", Currency: "BRL", Scale: 2, Prefix: "", Suffix: ""}

	registry := NewRegistry()
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	definition.Fields[0].NumericDisplay.Locale = "en-US"
	stored, _ := registry.Get(definition.Key)
	if stored.Fields[0].NumericDisplay.Locale != "pt-BR" {
		t.Fatal("registry must clone numeric display metadata during registration")
	}
	stored.Fields[0].NumericDisplay.Locale = "fr-FR"
	again, _ := registry.Get(definition.Key)
	if again.Fields[0].NumericDisplay.Locale != "pt-BR" {
		t.Fatal("registry must clone numeric display metadata when returning definitions")
	}
}

func TestRegistryPrecompilesFieldPatterns(t *testing.T) {
	t.Parallel()
	definition := validDefinition("patterned")
	definition.Fields[0].Pattern = `^[A-Z].*$`
	registry := NewRegistry()
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	stored, ok := registry.Get(definition.Key)
	if !ok || len(stored.fieldPatterns) != 1 || stored.fieldPatterns[definition.Fields[0].Key] == nil {
		t.Fatalf("stored field patterns = %#v, want one compiled pattern", stored.fieldPatterns)
	}
}

func TestRegistrySealIsIdempotent(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	registry.Seal()
	registry.Seal()
	if !registry.Sealed() {
		t.Fatal("registry must remain sealed")
	}
}

func capabilitiesWithLookup() Capabilities {
	capabilities := mutableCapabilities()
	capabilities[CapabilityLookup] = struct{}{}
	return capabilities
}
