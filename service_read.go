package crud

import (
	"context"
	"strings"
)

// List returns a normalized, scoped, and authorized resource page.
func (service *Service) List(ctx context.Context, key ResourceKey, query Query) (Page, error) {
	state, err := service.resolveRead(ctx, key)
	if err != nil {
		return Page{}, err
	}
	normalized, err := normalizeQuery(ctx, state.definition, query)
	if err != nil {
		return Page{}, err
	}
	page, err := state.definition.Source.List(ctx, state.scope, normalized)
	if err != nil {
		return Page{}, unavailable(err)
	}
	return sanitizePage(state.definition, page), nil
}

// Get returns one scoped record. A record-specific authorization follows the
// preliminary authorization so policies can consider the loaded record.
func (service *Service) Get(ctx context.Context, key ResourceKey, id RecordID) (Record, error) {
	if id == "" {
		return Record{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	state, err := service.resolveRead(ctx, key)
	if err != nil {
		return Record{}, err
	}
	record, err := state.definition.Source.Get(ctx, state.scope, id)
	if err != nil {
		return Record{}, unavailable(err)
	}
	if err := service.authorizer.Authorize(ctx, state.principal, key, ActionRead, &record); err != nil {
		return Record{}, publicError(ErrorForbidden, "crud.error.forbidden", err)
	}
	return sanitizeRecord(state.definition, record), nil
}

// Lookup returns only the value and label shape declared by a lookup field.
func (service *Service) Lookup(ctx context.Context, key ResourceKey, fieldKey FieldKey, query LookupQuery) (LookupPage, error) {
	state, err := service.resolveRead(ctx, key)
	if err != nil {
		return LookupPage{}, err
	}
	field, ok := findField(state.definition.Fields, fieldKey)
	if !ok || field.Type != FieldLookup || field.Lookup == nil {
		return LookupPage{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	normalized, err := normalizeLookup(field.Lookup, query)
	if err != nil {
		return LookupPage{}, err
	}
	page, err := state.definition.Source.Lookup(ctx, state.scope, normalized)
	if err != nil {
		return LookupPage{}, unavailable(err)
	}
	return sanitizeLookupPage(page), nil
}

func normalizeQuery(ctx context.Context, definition Definition, query Query) (Query, error) {
	normalized := Query{Search: strings.TrimSpace(query.Search)}
	if len(query.Filters) > maxQueryFilters || len(query.Sort) > maxQuerySortTerms ||
		tooLong(normalized.Search, maxQuerySearchRunes) || len(query.Page.Cursor) > maxCursorBytes {
		return Query{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	if normalized.Search != "" && !definition.Source.Capabilities(ctx).Has(CapabilityContainsSearch) {
		return Query{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	for _, filter := range query.Filters {
		field, ok := findField(definition.Fields, filter.Field)
		if !ok || field.Sensitive || !filterAllowed(field.Type, filter.Operator) || !allowedValue(filter.Value) ||
			(filter.Operator == FilterIsNull && filter.Value != nil) {
			return Query{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
		}
		normalized.Filters = append(normalized.Filters, filter)
	}
	sort, err := normalizeSort(definition, query.Sort)
	if err != nil {
		return Query{}, err
	}
	normalized.Sort = sort
	page, err := normalizePage(definition.List.Pagination, query.Page)
	if err != nil {
		return Query{}, err
	}
	normalized.Page = page
	return normalized, nil
}

func normalizeSort(definition Definition, sort []Sort) ([]Sort, error) {
	if len(sort) == 0 {
		sort = definition.List.DefaultSort
	}
	allowed := make(map[FieldKey]struct{}, len(definition.List.Sortable))
	for _, field := range definition.List.Sortable {
		allowed[field] = struct{}{}
	}
	normalized := make([]Sort, 0, len(sort)+1)
	for _, term := range sort {
		if _, ok := allowed[term.Field]; !ok || (term.Direction != SortAscending && term.Direction != SortDescending) {
			return nil, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
		}
		normalized = append(normalized, term)
	}
	for _, term := range normalized {
		if term.Field == "id" {
			return normalized, nil
		}
	}
	return append(normalized, Sort{Field: "id", Direction: SortAscending}), nil
}

func normalizePage(definition PaginationDefinition, page PageRequest) (PageRequest, error) {
	if page.Mode != "" && page.Mode != definition.Mode {
		return PageRequest{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	normalized := PageRequest{Mode: definition.Mode, Size: page.Size}
	if normalized.Size == 0 {
		normalized.Size = definition.DefaultSize
	}
	if !containsPageSize(definition.AllowedSizes, normalized.Size) {
		return PageRequest{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	switch definition.Mode {
	case PageModeOffset:
		if page.Cursor != "" {
			return PageRequest{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
		}
		normalized.Number = page.Number
		if normalized.Number == 0 {
			normalized.Number = 1
		}
	case PageModeCursor:
		if page.Number != 0 {
			return PageRequest{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
		}
		normalized.Cursor = page.Cursor
	default:
		return PageRequest{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	return normalized, nil
}

func normalizeLookup(definition *LookupDefinition, query LookupQuery) (LookupQuery, error) {
	normalized := LookupQuery{Search: strings.TrimSpace(query.Search), Cursor: query.Cursor, Size: query.Size}
	if tooLong(normalized.Search, maxQuerySearchRunes) || len(normalized.Cursor) > maxCursorBytes ||
		len(query.Dependencies) > len(definition.Dependencies) {
		return LookupQuery{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	if normalized.Size == 0 {
		normalized.Size = definition.PageSize
	}
	if normalized.Size > definition.PageSize {
		return LookupQuery{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	for key, value := range query.Dependencies {
		if !containsField(definition.Dependencies, key) || !allowedValue(value) {
			return LookupQuery{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
		}
		normalized.Dependencies = cloneFields(query.Dependencies)
	}
	return normalized, nil
}

func filterAllowed(fieldType FieldType, operator FilterOperator) bool {
	if operator == FilterEqual || operator == FilterNotEqual || operator == FilterIsNull {
		return true
	}
	switch fieldType {
	case FieldString, FieldText, FieldEmail, FieldPhone, FieldEnum, FieldLookup:
		return operator == FilterContains || operator == FilterPrefix
	case FieldInteger, FieldDecimal, FieldDate, FieldDateTime:
		return operator == FilterLessThan || operator == FilterLessOrEqual || operator == FilterGreaterThan || operator == FilterGreaterOrEqual
	default:
		return false
	}
}

func sanitizePage(definition Definition, page Page) Page {
	records := make([]Record, len(page.Records))
	for index, record := range page.Records {
		records[index] = sanitizeRecord(definition, record)
	}
	page.Records = records
	return page
}

func sanitizeRecord(definition Definition, record Record) Record {
	fields := make(Fields, len(record.Fields))
	for key, value := range record.Fields {
		if _, ok := findField(definition.Fields, key); ok {
			fields[key] = value
		}
	}
	record.Fields = fields
	return record
}

func sanitizeLookupPage(page LookupPage) LookupPage {
	page.Options = append([]LookupOption(nil), page.Options...)
	return page
}

func findField(fields []Field, key FieldKey) (Field, bool) {
	for _, field := range fields {
		if field.Key == key {
			return field, true
		}
	}
	return Field{}, false
}

func containsField(fields []FieldKey, key FieldKey) bool {
	for _, field := range fields {
		if field == key {
			return true
		}
	}
	return false
}

func containsPageSize(sizes []uint16, size uint16) bool {
	for _, allowed := range sizes {
		if allowed == size {
			return true
		}
	}
	return false
}

func cloneFields(fields Fields) Fields {
	clone := make(Fields, len(fields))
	for key, value := range fields {
		clone[key] = value
	}
	return clone
}

func tooLong(value string, maximum int) bool {
	return len([]rune(value)) > maximum
}
