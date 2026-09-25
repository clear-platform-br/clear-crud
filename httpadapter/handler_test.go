package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	crud "github.com/clear-platform-br/clear-crud"
)

func TestHandlerDefinitionAndErrorsAreSafe(t *testing.T) {
	h := testHandler(t, &source{})
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/crud/contacts/definition", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"data"`) {
		t.Fatalf("definition: %d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/crud/missing/definition", nil))
	if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "driver") {
		t.Fatalf("error: %d %s", response.Code, response.Body.String())
	}
}

func TestHandlerRejectsBadRequestBeforeMutation(t *testing.T) {
	source := &source{}
	h := testHandler(t, source)
	for _, request := range []*http.Request{
		httptest.NewRequest(http.MethodPost, "/api/v1/crud/contacts/records", strings.NewReader(`{"fields":{"name":"Ana","extra":"x"}}`)),
		httptest.NewRequest(http.MethodPost, "/api/v1/crud/contacts/records", strings.NewReader(`{"fields":{"age":1.5}}`)),
		httptest.NewRequest(http.MethodGet, "/api/v1/crud/contacts/records?tenant_id=x", nil),
	} {
		response := httptest.NewRecorder()
		if request.Method == http.MethodPost {
			request.Header.Set("Content-Type", "application/json")
		}
		h.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
	}
	if source.creates != 0 {
		t.Fatal("invalid request reached persistence")
	}
}

func TestHandlerMutationsAndTechnicalFailure(t *testing.T) {
	source := &source{}
	h := testHandler(t, source)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/crud/contacts/records", strings.NewReader(`{"fields":{"name":"Ana","age":3}}`))
	request.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || source.creates != 1 {
		t.Fatalf("create %d %s", response.Code, response.Body.String())
	}
	source.err = errors.New("sqlite driver password")
	response = httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/crud/contacts/records", nil))
	if response.Code != http.StatusServiceUnavailable || strings.Contains(response.Body.String(), "sqlite") {
		t.Fatalf("leak %d %s", response.Code, response.Body.String())
	}
}

func TestHandlerRouteHelpers(t *testing.T) {
	h := testHandler(t, &source{})
	cases := []struct {
		method, path, body string
		want               int
	}{{http.MethodGet, "/api/v1/crud/contacts/records/1", "", 200}, {http.MethodPut, "/api/v1/crud/contacts/records/1", `{"version":1,"fields":{"name":"A","age":2}}`, 200}, {http.MethodDelete, "/api/v1/crud/contacts/records/1", `{"version":1,"fields":{}}`, 204}, {http.MethodGet, "/api/v1/crud/contacts/lookups/name?size=bad", "", 400}, {http.MethodPatch, "/api/v1/crud/contacts/records", "", 400}, {http.MethodGet, "/wrong", "", 404}}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		if c.body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Fatalf("%s %s = %d %s", c.method, c.path, w.Code, w.Body.String())
		}
	}
	for _, code := range []crud.ErrorCode{crud.ErrorUnauthenticated, crud.ErrorForbidden, crud.ErrorNotFound, crud.ErrorValidationFailed, crud.ErrorConflict, crud.ErrorDeleteRestricted, crud.ErrorRateLimited, crud.ErrorTemporarilyUnavailable, crud.ErrorInvalidRequest} {
		if status(code) == 0 {
			t.Fatal(code)
		}
	}
	if _, _, err := pageValues(httptest.NewRequest(http.MethodGet, "/?page=x", nil)); err == nil {
		t.Fatal("bad page accepted")
	}
	if _, err := lookupSize(httptest.NewRequest(http.MethodGet, "/?size=x", nil)); err == nil {
		t.Fatal("bad lookup size accepted")
	}
}

func TestTransportHelpers(t *testing.T) {
	if _, err := New(nil, Options{Translator: translator{}}); err == nil {
		t.Fatal("nil service")
	}
	definition := crud.PublicDefinition{Fields: []crud.Field{{Key: "name", Type: crud.FieldString}, {Key: "age", Type: crud.FieldInteger}}}
	h := testHandler(t, &source{})
	for _, c := range []struct{ content, body string }{{"text/plain", `{}`}, {"application/json", `{"unknown":1}`}, {"application/json", `{} {}`}} {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(c.body))
		r.Header.Set("Content-Type", c.content)
		if _, err := h.mutation(r, definition, false); err == nil {
			t.Fatal(c)
		}
	}
	if _, err := h.mutation(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"fields":{"age":2}}`)), definition, false); err == nil {
		t.Fatal("content type")
	}
	if _, _, err := h.versionedMutation(httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"version":0,"fields":{}`)), definition); err == nil {
		t.Fatal("version")
	}
	if page, size, err := pageValues(httptest.NewRequest(http.MethodGet, "/?page=2&size=25", nil)); err != nil || page != 2 || size != 25 {
		t.Fatal(page, size, err)
	}
	if size, err := lookupSize(httptest.NewRequest(http.MethodGet, "/?size=25", nil)); err != nil || size != 25 {
		t.Fatal(size, err)
	}
	if err := normalizeFields(definition, crud.Fields{"name": json.Number("1")}); err == nil {
		t.Fatal("string number")
	}
	w := httptest.NewRecorder()
	h.ok(w, httptest.NewRequest(http.MethodGet, "/", nil), 200, map[string]string{"a": "b"}, &crud.Page{Page: 1, Size: 1})
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	w = httptest.NewRecorder()
	h.fail(w, httptest.NewRequest(http.MethodGet, "/", nil), &crud.Error{Code: crud.ErrorValidationFailed, Message: "bad", Fields: map[crud.FieldKey]crud.MessageCode{"name": "bad"}}, crud.ErrorValidationFailed)
	if w.Code != 422 {
		t.Fatal(w.Code)
	}
	w = httptest.NewRecorder()
	h.writeError(w, httptest.NewRequest(http.MethodGet, "/", nil), errors.New("internal"))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
}

func TestFullHTTPMatrix(t *testing.T) {
	h := testHandler(t, &source{})
	cases := []struct {
		m, p, b, ct string
		want        int
	}{
		{"GET", "/api/v1/crud/contacts/records?page=2&size=25", "", "", 200},
		{"GET", "/api/v1/crud/contacts/records?page=0&cursor=x", "", "", 400},
		{"POST", "/api/v1/crud/contacts/records", `{"fields":{}}`, "application/json", 201},
		{"PUT", "/api/v1/crud/contacts/records/1", `{"version":0,"fields":{}}`, "application/json", 400},
		{"DELETE", "/api/v1/crud/contacts/records/1", `{}`, "application/json", 400},
		{"GET", "/api/v1/crud/contacts/lookups/category?q=a&cursor=x&size=25", "", "", 200},
		{"GET", "/api/v1/crud/contacts/lookups/category?depends.x=y", "", "", 400},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.m, c.p, strings.NewReader(c.b))
		if c.ct != "" {
			r.Header.Set("Content-Type", c.ct)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Fatalf("%s %s got %d %s", c.m, c.p, w.Code, w.Body.String())
		}
	}
	if h.correlation(httptest.NewRequest("GET", "/", nil)) != "c" {
		t.Fatal("correlation")
	}
	s := &source{err: errors.New("driver")}
	h = testHandler(t, s)
	for _, path := range []string{"/api/v1/crud/contacts/records/1", "/api/v1/crud/contacts/lookups/category"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 503 {
			t.Fatal(path, w.Code)
		}
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/api/v1/crud/contacts/records/1", strings.NewReader(`{"version":1,"fields":{}}`))
		r.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
		if w.Code != 503 {
			t.Fatal(method, w.Code)
		}
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/crud/contacts/records", strings.NewReader(`{"fields":{}}`))
	r.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(w, r)
	if w.Code != 503 {
		t.Fatal("post", w.Code)
	}
}

type source struct {
	creates int
	err     error
}

func (s *source) Capabilities(context.Context) crud.Capabilities {
	return crud.Capabilities{crud.CapabilityOffsetPage: {}, crud.CapabilityTotalCount: {}, crud.CapabilityAtomicVersion: {}, crud.CapabilityUnitOfWork: {}, crud.CapabilityArchive: {}, crud.CapabilityLookup: {}}
}
func (s *source) List(context.Context, crud.Scope, crud.Query) (crud.Page, error) {
	if s.err != nil {
		return crud.Page{}, s.err
	}
	n := uint64(0)
	return crud.Page{Page: 1, Size: 25, Total: &n}, nil
}
func (s *source) Get(_ context.Context, _ crud.Scope, id crud.RecordID) (crud.Record, error) {
	if s.err != nil {
		return crud.Record{}, s.err
	}
	return crud.Record{ID: id, Version: 1, Fields: crud.Fields{"name": "A", "age": int64(2)}}, nil
}
func (s *source) Create(_ context.Context, _ crud.Scope, m crud.Mutation) (crud.Record, error) {
	if s.err != nil {
		return crud.Record{}, s.err
	}
	s.creates++
	return crud.Record{ID: "1", Version: 1, Fields: m.Fields}, nil
}
func (s *source) Update(_ context.Context, _ crud.Scope, id crud.RecordID, _ crud.Version, m crud.Mutation) (crud.Record, error) {
	if s.err != nil {
		return crud.Record{}, s.err
	}
	return crud.Record{ID: id, Version: 2, Fields: m.Fields}, nil
}
func (s *source) Delete(context.Context, crud.Scope, crud.RecordID, crud.Version, crud.DeleteMode) error {
	if s.err != nil {
		return s.err
	}
	return nil
}
func (s *source) Lookup(context.Context, crud.Scope, crud.LookupQuery) (crud.LookupPage, error) {
	if s.err != nil {
		return crud.LookupPage{}, s.err
	}
	return crud.LookupPage{Options: []crud.LookupOption{{Value: "1", Label: "Category"}}}, nil
}

type uow struct{}

func (uow) Within(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type principal struct{}

func (principal) Principal(context.Context) (crud.Principal, error) {
	return crud.Principal{ID: "u"}, nil
}

type scope struct{}

func (scope) Scope(context.Context, crud.ResourceKey) (crud.Scope, error) {
	return crud.Scope{"tenant_id": "t"}, nil
}

type auth struct{}

func (auth) Authorize(context.Context, crud.Principal, crud.ResourceKey, crud.Action, *crud.Record) error {
	return nil
}

type audit struct{}

func (audit) Append(context.Context, crud.AuditEvent) error { return nil }

type translator struct{}

func (translator) Message(_ context.Context, code crud.MessageCode, _ map[string]any) string {
	return string(code)
}

type clock struct{}

func (clock) Now() time.Time { return time.Unix(0, 0) }
func testHandler(t *testing.T, source *source) *Handler {
	t.Helper()
	definition := crud.Definition{Contract: crud.ContractDefinitionV1, Key: "contacts", Labels: crud.Labels{Title: "contacts", Singular: "contact"}, Scope: crud.ScopeRequirements{Keys: []string{"tenant_id"}}, Permissions: crud.Permissions{Read: "r", Create: "c", Update: "u", Delete: "d"}, Fields: []crud.Field{{Key: "name", Label: "name", Type: crud.FieldString, Visible: true}, {Key: "age", Label: "age", Type: crud.FieldInteger, Visible: true}, {Key: "category", Label: "category", Type: crud.FieldLookup, Visible: true, Lookup: &crud.LookupDefinition{Resource: "categories", ValueField: "id", LabelField: "name", PageSize: 25}}}, List: crud.ListDefinition{Columns: []crud.FieldKey{"name"}, Searchable: []crud.FieldKey{"name"}, Sortable: []crud.FieldKey{"name"}, DefaultSort: []crud.Sort{{Field: "name", Direction: crud.SortAscending}}, Pagination: crud.PaginationDefinition{Mode: crud.PageModeOffset, DefaultSize: 25, AllowedSizes: []uint16{25, 50, 100}, Total: true}}, Presentation: crud.Presentation{Collection: crud.CollectionAuto, Density: crud.DensityCompact}, Source: source, UOW: uow{}, Delete: crud.DeletePolicy{Mode: crud.DeleteModeArchive}, Concurrency: crud.ConcurrencyPolicy{Mode: crud.ConcurrencyVersion}}
	registry := crud.NewRegistry()
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatal(err)
	}
	registry.Seal()
	service, err := crud.NewService(crud.Dependencies{Registry: registry, Principal: principal{}, Scope: scope{}, Authorizer: auth{}, Audit: audit{}, Translator: translator{}, Clock: clock{}})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(service, Options{Translator: translator{}, CorrelationID: func(*http.Request) string { return "c" }})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}
