package crud

import (
	"context"
	"errors"
	"testing"
)

type metadataTestSource struct {
	*recordingSource
	metadata SourceMetadata
	err      error
}

func (source *metadataTestSource) Metadata(context.Context) (SourceMetadata, error) {
	return source.metadata, source.err
}

func TestRegistryUsesSourceMetadataAsTheDefaultEnum(t *testing.T) {
	definition := validDefinition("metadata_contacts")
	source := &metadataTestSource{
		recordingSource: &recordingSource{capabilities: mutableCapabilities()},
		metadata: SourceMetadata{Fields: map[FieldKey]FieldMetadata{
			"name":   {Enum: []Value{"new", true, int64(2), ""}},
			"active": {Enum: []Value{int64(0), int64(1)}},
		}},
	}
	definition.Source = source
	registry := NewRegistry()
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatal(err)
	}
	registered, ok := registry.Get(definition.Key)
	if !ok {
		t.Fatal("metadata definition not registered")
	}
	field := registered.Fields[0]
	if field.Type != FieldEnum || len(field.Enum) != 4 || field.Enum[0].Label != "new" || field.Enum[1].Label != "true" || field.Enum[2].Label != "2" || field.Enum[3].Label != "∅" {
		t.Fatalf("automatic enum metadata = %#v", field)
	}
	if registered.Fields[1].Type != FieldBoolean {
		t.Fatalf("boolean check metadata changed field type = %q", registered.Fields[1].Type)
	}

	mutation := Mutation{Fields: Fields{"name": int64(2), "active": false}}
	if _, err := normalizeMutation(context.Background(), Scope{}, registered, mutation); err != nil {
		t.Fatalf("integer enum value rejected: %v", err)
	}
}

func TestRegistryValidatesExplicitEnumAgainstSourceMetadata(t *testing.T) {
	definition := validDefinition("metadata_contacts")
	definition.Fields[0].Type = FieldEnum
	definition.Fields[0].Enum = []Option{{Value: "new", Label: "Novo"}}
	definition.Source = &metadataTestSource{
		recordingSource: &recordingSource{capabilities: mutableCapabilities()},
		metadata:        SourceMetadata{Fields: map[FieldKey]FieldMetadata{"name": {Enum: []Value{"new", "closed"}}}},
	}
	if err := NewRegistry().Register(context.Background(), definition); err != nil {
		t.Fatal(err)
	}
	definition.Fields[0].Enum = []Option{{Value: "cancelled", Label: "Cancelado"}}
	if err := NewRegistry().Register(context.Background(), definition); err == nil {
		t.Fatal("enum value outside source metadata accepted")
	}
}

func TestRegistryPreservesExplicitLookupAgainstSourceMetadata(t *testing.T) {
	definition := validDefinition("metadata_lookup_contacts")
	definition.Fields[0].Type = FieldLookup
	definition.Fields[0].Lookup = &LookupDefinition{Resource: "reference_values", ValueField: "id", LabelField: "name", PageSize: 25}
	definition.Source = &metadataTestSource{
		recordingSource: &recordingSource{capabilities: capabilitiesWithLookup()},
		metadata:        SourceMetadata{Fields: map[FieldKey]FieldMetadata{"name": {Enum: []Value{"new", "closed"}}}},
	}
	registry := NewRegistry()
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatal(err)
	}
	registered, ok := registry.Get(definition.Key)
	if !ok {
		t.Fatal("lookup definition not registered")
	}
	field := registered.Fields[0]
	if field.Type != FieldLookup || field.Lookup == nil || len(field.Enum) != 0 {
		t.Fatalf("explicit lookup was replaced by metadata enum = %#v", field)
	}
}

func TestRegistryMarksRegisteredReadModelFieldsReadOnly(t *testing.T) {
	definition := validDefinition("read_model_contacts")
	definition.Fields = append(definition.Fields, Field{
		Key: "customer_label", Label: "crud.read_model_contacts.customer_label",
		Type: FieldString, Visible: true,
	})
	definition.Grid.Columns = []FieldKey{"name", "customer_label", "active"}
	definition.Source = &metadataTestSource{
		recordingSource: &recordingSource{capabilities: mutableCapabilities()},
		metadata: SourceMetadata{Fields: map[FieldKey]FieldMetadata{
			"customer_label": {ReadOnly: true},
		}},
	}
	registry := NewRegistry()
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatal(err)
	}
	registered, ok := registry.Get(definition.Key)
	if !ok {
		t.Fatal("read model definition not registered")
	}
	field, ok := findField(registered.Fields, "customer_label")
	if !ok || !field.ReadOnly {
		t.Fatalf("registered read model field = %#v, want read-only", field)
	}
	if _, err := normalizeMutation(context.Background(), Scope{"tenant_id": "tenant-a"}, registered, Mutation{Fields: Fields{"name": "Ana", "active": true, "customer_label": "forged"}}); err == nil {
		t.Fatal("read-only read model field accepted in mutation")
	}
}

func TestRegistryReturnsMetadataErrorsAndRejectsDuplicateValues(t *testing.T) {
	definition := validDefinition("metadata_contacts")
	definition.Source = &metadataTestSource{
		recordingSource: &recordingSource{capabilities: mutableCapabilities()},
		err:             errors.New("schema unavailable"),
	}
	if err := NewRegistry().Register(context.Background(), definition); err == nil || err.Error() != "schema unavailable" {
		t.Fatalf("metadata error = %v", err)
	}
	definition.Source = &metadataTestSource{
		recordingSource: &recordingSource{capabilities: mutableCapabilities()},
		metadata:        SourceMetadata{Fields: map[FieldKey]FieldMetadata{"name": {Enum: []Value{"new", "new"}}}},
	}
	if err := NewRegistry().Register(context.Background(), definition); err == nil {
		t.Fatal("duplicate metadata values accepted")
	}
}
