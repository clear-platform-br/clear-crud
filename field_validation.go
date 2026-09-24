package crud

import "fmt"

func validateFields(fields []Field) (map[FieldKey]Field, error) {
	if len(fields) == 0 {
		return nil, invalidDefinition("fields", "must contain at least one field")
	}
	known := make(map[FieldKey]Field, len(fields))
	visible := false
	for index, field := range fields {
		path := fmt.Sprintf("fields[%d]", index)
		if !validKey(string(field.Key)) {
			return nil, invalidDefinition(path+".key", "must be a valid key")
		}
		if _, exists := known[field.Key]; exists {
			return nil, invalidDefinition("fields", "must not contain duplicate keys")
		}
		if field.Label == "" {
			return nil, invalidDefinition(path+".label", "is required")
		}
		if !knownFieldType(field.Type) {
			return nil, invalidDefinition(path+".type", "is unknown")
		}
		if field.MinLength < 0 || field.MaxLength < 0 || (field.MaxLength > 0 && field.MinLength > field.MaxLength) {
			return nil, invalidDefinition(path+".length", "has an invalid range")
		}
		if err := validateFieldConfiguration(field, path); err != nil {
			return nil, err
		}
		known[field.Key] = field
		visible = visible || field.Visible
	}
	if !visible {
		return nil, invalidDefinition("fields", "must contain at least one visible field")
	}
	if err := validateFieldReferences(fields, known); err != nil {
		return nil, err
	}
	return known, nil
}

func validateFieldConfiguration(field Field, path string) error {
	if field.Type == FieldEnum {
		return validateOptions(field.Enum, path+".enum")
	}
	if len(field.Enum) != 0 {
		return invalidDefinition(path+".enum", "is only allowed for enum fields")
	}
	if field.Type == FieldLookup {
		return validateLookup(field.Lookup, path+".lookup")
	}
	if field.Lookup != nil {
		return invalidDefinition(path+".lookup", "is only allowed for lookup fields")
	}
	return nil
}

func validateOptions(options []Option, path string) error {
	if len(options) == 0 {
		return invalidDefinition(path, "must contain at least one option")
	}
	seen := make(map[string]struct{}, len(options))
	for index, option := range options {
		if !allowedValue(option.Value) {
			return invalidDefinition(fmt.Sprintf("%s[%d].value", path, index), "must be nil, string, bool, or int64")
		}
		if option.Label == "" {
			return invalidDefinition(fmt.Sprintf("%s[%d].label", path, index), "is required")
		}
		identity := fmt.Sprintf("%T:%v", option.Value, option.Value)
		if _, exists := seen[identity]; exists {
			return invalidDefinition(path, "must not contain duplicate values")
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func validateLookup(lookup *LookupDefinition, path string) error {
	if lookup == nil {
		return invalidDefinition(path, "is required")
	}
	if !validKey(string(lookup.Resource)) {
		return invalidDefinition(path+".resource", "must be a lowercase identifier")
	}
	if lookup.ValueField == "" || lookup.LabelField == "" {
		return invalidDefinition(path, "requires value and label fields")
	}
	if lookup.PageSize == 0 || lookup.PageSize > 100 {
		return invalidDefinition(path+".pageSize", "must be between 1 and 100")
	}
	return nil
}

func validateList(list ListDefinition, fields map[FieldKey]Field) error {
	if len(list.Columns) == 0 {
		return invalidDefinition("list.columns", "must contain at least one field")
	}
	if err := validateFieldKeys("list.columns", list.Columns, fields); err != nil {
		return err
	}
	if err := validateFieldKeys("list.searchable", list.Searchable, fields); err != nil {
		return err
	}
	if err := validateFieldKeys("list.sortable", list.Sortable, fields); err != nil {
		return err
	}
	if len(list.DefaultSort) == 0 {
		return invalidDefinition("list.defaultSort", "must provide deterministic ordering")
	}
	sortable := make(map[FieldKey]struct{}, len(list.Sortable))
	for _, key := range list.Sortable {
		sortable[key] = struct{}{}
	}
	for index, sort := range list.DefaultSort {
		if _, exists := sortable[sort.Field]; !exists {
			return invalidDefinition(fmt.Sprintf("list.defaultSort[%d].field", index), "must be sortable")
		}
		if sort.Direction != SortAscending && sort.Direction != SortDescending {
			return invalidDefinition(fmt.Sprintf("list.defaultSort[%d].direction", index), "is unknown")
		}
	}
	return validatePagination(list.Pagination)
}

func validateFieldKeys(path string, keys []FieldKey, fields map[FieldKey]Field) error {
	seen := make(map[FieldKey]struct{}, len(keys))
	for index, key := range keys {
		if _, exists := fields[key]; !exists {
			return invalidDefinition(fmt.Sprintf("%s[%d]", path, index), "does not reference a declared field")
		}
		if _, exists := seen[key]; exists {
			return invalidDefinition(path, "must not contain duplicates")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validatePagination(pagination PaginationDefinition) error {
	if pagination.Mode != PageModeOffset && pagination.Mode != PageModeCursor {
		return invalidDefinition("list.pagination.mode", "is unknown")
	}
	if pagination.DefaultSize != 25 {
		return invalidDefinition("list.pagination.defaultSize", "must be 25")
	}
	if len(pagination.AllowedSizes) != 3 || pagination.AllowedSizes[0] != 25 || pagination.AllowedSizes[1] != 50 || pagination.AllowedSizes[2] != 100 {
		return invalidDefinition("list.pagination.allowedSizes", "must be [25, 50, 100]")
	}
	return nil
}

func validateFieldReferences(fields []Field, known map[FieldKey]Field) error {
	for index, field := range fields {
		if field.Lookup == nil {
			continue
		}
		seen := make(map[FieldKey]struct{}, len(field.Lookup.Dependencies))
		for dependencyIndex, key := range field.Lookup.Dependencies {
			path := fmt.Sprintf("fields[%d].lookup.dependencies[%d]", index, dependencyIndex)
			if _, exists := known[key]; !exists {
				return invalidDefinition(path, "does not reference a declared field")
			}
			if key == field.Key {
				return invalidDefinition(path, "must not reference itself")
			}
			if _, exists := seen[key]; exists {
				return invalidDefinition("fields["+fmt.Sprint(index)+"].lookup.dependencies", "must not contain duplicates")
			}
			seen[key] = struct{}{}
		}
	}
	return nil
}

func knownFieldType(fieldType FieldType) bool {
	switch fieldType {
	case FieldString, FieldText, FieldInteger, FieldDecimal, FieldBoolean, FieldDate, FieldDateTime, FieldEmail, FieldPhone, FieldEnum, FieldLookup:
		return true
	default:
		return false
	}
}

func allowedValue(value Value) bool {
	switch value.(type) {
	case nil, string, bool, int64:
		return true
	default:
		return false
	}
}
