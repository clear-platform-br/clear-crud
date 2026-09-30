package crud

import (
	"context"
	"errors"
	"testing"
)

func TestResolveDetailFieldsAppliesParentMetadataWithoutCreatingChildFields(t *testing.T) {
	parent := Definition{Fields: []Field{
		{Key: "value_1_label", Type: FieldString, Visible: true},
		{Key: "value_1_type", Type: FieldString, Visible: true},
		{Key: "value_1_required", Type: FieldBoolean, Visible: true},
	}}
	child := Definition{Fields: []Field{
		{Key: "value_1", Label: "crud.value_1", Type: FieldString, Optional: true, Visible: true},
		{Key: "parent_id", Label: "crud.parent_id", Type: FieldString, ReadOnly: true, Visible: false},
	}}
	detail := DetailDefinition{Key: "items", FieldMetadata: []DetailFieldMetadataSource{{
		Field: "value_1", LabelField: "value_1_label", TypeField: "value_1_type", RequiredField: "value_1_required",
	}}}
	fields, err := resolveDetailFields(parent, detail, child, Fields{
		"value_1_label":    "Nome da opção",
		"value_1_type":     "text",
		"value_1_required": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != len(child.Fields) || fields[0].DisplayLabel != "Nome da opção" || fields[0].Type != FieldText || !fields[0].IsRequired() {
		t.Fatalf("resolved fields = %#v", fields)
	}
	if fields[1].Visible || fields[1].DisplayLabel != "" {
		t.Fatalf("child technical field changed = %#v", fields[1])
	}
}

func TestResolveDetailFieldsHidesUnusedMappedSlots(t *testing.T) {
	parent := Definition{Fields: []Field{{Key: "value_2_label", Type: FieldString, Visible: true}}}
	child := Definition{Fields: []Field{{Key: "value_2", Label: "crud.value_2", Type: FieldString, Visible: true}}}
	detail := DetailDefinition{FieldMetadata: []DetailFieldMetadataSource{{Field: "value_2", LabelField: "value_2_label"}}}
	fields, err := resolveDetailFields(parent, detail, child, Fields{"value_2_label": ""})
	if err != nil {
		t.Fatal(err)
	}
	if fields[0].Visible {
		t.Fatal("unused mapped slot remained visible")
	}
}

func TestResolveDetailFieldsRejectsUnknownChildSlotAtRuntime(t *testing.T) {
	parent := Definition{Fields: []Field{{Key: "label", Type: FieldString, Visible: true}}}
	child := Definition{Fields: []Field{{Key: "value", Type: FieldString, Visible: true}}}
	detail := DetailDefinition{FieldMetadata: []DetailFieldMetadataSource{{Field: "missing"}}}
	if _, err := resolveDetailFields(parent, detail, child, nil); err == nil {
		t.Fatal("unknown child slot accepted at runtime")
	}
}

func TestResolveDetailFieldsRejectsUnsupportedRuntimeType(t *testing.T) {
	parent := Definition{Fields: []Field{{Key: "type", Type: FieldString, Visible: true}}}
	child := Definition{Fields: []Field{{Key: "value", Type: FieldString, Visible: true}}}
	detail := DetailDefinition{FieldMetadata: []DetailFieldMetadataSource{{Field: "value", TypeField: "type"}}}
	_, err := resolveDetailFields(parent, detail, child, Fields{"type": "money"})
	if err == nil {
		t.Fatal("unsupported runtime type accepted")
	}
	var public *Error
	if !errors.As(err, &public) || public.Code != ErrorValidationFailed {
		t.Fatalf("runtime type error = %#v", err)
	}
}

func TestDetailMetadataHelpersAcceptCanonicalBooleanAndFieldTypes(t *testing.T) {
	for _, value := range []Value{false, true, int64(0), int64(1), uint64(0), uint64(1)} {
		if _, ok := booleanMetadata(value); !ok {
			t.Fatalf("boolean metadata rejected %#v", value)
		}
	}
	if _, ok := booleanMetadata("true"); ok {
		t.Fatal("string boolean metadata accepted")
	}
	for _, typeName := range []string{"string", "text", "integer", "decimal", "boolean", "date", "datetime", "email", "phone", "enum", "lookup"} {
		if got, err := parseDetailFieldType(typeName); err != nil || string(got) != typeName {
			t.Fatalf("field type %q = %q, %v", typeName, got, err)
		}
	}
}

func TestValidateDetailFieldMetadataRejectsUnknownMapping(t *testing.T) {
	parent := validDefinition("metadata_parent")
	child := validDefinition("metadata_child")
	child.Fields = append(child.Fields, Field{Key: "parent_id", Label: "crud.parent_id", Type: FieldString, ReadOnly: true, Visible: false})
	parent.Details = []DetailDefinition{{
		Key: "items", Resource: child.Key, ParentField: "parent_id", Maximum: 1,
		FieldMetadata: []DetailFieldMetadataSource{{Field: "missing", LabelField: "name"}},
	}}
	registry := NewRegistry()
	if err := registry.Register(context.Background(), child); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(context.Background(), parent); err != nil {
		t.Fatal(err)
	}
	if err := validateMasterDetailDefinitions(registry); err == nil {
		t.Fatal("unknown detail metadata mapping accepted")
	}
}

func TestValidateDetailFieldMetadataAcceptsPublicMapping(t *testing.T) {
	parent := Definition{Fields: []Field{{Key: "label", Type: FieldString, Visible: true}}}
	child := Definition{Fields: []Field{{Key: "value", Type: FieldString, Visible: true}}}
	detail := DetailDefinition{FieldMetadata: []DetailFieldMetadataSource{{Field: "value", LabelField: "label"}}}
	if err := validateDetailFieldMetadata(parent, 0, detail, child); err != nil {
		t.Fatal(err)
	}
}
