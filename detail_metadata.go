package crud

import (
	"fmt"
	"strings"
)

// resolveDetailFields applies server-owned parent metadata to fields already
// declared by a child resource. It is deliberately independent of persistence,
// table names and product vocabulary.
func resolveDetailFields(parent Definition, detail DetailDefinition, child Definition, parentFields Fields) ([]Field, error) {
	fields := make([]Field, len(child.Fields))
	copy(fields, child.Fields)
	if len(detail.FieldMetadata) == 0 {
		return fields, nil
	}
	for _, source := range detail.FieldMetadata {
		index := fieldIndex(fields, source.Field)
		if index < 0 || fields[index].Sensitive || !fields[index].Visible {
			return nil, invalidDefinition("details.field_metadata.field", "must reference a visible, non-sensitive child field")
		}
		if source.LabelField != "" {
			if _, ok := findMetadataField(parent.Fields, source.LabelField); !ok {
				return nil, invalidDefinition("details.field_metadata.label_field", "must reference a parent field")
			}
			if label, ok := nonEmptyString(parentFields[source.LabelField]); ok {
				fields[index].Visible = true
				fields[index].DisplayLabel = label
			} else {
				// A mapped slot with no catalog label is intentionally unused.
				fields[index].Visible = false
				fields[index].DisplayLabel = ""
			}
		}
		if source.TypeField != "" {
			if _, ok := findMetadataField(parent.Fields, source.TypeField); !ok {
				return nil, invalidDefinition("details.field_metadata.type_field", "must reference a parent field")
			}
			if typeName, ok := nonEmptyString(parentFields[source.TypeField]); ok {
				fieldType, err := parseDetailFieldType(typeName)
				if err != nil {
					return nil, invalidMutation(FieldErrors{source.Field: "crud.field.invalid"})
				}
				if (fieldType == FieldEnum && len(fields[index].Enum) == 0) || (fieldType == FieldLookup && fields[index].Lookup == nil) {
					return nil, invalidMutation(FieldErrors{source.Field: "crud.field.invalid"})
				}
				previousType := fields[index].Type
				fields[index].Type = fieldType
				if fieldType != previousType {
					fields[index].Enum = nil
					fields[index].Lookup = nil
					fields[index].EnumControl = ""
					fields[index].Pattern = ""
					fields[index].PatternMessage = ""
					fields[index].Minimum = ""
					fields[index].Maximum = ""
					fields[index].MinLength = 0
					fields[index].MaxLength = 0
				}
			}
		}
		if source.RequiredField != "" {
			if _, ok := findMetadataField(parent.Fields, source.RequiredField); !ok {
				return nil, invalidDefinition("details.field_metadata.required_field", "must reference a parent field")
			}
			if required, ok := booleanMetadata(parentFields[source.RequiredField]); ok {
				fields[index].Required = required
				fields[index].Optional = !required
			}
		}
	}
	return fields, nil
}

func validateDetailFieldMetadata(parent Definition, detailIndex int, detail DetailDefinition, child Definition) error {
	seen := make(map[FieldKey]struct{}, len(detail.FieldMetadata))
	for index, source := range detail.FieldMetadata {
		path := fmt.Sprintf("details[%d].field_metadata[%d]", detailIndex, index)
		if source.Field == "" {
			return invalidDefinition(path+".field", "is required")
		}
		if _, exists := seen[source.Field]; exists {
			return invalidDefinition(path+".field", "must be unique")
		}
		seen[source.Field] = struct{}{}
		field, ok := findField(child.Fields, source.Field)
		if !ok || field.ReadOnly || !field.Visible || field.Sensitive {
			return invalidDefinition(path+".field", "must reference a visible, non-sensitive child field")
		}
		for _, mapping := range []struct {
			name string
			key  FieldKey
		}{{"label_field", source.LabelField}, {"type_field", source.TypeField}, {"required_field", source.RequiredField}} {
			name, key := mapping.name, mapping.key
			if key == "" {
				continue
			}
			parentField, ok := findMetadataField(parent.Fields, key)
			if !ok || parentField.Sensitive {
				return invalidDefinition(path+"."+name, "must reference a non-sensitive parent field")
			}
		}
	}
	return nil
}

func findMetadataField(fields []Field, key FieldKey) (Field, bool) {
	field, ok := findField(fields, key)
	return field, ok
}

func fieldIndex(fields []Field, key FieldKey) int {
	for index, field := range fields {
		if field.Key == key {
			return index
		}
	}
	return -1
}

func nonEmptyString(value Value) (string, bool) {
	text, ok := value.(string)
	text = strings.TrimSpace(text)
	return text, ok && text != ""
}

func booleanMetadata(value Value) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case int:
		return typed != 0, typed == 0 || typed == 1
	case int64:
		return typed != 0, typed == 0 || typed == 1
	case uint64:
		return typed != 0, typed == 0 || typed == 1
	default:
		return false, false
	}
}

func parseDetailFieldType(value string) (FieldType, error) {
	typeName := FieldType(strings.ToLower(strings.TrimSpace(value)))
	switch typeName {
	case FieldString, FieldText, FieldInteger, FieldDecimal, FieldBoolean, FieldDate, FieldDateTime, FieldEmail, FieldPhone, FieldEnum, FieldLookup:
		return typeName, nil
	default:
		return "", fmt.Errorf("unsupported detail field type %q", value)
	}
}
