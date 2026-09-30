package crud

import (
	"context"
	"fmt"
	"sort"
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
	details, err := service.loadDetails(ctx, state, id, record.Fields)
	if err != nil {
		return Record{}, err
	}
	record = sanitizeRecord(state.definition, record)
	record.Details = details
	return record, nil
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
	target := state
	if field.Lookup.Resource != key {
		if _, exists := service.registry.Get(field.Lookup.Resource); exists {
			target, err = service.resolveRead(ctx, field.Lookup.Resource)
			if err != nil {
				return LookupPage{}, err
			}
		}
	}
	normalized.Resource = field.Lookup.Resource
	normalized.ValueField = field.Lookup.ValueField
	normalized.LabelField = field.Lookup.LabelField
	normalized.FixedFilters = cloneFixedLookupFilters(field.Lookup.FixedFilters)
	cacheKey := lookupCacheKey(key, fieldKey, target, normalized)
	page, err := service.loadLookup(ctx, cacheKey, target, normalized)
	if err != nil {
		return LookupPage{}, unavailable(err)
	}
	return sanitizeLookupPage(page), nil
}

type lookupFlight struct {
	done chan struct{}
	page LookupPage
	err  error
}

func (service *Service) loadLookup(ctx context.Context, cacheKey string, state readState, query LookupQuery) (LookupPage, error) {
	if page, ok := service.lookupCache.Get(cacheKey); ok {
		return page, nil
	}
	service.lookupMu.Lock()
	if flight, ok := service.lookupFlights[cacheKey]; ok {
		service.lookupMu.Unlock()
		select {
		case <-flight.done:
			return flight.page, flight.err
		case <-ctx.Done():
			return LookupPage{}, ctx.Err()
		}
	}
	flight := &lookupFlight{done: make(chan struct{})}
	service.lookupFlights[cacheKey] = flight
	service.lookupMu.Unlock()

	page, err := state.definition.Source.Lookup(ctx, state.scope, query)
	if err == nil {
		page = sanitizeLookupPage(page)
		service.lookupCache.Set(cacheKey, page)
	}
	service.lookupMu.Lock()
	flight.page, flight.err = page, err
	delete(service.lookupFlights, cacheKey)
	close(flight.done)
	service.lookupMu.Unlock()
	return page, err
}

func lookupCacheKey(parent ResourceKey, field FieldKey, state readState, query LookupQuery) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s|%s|%s|%s|%s|%d|%s", parent, field, query.Resource, query.ValueField, query.LabelField, query.Size, query.Cursor)
	builder.WriteString("|q=")
	builder.WriteString(query.Search)
	if state.definition.Scope.Mode != ScopeModeGlobal {
		keys := make([]string, 0, len(state.definition.Scope.Keys))
		for key := range state.scope {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(&builder, "|s:%s=%T:%v", key, state.scope[key], state.scope[key])
		}
	}
	keys := make([]string, 0, len(query.Dependencies))
	for key := range query.Dependencies {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := query.Dependencies[FieldKey(key)]
		fmt.Fprintf(&builder, "|d:%s=%T:%v", key, value, value)
	}
	for _, filter := range query.FixedFilters {
		fmt.Fprintf(&builder, "|f:%s", filter.Field)
		for _, value := range filter.Values {
			fmt.Fprintf(&builder, "=%T:%v", value, value)
		}
	}
	return builder.String()
}

func normalizeQuery(ctx context.Context, definition Definition, query Query) (Query, error) {
	if query.IncludeArchived && definition.Grid.ArchiveVisibility != ArchiveVisibilityActiveAndArchived {
		return Query{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	normalized := Query{Search: strings.TrimSpace(query.Search), IncludeArchived: query.IncludeArchived}
	if len(query.Filters) > maxQueryFilters || len(query.Sort) > maxQuerySortTerms ||
		tooLong(normalized.Search, maxQuerySearchRunes) || len(query.Page.Cursor) > maxCursorBytes {
		return Query{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	if normalized.Search != "" && !definition.Source.Capabilities(ctx).Has(CapabilityContainsSearch) {
		return Query{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
	}
	for _, filter := range query.Filters {
		field, ok := findField(definition.Fields, filter.Field)
		if !ok || field.Sensitive || !filterAllowed(field.Type, filter.Operator) || !validFilterShape(filter) {
			return Query{}, publicError(ErrorInvalidRequest, "crud.error.invalid_request", nil)
		}
		normalized.Filters = append(normalized.Filters, filter)
	}
	sort, err := normalizeSort(definition, query.Sort)
	if err != nil {
		return Query{}, err
	}
	normalized.Sort = sort
	page, err := normalizePage(definition.Grid.Pagination, query.Page)
	if err != nil {
		return Query{}, err
	}
	normalized.Page = page
	return normalized, nil
}

func normalizeSort(definition Definition, sort []Sort) ([]Sort, error) {
	if len(sort) == 0 {
		sort = definition.Grid.DefaultSort
	}
	allowed := make(map[FieldKey]struct{}, len(definition.Grid.Sortable))
	for _, field := range definition.Grid.Sortable {
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
	if operator == FilterEqual || operator == FilterNotEqual || operator == FilterIsNull || operator == FilterIn {
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

func validFilterShape(filter Filter) bool {
	if filter.Operator == FilterIn {
		if filter.Value != nil || len(filter.Values) == 0 || len(filter.Values) > 100 {
			return false
		}
		for _, value := range filter.Values {
			if !allowedValue(value) {
				return false
			}
		}
		return true
	}
	return len(filter.Values) == 0 && allowedValue(filter.Value) && (filter.Operator != FilterIsNull || filter.Value == nil)
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
		field, ok := findField(definition.Fields, key)
		if ok && field.Visible {
			fields[key] = value
		}
	}
	record.Fields = fields
	record.Details = nil
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
