// Package httpadapter exposes the closed clear.crud HTTP v1 surface.
package httpadapter

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	crud "github.com/clear-platform-br/clear-crud"
)

const maxJSONBytes int64 = 1 << 20

// Options configures host-owned transport details.
type Options struct {
	Translator    crud.Translator
	CorrelationID func(*http.Request) string
	MaxBodyBytes  int64
}

// Handler adapts a Service to the clear.crud.http.v1 routes.
type Handler struct {
	service *crud.Service
	options Options
}

// New validates the host dependencies once at startup.
func New(service *crud.Service, options Options) (*Handler, error) {
	if service == nil || options.Translator == nil {
		return nil, errors.New("httpadapter: service and translator are required")
	}
	if options.MaxBodyBytes <= 0 || options.MaxBodyBytes > maxJSONBytes {
		options.MaxBodyBytes = maxJSONBytes
	}
	return &Handler{service: service, options: options}, nil
}

func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "private, no-store")
	request = request.WithContext(crud.WithCorrelationID(request.Context(), handler.correlation(request)))
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if len(parts) < 5 || strings.Join(parts[:3], "/") != "api/v1/crud" {
		handler.fail(writer, request, nil, crud.ErrorNotFound)
		return
	}
	key := crud.ResourceKey(parts[3])
	definition, err := handler.service.Definition(request.Context(), key)
	if err != nil {
		handler.writeError(writer, request, err)
		return
	}
	if len(parts) == 5 && parts[4] == "definition" && request.Method == http.MethodGet {
		handler.ok(writer, request, http.StatusOK, definition, nil)
		return
	}
	if len(parts) == 5 && parts[4] == "records" {
		handler.records(writer, request, key, definition)
		return
	}
	if len(parts) == 6 && parts[4] == "records" {
		handler.record(writer, request, key, definition, crud.RecordID(parts[5]))
		return
	}
	if len(parts) == 6 && parts[4] == "lookups" && request.Method == http.MethodGet {
		query, err := lookupQuery(request, definition, crud.FieldKey(parts[5]))
		if err != nil {
			handler.writeError(writer, request, err)
			return
		}
		page, err := handler.service.Lookup(request.Context(), key, crud.FieldKey(parts[5]), query)
		if err != nil {
			handler.writeError(writer, request, err)
			return
		}
		handler.ok(writer, request, http.StatusOK, page.Options, nil)
		return
	}
	handler.fail(writer, request, nil, crud.ErrorNotFound)
}

func (handler *Handler) records(w http.ResponseWriter, r *http.Request, key crud.ResourceKey, definition crud.PublicDefinition) {
	switch r.Method {
	case http.MethodGet:
		query, err := listQuery(r, definition)
		if err != nil {
			handler.writeError(w, r, err)
			return
		}
		page, err := handler.service.List(r.Context(), key, query)
		if err != nil {
			handler.writeError(w, r, err)
			return
		}
		handler.ok(w, r, http.StatusOK, page.Records, &page)
	case http.MethodPost:
		mutation, err := handler.mutation(r, definition, false)
		if err != nil {
			handler.writeError(w, r, err)
			return
		}
		record, err := handler.service.Create(r.Context(), key, mutation)
		if err != nil {
			handler.writeError(w, r, err)
			return
		}
		handler.ok(w, r, http.StatusCreated, record, nil)
	default:
		handler.method(w, r)
	}
}

func listQuery(r *http.Request, definition crud.PublicDefinition) (crud.Query, error) {
	pageNumber, size, err := pageValues(r)
	if err != nil {
		return crud.Query{}, err
	}
	includeArchived, err := includeArchivedValue(r)
	if err != nil {
		return crud.Query{}, err
	}
	filters, err := filterValues(r, definition)
	if err != nil {
		return crud.Query{}, err
	}
	for key, values := range r.URL.Query() {
		if key == "q" || key == "page" || key == "size" || key == "include_archived" || strings.HasPrefix(key, "filter.") {
			if !strings.HasPrefix(key, "filter.") && len(values) != 1 {
				return crud.Query{}, invalid()
			}
			continue
		}
		return crud.Query{}, invalid()
	}
	return crud.Query{Search: r.URL.Query().Get("q"), Filters: filters, IncludeArchived: includeArchived, Page: crud.PageRequest{Number: pageNumber, Size: size}}, nil
}

func filterValues(r *http.Request, definition crud.PublicDefinition) ([]crud.Filter, error) {
	filters := make([]crud.Filter, 0)
	fields := make(map[crud.FieldKey]crud.Field, len(definition.Fields))
	for _, field := range definition.Fields {
		fields[field.Key] = field
	}
	for key, values := range r.URL.Query() {
		if !strings.HasPrefix(key, "filter.") {
			continue
		}
		parts := strings.Split(strings.TrimPrefix(key, "filter."), ".")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" || len(values) == 0 {
			return nil, invalid()
		}
		fieldKey := crud.FieldKey(parts[0])
		field, ok := fields[fieldKey]
		if !ok || field.Sensitive {
			return nil, invalid()
		}
		operator := crud.FilterOperator(parts[1])
		if operator == crud.FilterIn {
			if len(values) > 100 {
				return nil, invalid()
			}
			parsed := make([]crud.Value, 0, len(values))
			for _, raw := range values {
				value, err := parseFilterValue(field, operator, raw)
				if err != nil {
					return nil, err
				}
				parsed = append(parsed, value)
			}
			filters = append(filters, crud.Filter{Field: fieldKey, Operator: operator, Values: parsed})
			continue
		}
		if len(values) != 1 {
			return nil, invalid()
		}
		value, err := parseFilterValue(field, operator, values[0])
		if err != nil {
			return nil, err
		}
		filters = append(filters, crud.Filter{Field: fieldKey, Operator: operator, Value: value})
	}
	return filters, nil
}

func parseFilterValue(field crud.Field, operator crud.FilterOperator, raw string) (crud.Value, error) {
	if operator == crud.FilterIsNull {
		if raw != "" {
			return nil, invalid()
		}
		return nil, nil
	}
	switch field.Type {
	case crud.FieldBoolean:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, invalid()
		}
		return value, nil
	case crud.FieldInteger:
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, invalid()
		}
		return value, nil
	default:
		return raw, nil
	}
}

func includeArchivedValue(request *http.Request) (bool, error) {
	value := request.URL.Query().Get("include_archived")
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, invalid()
	}
	return parsed, nil
}

func (handler *Handler) record(w http.ResponseWriter, r *http.Request, key crud.ResourceKey, definition crud.PublicDefinition, id crud.RecordID) {
	switch r.Method {
	case http.MethodGet:
		record, err := handler.service.Get(r.Context(), key, id)
		if err != nil {
			handler.writeError(w, r, err)
			return
		}
		handler.ok(w, r, http.StatusOK, record, nil)
	case http.MethodPut:
		mutation, version, err := handler.versionedMutation(r, definition)
		if err != nil {
			handler.writeError(w, r, err)
			return
		}
		record, err := handler.service.Update(r.Context(), key, id, version, mutation)
		if err != nil {
			handler.writeError(w, r, err)
			return
		}
		handler.ok(w, r, http.StatusOK, record, nil)
	case http.MethodDelete:
		mutation, version, err := handler.versionedMutation(r, definition)
		_ = mutation
		if err != nil {
			handler.writeError(w, r, err)
			return
		}
		if err := handler.service.Delete(r.Context(), key, id, version); err != nil {
			handler.writeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		handler.method(w, r)
	}
}

type mutationRequest struct {
	Version crud.Version         `json:"version"`
	Fields  crud.Fields          `json:"fields"`
	Details crud.DetailMutations `json:"details"`
}

func (handler *Handler) mutation(r *http.Request, definition crud.PublicDefinition, versioned bool) (crud.Mutation, error) {
	request, err := handler.decode(r)
	if err != nil {
		return crud.Mutation{}, err
	}
	if err := normalizeFields(definition, request.Fields); err != nil {
		return crud.Mutation{}, err
	}
	if err := normalizeDetails(definition, request.Details); err != nil {
		return crud.Mutation{}, err
	}
	if versioned && request.Version == 0 {
		return crud.Mutation{}, invalid()
	}
	return crud.Mutation{Fields: request.Fields, Details: request.Details}, nil
}
func (handler *Handler) versionedMutation(r *http.Request, definition crud.PublicDefinition) (crud.Mutation, crud.Version, error) {
	request, err := handler.decode(r)
	if err != nil {
		return crud.Mutation{}, 0, err
	}
	if request.Version == 0 {
		return crud.Mutation{}, 0, invalid()
	}
	if err := normalizeFields(definition, request.Fields); err != nil {
		return crud.Mutation{}, 0, err
	}
	if err := normalizeDetails(definition, request.Details); err != nil {
		return crud.Mutation{}, 0, err
	}
	return crud.Mutation{Fields: request.Fields, Details: request.Details}, request.Version, nil
}
func (handler *Handler) decode(r *http.Request) (mutationRequest, error) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return mutationRequest{}, invalid()
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, handler.options.MaxBodyBytes+1))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var value mutationRequest
	if err := decoder.Decode(&value); err != nil {
		return mutationRequest{}, invalid()
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return mutationRequest{}, invalid()
	}
	return value, nil
}

func pageValues(r *http.Request) (uint64, uint16, error) {
	page, size := uint64(0), uint64(0)
	var err error
	if raw := r.URL.Query().Get("page"); raw != "" {
		page, err = strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return 0, 0, invalid()
		}
	}
	if raw := r.URL.Query().Get("size"); raw != "" {
		size, err = strconv.ParseUint(raw, 10, 16)
		if err != nil {
			return 0, 0, invalid()
		}
	}
	return page, uint16(size), nil
}

func lookupSize(r *http.Request) (uint16, error) {
	raw := r.URL.Query().Get("size")
	if raw == "" {
		return 0, nil
	}
	size, err := strconv.ParseUint(raw, 10, 16)
	if err != nil {
		return 0, invalid()
	}
	return uint16(size), nil
}

func lookupQuery(r *http.Request, definition crud.PublicDefinition, fieldKey crud.FieldKey) (crud.LookupQuery, error) {
	var lookup *crud.LookupDefinition
	for _, field := range definition.Fields {
		if field.Key == fieldKey {
			lookup = field.Lookup
			break
		}
	}
	if lookup == nil {
		return crud.LookupQuery{}, invalid()
	}
	values := r.URL.Query()
	for key, entries := range values {
		if key == "q" || key == "cursor" || key == "size" {
			if len(entries) != 1 {
				return crud.LookupQuery{}, invalid()
			}
			continue
		}
		if !strings.HasPrefix(key, "depends.") || len(entries) != 1 {
			return crud.LookupQuery{}, invalid()
		}
		dependency := crud.FieldKey(strings.TrimPrefix(key, "depends."))
		allowed := false
		for _, declared := range lookup.Dependencies {
			if declared == dependency {
				allowed = true
				break
			}
		}
		if !allowed || dependency == "" {
			return crud.LookupQuery{}, invalid()
		}
	}
	size, err := lookupSize(r)
	if err != nil {
		return crud.LookupQuery{}, err
	}
	query := crud.LookupQuery{Search: values.Get("q"), Cursor: values.Get("cursor"), Size: size, Dependencies: crud.Fields{}}
	for _, dependency := range lookup.Dependencies {
		raw, ok := values["depends."+string(dependency)]
		if !ok || len(raw) == 0 || raw[0] == "" {
			continue
		}
		value, ok := lookupScalar(raw[0])
		if !ok {
			return crud.LookupQuery{}, invalid()
		}
		query.Dependencies[dependency] = value
	}
	if len(query.Dependencies) == 0 {
		query.Dependencies = nil
	}
	return query, nil
}

func lookupScalar(value string) (crud.Value, bool) {
	switch value {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	if integer, err := strconv.ParseInt(value, 10, 64); err == nil {
		return integer, true
	}
	return value, true
}
func normalizeFields(definition crud.PublicDefinition, fields crud.Fields) error {
	known := map[crud.FieldKey]crud.Field{}
	for _, field := range definition.Fields {
		known[field.Key] = field
	}
	for key, value := range fields {
		field, ok := known[key]
		if !ok {
			return invalid()
		}
		if number, ok := value.(json.Number); ok {
			if field.Type != crud.FieldInteger && field.Type != crud.FieldLookup {
				return invalid()
			}
			integer, err := number.Int64()
			if err != nil {
				return invalid()
			}
			fields[key] = integer
		}
	}
	return nil
}

func normalizeDetails(definition crud.PublicDefinition, details crud.DetailMutations) error {
	known := make(map[crud.DetailKey]crud.PublicDetailDefinition, len(definition.Details))
	for _, detail := range definition.Details {
		known[detail.Key] = detail
	}
	for key, mutations := range details {
		detail, ok := known[key]
		if !ok || len(mutations) > int(detail.Maximum) {
			return invalid()
		}
		child := crud.PublicDefinition{Fields: detail.Fields}
		for _, mutation := range mutations {
			if err := normalizeFields(child, mutation.Fields); err != nil {
				return err
			}
		}
	}
	return nil
}

func rejectUnknownQuery(r *http.Request, allowed map[string]bool) error {
	for key, values := range r.URL.Query() {
		if !allowed[key] || len(values) != 1 {
			return invalid()
		}
	}
	return nil
}
func invalid() *crud.Error {
	return &crud.Error{Code: crud.ErrorInvalidRequest, Message: "crud.error.invalid_request"}
}
func (handler *Handler) method(w http.ResponseWriter, r *http.Request) {
	handler.fail(w, r, nil, crud.ErrorInvalidRequest)
}
func (handler *Handler) ok(w http.ResponseWriter, r *http.Request, status int, data any, page *crud.Page) {
	meta := map[string]any{"correlationId": handler.correlation(r)}
	if page != nil {
		meta["page"] = page.Page
		meta["size"] = page.Size
		if page.Total != nil {
			meta["total"] = *page.Total
		}
		if page.NextCursor != "" {
			meta["nextCursor"] = page.NextCursor
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "meta": meta})
}
func (handler *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var public *crud.Error
	if !errors.As(err, &public) {
		handler.fail(w, r, nil, crud.ErrorTemporarilyUnavailable)
		return
	}
	handler.fail(w, r, public, public.Code)
}
func (handler *Handler) fail(w http.ResponseWriter, r *http.Request, public *crud.Error, code crud.ErrorCode) {
	if public == nil {
		public = &crud.Error{Code: code, Message: crud.MessageCode("crud.error." + string(code))}
	}
	fields := map[string]string{}
	for key, message := range public.Fields {
		fields[string(key)] = handler.options.Translator.Message(r.Context(), message, nil)
	}
	body := map[string]any{"code": code, "message": handler.options.Translator.Message(r.Context(), public.Message, nil), "correlationId": handler.correlation(r)}
	if len(fields) > 0 {
		body["fields"] = fields
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status(code))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": body})
}
func (handler *Handler) correlation(r *http.Request) string {
	if id := crud.CorrelationID(r.Context()); id != "" {
		return id
	}
	if handler.options.CorrelationID == nil {
		return ""
	}
	return handler.options.CorrelationID(r)
}
func status(code crud.ErrorCode) int {
	switch code {
	case crud.ErrorUnauthenticated:
		return 401
	case crud.ErrorForbidden:
		return 403
	case crud.ErrorNotFound:
		return 404
	case crud.ErrorValidationFailed:
		return 422
	case crud.ErrorConflict, crud.ErrorDeleteRestricted:
		return 409
	case crud.ErrorRateLimited:
		return 429
	case crud.ErrorTemporarilyUnavailable:
		return 503
	default:
		return 400
	}
}
