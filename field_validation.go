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
		if field.Default != nil {
			if field.ReadOnly {
				return nil, invalidDefinition(path+".default", "is not allowed on read-only fields")
			}
			if !field.Visible {
				return nil, invalidDefinition(path+".default", "requires a visible field")
			}
			if field.Sensitive {
				return nil, invalidDefinition(path+".default", "is not allowed on sensitive fields")
			}
			if err := validateFieldValue(field, field.Default); err != nil {
				return nil, invalidDefinition(path+".default", "does not match the field type or constraints")
			}
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
	for index, field := range fields {
		if field.Lookup == nil {
			continue
		}
		for dependencyIndex, dependency := range field.Lookup.Dependencies {
			if _, ok := known[dependency]; !ok {
				return nil, invalidDefinition(fmt.Sprintf("fields[%d].lookup.dependencies[%d]", index, dependencyIndex), "must reference a declared field")
			}
		}
	}
	return known, nil
}

func validateFieldConfiguration(field Field, path string) error {
	if err := validatePatternDefinition(field, path); err != nil {
		return err
	}
	if field.EnumControl != "" && !validEnumControl(field.EnumControl) {
		return invalidDefinition(path+".enumControl", "is unknown")
	}
	if field.EnumControl != "" && field.Type != FieldEnum {
		return invalidDefinition(path+".enumControl", "is only allowed for enum fields")
	}
	if field.BooleanDisplay != nil {
		if field.Type != FieldBoolean {
			return invalidDefinition(path+".booleanDisplay", "is only allowed for boolean fields")
		}
		if field.BooleanDisplay.True == "" || field.BooleanDisplay.False == "" {
			return invalidDefinition(path+".booleanDisplay", "requires true and false symbols")
		}
	}
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

func validEnumControl(control EnumControl) bool {
	switch control {
	case EnumControlAuto, EnumControlSelect, EnumControlRadio, EnumControlSegmented, EnumControlButtons:
		return true
	default:
		return false
	}
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
	if lookup.MinSearchLength > maxLookupMinSearchLength {
		return invalidDefinition(path+".minSearchLength", "must be between 0 and 64")
	}
	seen := make(map[FieldKey]struct{}, len(lookup.FixedFilters))
	for index, filter := range lookup.FixedFilters {
		filterPath := fmt.Sprintf("%s.fixedFilters[%d]", path, index)
		if !validKey(string(filter.Field)) {
			return invalidDefinition(filterPath+".field", "must be a lowercase identifier")
		}
		if len(filter.Values) == 0 || len(filter.Values) > int(maxPageSize) {
			return invalidDefinition(filterPath+".values", "must contain between 1 and 100 values")
		}
		values := make(map[string]struct{}, len(filter.Values))
		for valueIndex, value := range filter.Values {
			if value == nil || !allowedValue(value) {
				return invalidDefinition(fmt.Sprintf("%s.values[%d]", filterPath, valueIndex), "must be string, bool, or int64")
			}
			identity := fmt.Sprintf("%T:%v", value, value)
			if _, exists := values[identity]; exists {
				return invalidDefinition(filterPath+".values", "must not contain duplicate values")
			}
			values[identity] = struct{}{}
		}
		if _, exists := seen[filter.Field]; exists {
			return invalidDefinition(path+".fixedFilters", "must not contain duplicate fields")
		}
		seen[filter.Field] = struct{}{}
	}
	return nil
}

func validateGrid(grid GridDefinition, fields map[FieldKey]Field) error {
	if grid.ArchiveVisibility != "" && grid.ArchiveVisibility != ArchiveVisibilityActiveOnly && grid.ArchiveVisibility != ArchiveVisibilityActiveAndArchived {
		return invalidDefinition("grid.archiveVisibility", "is unknown")
	}
	if len(grid.Columns) == 0 {
		return invalidDefinition("grid.columns", "must contain at least one field")
	}
	if err := validateFieldKeys("grid.columns", grid.Columns, fields); err != nil {
		return err
	}
	if err := validateFieldKeys("grid.searchable", grid.Searchable, fields); err != nil {
		return err
	}
	if err := validateFieldKeys("grid.sortable", grid.Sortable, fields); err != nil {
		return err
	}
	if len(grid.DefaultSort) == 0 {
		return invalidDefinition("grid.defaultSort", "must provide deterministic ordering")
	}
	sortable := make(map[FieldKey]struct{}, len(grid.Sortable))
	for _, key := range grid.Sortable {
		sortable[key] = struct{}{}
	}
	for index, sort := range grid.DefaultSort {
		if _, exists := sortable[sort.Field]; !exists {
			return invalidDefinition(fmt.Sprintf("grid.defaultSort[%d].field", index), "must be sortable")
		}
		if sort.Direction != SortAscending && sort.Direction != SortDescending {
			return invalidDefinition(fmt.Sprintf("grid.defaultSort[%d].direction", index), "is unknown")
		}
	}
	return validatePagination(grid.Pagination)
}

func validateForm(form FormDefinition, fields map[FieldKey]Field) error {
	if err := validateFieldKeys("form.fields", form.Fields, fields); err != nil {
		return err
	}
	for index, key := range form.Fields {
		if !fields[key].Visible {
			return invalidDefinition(fmt.Sprintf("form.fields[%d]", index), "must reference a visible field")
		}
	}
	return nil
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
		return invalidDefinition("grid.pagination.mode", "is unknown")
	}
	if pagination.DefaultSize == 0 || pagination.DefaultSize > maxPageSize {
		return invalidDefinition("grid.pagination.defaultSize", "must be between 1 and 100")
	}
	if len(pagination.AllowedSizes) == 0 || len(pagination.AllowedSizes) > maxAllowedPageSizes {
		return invalidDefinition("grid.pagination.allowedSizes", "must contain between 1 and 8 sizes")
	}
	previous := uint16(0)
	containsDefault := false
	for index, size := range pagination.AllowedSizes {
		if size == 0 || size > maxPageSize {
			return invalidDefinition(fmt.Sprintf("grid.pagination.allowedSizes[%d]", index), "must be between 1 and 100")
		}
		if index > 0 && size <= previous {
			return invalidDefinition("grid.pagination.allowedSizes", "must be strictly ascending")
		}
		if size == pagination.DefaultSize {
			containsDefault = true
		}
		previous = size
	}
	if !containsDefault {
		return invalidDefinition("grid.pagination.defaultSize", "must be one of allowed sizes")
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
