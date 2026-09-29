package crud

import (
	"context"
	"testing"
)

func TestFieldsAreRequiredByDefaultAndOptionalIsExplicit(t *testing.T) {
	definition := Definition{Fields: []Field{{Key: "name", Type: FieldString, Visible: true}}}
	if _, err := normalizeMutation(context.Background(), Scope{}, definition, Mutation{Fields: Fields{}}); err == nil {
		t.Fatal("field without an explicit optional exception was accepted empty")
	}
	definition.Fields[0].Optional = true
	if mutation, err := normalizeMutation(context.Background(), Scope{}, definition, Mutation{Fields: Fields{}}); err != nil || mutation.Fields["name"] != nil {
		t.Fatalf("explicit optional field = %#v, %v", mutation, err)
	}
}

func TestPublicFieldExposesTheGenericRequiredDefault(t *testing.T) {
	if !publicField(Field{Key: "name"}).Required {
		t.Fatal("default field was not exposed as required to the renderer")
	}
	if publicField(Field{Key: "notes", Optional: true}).Required {
		t.Fatal("explicit optional field was exposed as required")
	}
}

func TestPublicFieldClonesBooleanDisplay(t *testing.T) {
	original := Field{Key: "enabled", BooleanDisplay: &BooleanDisplay{True: "●", False: "○"}}
	clone := publicField(original)
	if clone.BooleanDisplay == original.BooleanDisplay || clone.BooleanDisplay.True != "●" || clone.BooleanDisplay.False != "○" {
		t.Fatalf("boolean display clone = %#v", clone.BooleanDisplay)
	}
}

func TestCreateNormalizationAppliesStaticDefaultsWithoutChangingUpdates(t *testing.T) {
	definition := Definition{Fields: []Field{
		{Key: "name", Type: FieldString, Visible: true},
		{Key: "status", Type: FieldEnum, Visible: true, Enum: []Option{{Value: "new", Label: "crud.status.new"}, {Value: "closed", Label: "crud.status.closed"}}, Default: "new"},
		{Key: "enabled", Type: FieldBoolean, Visible: true, Default: true},
	}}
	normalized, err := normalizeMutationForAction(context.Background(), Scope{}, definition, ActionCreate, Mutation{Fields: Fields{"name": "Ana"}})
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Fields["status"] != "new" || normalized.Fields["enabled"] != true {
		t.Fatalf("create defaults = %#v", normalized.Fields)
	}
	overridden, err := normalizeMutationForAction(context.Background(), Scope{}, definition, ActionCreate, Mutation{Fields: Fields{"name": "Ana", "status": "closed", "enabled": false}})
	if err != nil || overridden.Fields["status"] != "closed" || overridden.Fields["enabled"] != false {
		t.Fatalf("explicit create values = %#v, %v", overridden.Fields, err)
	}
	if _, err := normalizeMutationForAction(context.Background(), Scope{}, definition, ActionUpdate, Mutation{Fields: Fields{"name": "Ana"}}); err == nil {
		t.Fatal("update unexpectedly applied create defaults")
	}
}

func TestDefinitionRejectsUnsafeStaticDefaults(t *testing.T) {
	definition := validDefinition("defaulted")
	definition.Fields[0].Default = int64(1)
	if err := ValidateDefinition(context.Background(), definition); err == nil {
		t.Fatal("invalid default type accepted")
	}
	definition = validDefinition("defaulted")
	definition.Fields[0].Sensitive = true
	definition.Fields[0].Default = "secret"
	if err := ValidateDefinition(context.Background(), definition); err == nil {
		t.Fatal("sensitive default accepted")
	}
}
