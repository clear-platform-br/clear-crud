package crud

import (
	"context"
	"math/big"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

const maxMutationFields = 100

func normalizeMutation(ctx context.Context, scope Scope, definition Definition, mutation Mutation) (Mutation, error) {
	return normalizeMutationForAction(ctx, scope, definition, ActionCreate, mutation)
}

func normalizeMutationForAction(ctx context.Context, scope Scope, definition Definition, action Action, mutation Mutation) (Mutation, error) {
	normalized, err := normalizeMutationFields(definition, action, mutation)
	if err != nil {
		return Mutation{}, err
	}
	if definition.Hooks.BeforeValidate != nil {
		normalized, err = definition.Hooks.BeforeValidate(ctx, normalized)
		if err != nil {
			return Mutation{}, unavailable(err)
		}
		normalized, err = normalizeMutationFields(definition, action, normalized)
		if err != nil {
			return Mutation{}, err
		}
	}
	if definition.Validator != nil {
		if fields := definition.Validator(ctx, cloneScope(scope), normalized); len(fields) != 0 {
			return Mutation{}, invalidMutation(fields)
		}
	}
	return normalized, nil
}

func normalizeMutationFields(definition Definition, action Action, mutation Mutation) (Mutation, error) {
	if len(mutation.Fields) > maxMutationFields {
		return Mutation{}, invalidMutation(nil)
	}
	normalized := Mutation{Fields: make(Fields, len(definition.Fields)), Details: cloneDetailMutations(mutation.Details)}
	for key, value := range mutation.Fields {
		field, ok := findField(definition.Fields, key)
		if !ok || field.ReadOnly || !field.Visible || field.CreateOnly && action != ActionCreate {
			return Mutation{}, invalidMutation(nil)
		}
		if err := validateFieldValue(field, value); err != nil {
			return Mutation{}, err
		}
		normalized.Fields[key] = value
	}
	for _, field := range definition.Fields {
		if field.ReadOnly || !field.Visible || field.CreateOnly && action != ActionCreate {
			continue
		}
		value, exists := normalized.Fields[field.Key]
		if !exists {
			if field.Required {
				return Mutation{}, invalidMutation(FieldErrors{field.Key: "crud.field.required"})
			}
			normalized.Fields[field.Key] = nil
			continue
		}
		if field.Required && value == nil {
			return Mutation{}, invalidMutation(FieldErrors{field.Key: "crud.field.required"})
		}
	}
	return normalized, nil
}

func cloneDetailMutations(details DetailMutations) DetailMutations {
	if details == nil {
		return nil
	}
	clone := make(DetailMutations, len(details))
	for key, mutations := range details {
		clone[key] = make([]DetailMutation, len(mutations))
		for index, mutation := range mutations {
			clone[key][index] = DetailMutation{
				ID:      mutation.ID,
				Version: mutation.Version,
				Delete:  mutation.Delete,
				Fields:  cloneFields(mutation.Fields),
			}
		}
	}
	return clone
}

func validateFieldValue(field Field, value Value) error {
	if value == nil {
		return nil
	}
	if !allowedValue(value) {
		return invalidMutation(FieldErrors{field.Key: "crud.field.invalid"})
	}
	switch field.Type {
	case FieldInteger:
		if _, ok := value.(int64); !ok {
			return invalidMutation(FieldErrors{field.Key: "crud.field.invalid"})
		}
	case FieldBoolean:
		if _, ok := value.(bool); !ok {
			return invalidMutation(FieldErrors{field.Key: "crud.field.invalid"})
		}
	default:
		if _, ok := value.(string); !ok {
			return invalidMutation(FieldErrors{field.Key: "crud.field.invalid"})
		}
	}
	text, isText := value.(string)
	if isText && (field.MinLength > 0 && len([]rune(text)) < field.MinLength || field.MaxLength > 0 && len([]rune(text)) > field.MaxLength) {
		return invalidMutation(FieldErrors{field.Key: "crud.field.length"})
	}
	if err := validateFieldFormat(field, text, isText); err != nil {
		return err
	}
	if field.Type == FieldEnum && !enumContains(field.Enum, value) {
		return invalidMutation(FieldErrors{field.Key: "crud.field.invalid"})
	}
	return validateFieldRange(field, value)
}

func validateFieldFormat(field Field, value string, isText bool) error {
	if !isText {
		return nil
	}
	switch field.Type {
	case FieldDecimal:
		if _, ok := new(big.Rat).SetString(value); !ok {
			return invalidMutation(FieldErrors{field.Key: "crud.field.invalid"})
		}
	case FieldDate:
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return invalidMutation(FieldErrors{field.Key: "crud.field.invalid"})
		}
	case FieldDateTime:
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return invalidMutation(FieldErrors{field.Key: "crud.field.invalid"})
		}
	case FieldEmail:
		address, err := mail.ParseAddress(value)
		if err != nil || address.Address != value {
			return invalidMutation(FieldErrors{field.Key: "crud.field.invalid"})
		}
	}
	return nil
}

func validateFieldRange(field Field, value Value) error {
	if field.Minimum == "" && field.Maximum == "" {
		return nil
	}
	valueText := fieldValueText(value)
	if field.Type == FieldInteger {
		integer := value.(int64)
		if field.Minimum != "" {
			minimum, err := strconv.ParseInt(field.Minimum, 10, 64)
			if err != nil || integer < minimum {
				return invalidMutation(FieldErrors{field.Key: "crud.field.range"})
			}
		}
		if field.Maximum != "" {
			maximum, err := strconv.ParseInt(field.Maximum, 10, 64)
			if err != nil || integer > maximum {
				return invalidMutation(FieldErrors{field.Key: "crud.field.range"})
			}
		}
		return nil
	}
	if field.Type == FieldDecimal {
		valueRatio, _ := new(big.Rat).SetString(valueText)
		if field.Minimum != "" {
			minimum, ok := new(big.Rat).SetString(field.Minimum)
			if !ok || valueRatio.Cmp(minimum) < 0 {
				return invalidMutation(FieldErrors{field.Key: "crud.field.range"})
			}
		}
		if field.Maximum != "" {
			maximum, ok := new(big.Rat).SetString(field.Maximum)
			if !ok || valueRatio.Cmp(maximum) > 0 {
				return invalidMutation(FieldErrors{field.Key: "crud.field.range"})
			}
		}
		return nil
	}
	if field.Minimum != "" && strings.Compare(valueText, field.Minimum) < 0 || field.Maximum != "" && strings.Compare(valueText, field.Maximum) > 0 {
		return invalidMutation(FieldErrors{field.Key: "crud.field.range"})
	}
	return nil
}

func fieldValueText(value Value) string {
	if integer, ok := value.(int64); ok {
		return strconv.FormatInt(integer, 10)
	}
	text, _ := value.(string)
	return text
}

func enumContains(options []Option, value Value) bool {
	for _, option := range options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func invalidMutation(fields FieldErrors) *Error {
	return &Error{Code: ErrorValidationFailed, Message: "crud.error.validation_failed", Fields: fields}
}
