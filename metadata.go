package crud

import (
	"context"
	"fmt"
	"strconv"
)

// SourceMetadata contains structural facts discovered by a registered data
// source. Adapters may provide enum values and read-only fields from their
// registered read model at startup. Metadata is never read from an HTTP
// request.
type SourceMetadata struct {
	Fields map[FieldKey]FieldMetadata
}

// FieldMetadata contains adapter-discovered constraints for one public field.
// ReadOnly is useful for fields supplied by a join or calculation: the adapter
// returns the value for reads, while the core rejects it in mutations.
type FieldMetadata struct {
	Enum     []Value
	ReadOnly bool
}

// MetadataSource is an optional adapter capability used during registration.
// It lets a source contribute safe schema facts without coupling the core to a
// particular database or schema language.
type MetadataSource interface {
	Metadata(context.Context) (SourceMetadata, error)
}

func applySourceMetadata(ctx context.Context, definition Definition) (Definition, error) {
	if isNil(definition.Source) {
		return definition, nil
	}
	inspector, ok := definition.Source.(MetadataSource)
	if !ok || isNil(inspector) {
		return definition, nil
	}
	metadata, err := inspector.Metadata(ctx)
	if err != nil {
		return Definition{}, err
	}
	if len(metadata.Fields) == 0 {
		return definition, nil
	}
	clone := cloneDefinition(definition)
	for index := range clone.Fields {
		field := &clone.Fields[index]
		facts, ok := metadata.Fields[field.Key]
		if !ok {
			continue
		}
		if facts.ReadOnly {
			field.ReadOnly = true
		}
		if len(facts.Enum) == 0 {
			continue
		}
		if field.Type == FieldBoolean {
			continue
		}
		if field.Lookup != nil {
			// A consumer-declared lookup is more specific than an enum
			// inferred from source metadata.
			continue
		}
		options, err := metadataOptions(field.Key, facts.Enum)
		if err != nil {
			return Definition{}, err
		}
		if len(field.Enum) == 0 {
			field.Type = FieldEnum
			field.Enum = options
			continue
		}
		if field.Type != FieldEnum {
			return Definition{}, invalidDefinition(fmt.Sprintf("fields[%d].enum", index), "requires enum field type")
		}
		for optionIndex, option := range field.Enum {
			if !containsEnumValue(facts.Enum, option.Value) {
				return Definition{}, invalidDefinition(fmt.Sprintf("fields[%d].enum[%d].value", index, optionIndex), "is not allowed by the source schema")
			}
		}
	}
	return clone, nil
}

func metadataOptions(field FieldKey, values []Value) ([]Option, error) {
	options := make([]Option, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		identity := fmt.Sprintf("%T:%v", value, value)
		if _, exists := seen[identity]; exists {
			return nil, invalidDefinition(fmt.Sprintf("fields.%s.enum[%d]", field, index), "contains duplicate values")
		}
		seen[identity] = struct{}{}
		options = append(options, Option{Value: value, Label: MessageCode(metadataValueLabel(value))})
	}
	return options, nil
}

func containsEnumValue(values []Value, wanted Value) bool {
	wantedIdentity := fmt.Sprintf("%T:%v", wanted, wanted)
	for _, value := range values {
		if fmt.Sprintf("%T:%v", value, value) == wantedIdentity {
			return true
		}
	}
	return false
}

func metadataValueLabel(value Value) string {
	switch typed := value.(type) {
	case nil:
		return "∅"
	case string:
		if typed == "" {
			return "∅"
		}
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return fmt.Sprint(typed)
	}
}
