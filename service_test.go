package crud

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewServiceRequiresEveryHostPort(t *testing.T) {
	t.Parallel()

	dependencies := serviceDependencies(t, &recordingSource{})
	tests := map[string]func(*Dependencies){
		"registry":   func(dependencies *Dependencies) { dependencies.Registry = nil },
		"principal":  func(dependencies *Dependencies) { dependencies.Principal = nil },
		"scope":      func(dependencies *Dependencies) { dependencies.Scope = nil },
		"authorizer": func(dependencies *Dependencies) { dependencies.Authorizer = nil },
		"audit":      func(dependencies *Dependencies) { dependencies.Audit = nil },
		"translator": func(dependencies *Dependencies) { dependencies.Translator = nil },
		"clock":      func(dependencies *Dependencies) { dependencies.Clock = nil },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			copy := dependencies
			change(&copy)
			if service, err := NewService(copy); err == nil || service != nil {
				t.Fatalf("NewService() = %v, %v; want configuration error", service, err)
			}
		})
	}
}

func TestServiceDefinitionExposesOnlyRendererMetadataAndAllowedActions(t *testing.T) {
	t.Parallel()

	source := &recordingSource{}
	authorizer := &recordingAuthorizer{deny: map[Action]error{ActionDelete: errors.New("denied")}}
	service := newServiceForTest(t, source, authorizer)

	definition, err := service.Definition(context.Background(), "contact_categories")
	if err != nil {
		t.Fatalf("Definition() error = %v", err)
	}
	if definition.Key != "contact_categories" || len(definition.Fields) != 2 {
		t.Fatalf("Definition() = %#v", definition)
	}
	if !hasAction(definition.Actions, ActionCreate) || !hasAction(definition.Actions, ActionUpdate) || hasAction(definition.Actions, ActionDelete) {
		t.Fatalf("actions = %v", definition.Actions)
	}
	definition.Fields[0].Label = "changed"
	again, err := service.Definition(context.Background(), "contact_categories")
	if err != nil || again.Fields[0].Label != "crud.contact_categories.name" {
		t.Fatalf("Definition() must return defensive metadata: %#v, %v", again, err)
	}
	_, err = service.Definition(context.Background(), "missing")
	assertCode(t, err, ErrorNotFound)
}

func TestServiceListNormalizesScopeQueryAndRecordFields(t *testing.T) {
	t.Parallel()

	source := &recordingSource{
		capabilities: serviceCapabilities(),
		listPage: Page{Records: []Record{{ID: "1", Fields: Fields{
			"name": "Ana", "active": true, "driver_only": "must not cross the boundary",
		}}}},
	}
	service := newServiceForTest(t, source, &recordingAuthorizer{})
	page, err := service.List(context.Background(), "contact_categories", Query{
		Search:  "  Ana  ",
		Filters: []Filter{{Field: "name", Operator: FilterContains, Value: "an"}},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if source.listScope["tenant_id"] != "tenant-a" {
		t.Fatalf("List() scope = %#v", source.listScope)
	}
	if source.listQuery.Search != "Ana" || source.listQuery.Page.Number != 1 || source.listQuery.Page.Size != 25 {
		t.Fatalf("normalized query = %#v", source.listQuery)
	}
	if got := source.listQuery.Sort; len(got) != 2 || got[0].Field != "name" || got[1].Field != "id" {
		t.Fatalf("normalized sort = %#v", got)
	}
	if _, leaked := page.Records[0].Fields["driver_only"]; leaked {
		t.Fatalf("List() leaked a non-declared field: %#v", page)
	}
}

func TestServiceReadFailuresNeverCallDataSource(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		principalErr error
		scope        Scope
		scopeErr     error
		authErr      error
		want         ErrorCode
	}{
		"unknown resource":       {want: ErrorNotFound},
		"missing identity":       {principalErr: errors.New("oauth failed"), want: ErrorUnauthenticated},
		"scope provider failure": {scopeErr: errors.New("tenant lookup failed"), want: ErrorForbidden},
		"scope key absent":       {scope: Scope{}, want: ErrorForbidden},
		"authorization denied":   {authErr: errors.New("policy denied"), want: ErrorForbidden},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			source := &recordingSource{capabilities: serviceCapabilities()}
			dependencies := serviceDependencies(t, source)
			dependencies.Principal = principalStub{err: test.principalErr}
			scope := test.scope
			if scope == nil && name != "scope key absent" {
				scope = Scope{"tenant_id": "tenant-a"}
			}
			dependencies.Scope = scopeStub{scope: scope, err: test.scopeErr}
			dependencies.Authorizer = &recordingAuthorizer{deny: map[Action]error{ActionRead: test.authErr}}
			service, err := NewService(dependencies)
			if err != nil {
				t.Fatalf("NewService() error = %v", err)
			}
			key := ResourceKey("contact_categories")
			if name == "unknown resource" {
				key = "missing"
			}
			_, err = service.List(context.Background(), key, Query{})
			assertCode(t, err, test.want)
			if source.listCalls != 0 {
				t.Fatalf("List() called source %d times", source.listCalls)
			}
		})
	}
}

func TestServiceGetReauthorizesLoadedRecordAndSanitizesOutput(t *testing.T) {
	t.Parallel()

	source := &recordingSource{getRecord: Record{ID: "1", Fields: Fields{"name": "Ana", "internal": "x"}}}
	authorizer := &recordingAuthorizer{denyCurrent: errors.New("record policy denied")}
	service := newServiceForTest(t, source, authorizer)
	_, err := service.Get(context.Background(), "contact_categories", "1")
	assertCode(t, err, ErrorForbidden)
	if source.getCalls != 1 || authorizer.currentCalls != 1 {
		t.Fatalf("Get() source calls = %d, record authorizations = %d", source.getCalls, authorizer.currentCalls)
	}

	_, err = service.Get(context.Background(), "contact_categories", "")
	assertCode(t, err, ErrorInvalidRequest)
}

func TestServiceGetAndLookupSuccessAndFailures(t *testing.T) {
	t.Parallel()

	source := &recordingSource{getRecord: Record{ID: "1", Version: 2, Fields: Fields{"name": "Ana", "internal": "x"}}}
	service := newServiceForTest(t, source, &recordingAuthorizer{})
	record, err := service.Get(context.Background(), "contact_categories", "1")
	if err != nil || record.Fields["name"] != "Ana" {
		t.Fatalf("Get() = %#v, %v", record, err)
	}
	if _, leaked := record.Fields["internal"]; leaked {
		t.Fatal("Get() leaked an unknown field")
	}
	source.getErr = errors.New("driver failure")
	_, err = service.Get(context.Background(), "contact_categories", "1")
	assertCode(t, err, ErrorTemporarilyUnavailable)

	lookupSource := &recordingSource{capabilities: capabilitiesWithLookup(), lookupErr: errors.New("driver failure")}
	lookupService := newServiceWithLookup(t, lookupSource, &recordingAuthorizer{})
	_, err = lookupService.Lookup(context.Background(), "contact_categories", "missing", LookupQuery{})
	assertCode(t, err, ErrorInvalidRequest)
	_, err = lookupService.Lookup(context.Background(), "contact_categories", "category", LookupQuery{})
	assertCode(t, err, ErrorTemporarilyUnavailable)
}

func TestServiceLookupRestrictsDependenciesAndNormalizesPage(t *testing.T) {
	t.Parallel()

	source := &recordingSource{capabilities: capabilitiesWithLookup(), lookupPage: LookupPage{Options: []LookupOption{{Value: "1", Label: "Category"}}}}
	service := newServiceWithLookup(t, source, &recordingAuthorizer{})
	page, err := service.Lookup(context.Background(), "contact_categories", "category", LookupQuery{
		Search:       "  Cat ",
		Dependencies: Fields{"active": true},
	})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if len(page.Options) != 1 || source.lookupQuery.Search != "Cat" || source.lookupQuery.Size != 25 || source.lookupQuery.Dependencies["active"] != true {
		t.Fatalf("Lookup() = %#v, source query = %#v", page, source.lookupQuery)
	}
	_, err = service.Lookup(context.Background(), "contact_categories", "category", LookupQuery{Dependencies: Fields{"name": "not allowed"}})
	assertCode(t, err, ErrorInvalidRequest)
	if source.lookupCalls != 1 {
		t.Fatalf("invalid lookup must not call source: %d", source.lookupCalls)
	}
}

func TestReadValidationRejectsUntrustedQueryShapes(t *testing.T) {
	t.Parallel()

	definition := validDefinition("contact_categories")
	definition.Source = &recordingSource{capabilities: serviceCapabilities()}
	tooManyFilters := make([]Filter, maxQueryFilters+1)
	for index := range tooManyFilters {
		tooManyFilters[index] = Filter{Field: "name", Operator: FilterEqual, Value: "x"}
	}
	tests := []Query{
		{Search: string(make([]rune, maxQuerySearchRunes+1))},
		{Filters: tooManyFilters},
		{Filters: []Filter{{Field: "missing", Operator: FilterEqual, Value: "x"}}},
		{Filters: []Filter{{Field: "name", Operator: FilterIsNull, Value: "not nil"}}},
		{Filters: []Filter{{Field: "active", Operator: FilterContains, Value: "yes"}}},
		{Sort: []Sort{{Field: "unknown", Direction: SortAscending}}},
		{Sort: []Sort{{Field: "name", Direction: "injected"}}},
		{Page: PageRequest{Size: 99}},
		{Page: PageRequest{Mode: PageModeCursor}},
		{Page: PageRequest{Cursor: "not allowed for offset"}},
		{Page: PageRequest{Cursor: string(make([]byte, maxCursorBytes+1))}},
	}
	for _, query := range tests {
		if _, err := normalizeQuery(context.Background(), definition, query); err == nil {
			t.Fatalf("normalizeQuery(%#v) accepted an invalid request", query)
		}
	}

	definition.Fields[0].Sensitive = true
	if _, err := normalizeQuery(context.Background(), definition, Query{Filters: []Filter{{Field: "name", Operator: FilterEqual, Value: "Ana"}}}); err == nil {
		t.Fatal("normalizeQuery() accepted a sensitive field filter")
	}
}

func TestInjectionShapedIdentifiersNeverReachDataSource(t *testing.T) {
	t.Parallel()

	source := &recordingSource{capabilities: serviceCapabilities()}
	service := newServiceForTest(t, source, &recordingAuthorizer{})
	for _, query := range []Query{
		{Filters: []Filter{{Field: "name; DROP TABLE contacts", Operator: FilterEqual, Value: "x"}}},
		{Sort: []Sort{{Field: "name DESC; DROP TABLE contacts", Direction: SortAscending}}},
	} {
		_, err := service.List(context.Background(), "contact_categories", query)
		assertCode(t, err, ErrorInvalidRequest)
	}
	_, err := service.List(context.Background(), "contact_categories; DROP TABLE contacts", Query{})
	assertCode(t, err, ErrorNotFound)
	if source.listCalls != 0 {
		t.Fatalf("untrusted identifiers reached DataSource %d times", source.listCalls)
	}
}

func TestReadValidationHandlesCursorAndLookupBoundaries(t *testing.T) {
	t.Parallel()

	cursor := validDefinition("cursor_categories")
	cursor.Permissions = Permissions{Read: "crud.categories.read"}
	cursor.Delete.Mode = DeleteModeNone
	cursor.Concurrency.Mode = ConcurrencyNone
	cursor.UOW = nil
	cursor.List.Pagination = PaginationDefinition{Mode: PageModeCursor, DefaultSize: 25, AllowedSizes: []uint16{25, 50, 100}}
	cursor.Source = fakeSource{capabilities: Capabilities{CapabilityCursorPage: {}}}
	if page, err := normalizePage(cursor.List.Pagination, PageRequest{Cursor: "opaque"}); err != nil || page.Cursor != "opaque" {
		t.Fatalf("normalizePage() = %#v, %v", page, err)
	}
	if _, err := normalizePage(cursor.List.Pagination, PageRequest{Number: 1}); err == nil {
		t.Fatal("cursor page accepted offset number")
	}

	lookup := lookupField().Lookup
	for _, query := range []LookupQuery{
		{Size: 26},
		{Search: string(make([]rune, maxQuerySearchRunes+1))},
		{Cursor: string(make([]byte, maxCursorBytes+1))},
		{Dependencies: Fields{"active": []string{"not scalar"}}},
		{Dependencies: Fields{"active": true, "other": true}},
	} {
		if _, err := normalizeLookup(lookup, query); err == nil {
			t.Fatalf("normalizeLookup(%#v) accepted an invalid request", query)
		}
	}
}

func TestReadFailuresAreSanitized(t *testing.T) {
	t.Parallel()

	private := errors.New("driver syntax detail")
	source := &recordingSource{capabilities: serviceCapabilities(), listErr: private, getErr: private, lookupErr: private}
	service := newServiceForTest(t, source, &recordingAuthorizer{})
	_, err := service.List(context.Background(), "contact_categories", Query{})
	assertCode(t, err, ErrorTemporarilyUnavailable)
	if err.Error() != "crud: temporarily_unavailable" {
		t.Fatalf("List() leaked source details: %v", err)
	}
	_, err = service.Get(context.Background(), "contact_categories", "1")
	assertCode(t, err, ErrorTemporarilyUnavailable)

	lookupSource := &recordingSource{capabilities: capabilitiesWithLookup(), lookupErr: private}
	lookupService := newServiceWithLookup(t, lookupSource, &recordingAuthorizer{})
	_, err = lookupService.Lookup(context.Background(), "contact_categories", "category", LookupQuery{})
	assertCode(t, err, ErrorTemporarilyUnavailable)

	preserved := publicError(ErrorNotFound, "crud.error.not_found", nil)
	if got := unavailable(preserved); got != preserved {
		t.Fatal("unavailable() must preserve already-sanitized errors")
	}
}

func TestReadHelpersCoverSupportedActionsAndOperators(t *testing.T) {
	t.Parallel()

	permissions := Permissions{Create: "create", Read: "read", Update: "update", Delete: "delete", Help: "help"}
	for _, action := range []Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionHelp} {
		if !actionEnabled(permissions, action) {
			t.Fatalf("actionEnabled(%q) = false", action)
		}
	}
	if actionEnabled(Permissions{}, Action("unknown")) {
		t.Fatal("unknown action must be unavailable")
	}
	for _, test := range []struct {
		field    FieldType
		operator FilterOperator
	}{
		{FieldText, FilterPrefix}, {FieldInteger, FilterGreaterThan}, {FieldBoolean, FilterEqual}, {FieldBoolean, FilterLessThan},
	} {
		filterAllowed(test.field, test.operator)
	}
	page := LookupPage{Options: []LookupOption{{Value: "1", Label: "one"}}}
	clone := sanitizeLookupPage(page)
	clone.Options[0].Label = "changed"
	if page.Options[0].Label != "one" {
		t.Fatal("lookup options must be copied")
	}
}

func newServiceForTest(t *testing.T, source *recordingSource, authorizer *recordingAuthorizer) *Service {
	t.Helper()
	dependencies := serviceDependencies(t, source)
	dependencies.Authorizer = authorizer
	service, err := NewService(dependencies)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func newServiceWithLookup(t *testing.T, source *recordingSource, authorizer *recordingAuthorizer) *Service {
	t.Helper()
	registry := NewRegistry()
	definition := validDefinition("contact_categories")
	definition.Fields = append(definition.Fields, lookupField())
	definition.Source = source
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return newServiceForRegistry(t, registry, source, authorizer)
}

func serviceDependencies(t *testing.T, source *recordingSource) Dependencies {
	t.Helper()
	if source.capabilities == nil {
		source.capabilities = serviceCapabilities()
	}
	registry := NewRegistry()
	definition := validDefinition("contact_categories")
	definition.Source = source
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	return Dependencies{
		Registry: registry, Principal: principalStub{}, Scope: scopeStub{scope: Scope{"tenant_id": "tenant-a"}},
		Authorizer: &recordingAuthorizer{}, Audit: auditStub{}, Translator: translatorStub{}, Clock: clockStub{},
	}
}

func newServiceForRegistry(t *testing.T, registry *Registry, source *recordingSource, authorizer *recordingAuthorizer) *Service {
	t.Helper()
	service, err := NewService(Dependencies{
		Registry: registry, Principal: principalStub{}, Scope: scopeStub{scope: Scope{"tenant_id": "tenant-a"}},
		Authorizer: authorizer, Audit: auditStub{}, Translator: translatorStub{}, Clock: clockStub{},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func serviceCapabilities() Capabilities {
	capabilities := mutableCapabilities()
	capabilities[CapabilityContainsSearch] = struct{}{}
	return capabilities
}

func hasAction(actions []Action, want Action) bool {
	for _, action := range actions {
		if action == want {
			return true
		}
	}
	return false
}

func assertCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	var public *Error
	if !errors.As(err, &public) || public.Code != want {
		t.Fatalf("error = %v, want public code %q", err, want)
	}
}

type recordingSource struct {
	capabilities Capabilities
	listPage     Page
	listScope    Scope
	listQuery    Query
	listCalls    int
	getRecord    Record
	getCalls     int
	lookupPage   LookupPage
	lookupQuery  LookupQuery
	lookupCalls  int
	listErr      error
	getErr       error
	lookupErr    error
}

func (source *recordingSource) Capabilities(context.Context) Capabilities { return source.capabilities }
func (source *recordingSource) List(_ context.Context, scope Scope, query Query) (Page, error) {
	source.listCalls++
	source.listScope, source.listQuery = cloneScope(scope), query
	return source.listPage, source.listErr
}
func (source *recordingSource) Get(context.Context, Scope, RecordID) (Record, error) {
	source.getCalls++
	return source.getRecord, source.getErr
}
func (*recordingSource) Create(context.Context, Scope, Mutation) (Record, error) {
	return Record{}, nil
}
func (*recordingSource) Update(context.Context, Scope, RecordID, Version, Mutation) (Record, error) {
	return Record{}, nil
}
func (*recordingSource) Delete(context.Context, Scope, RecordID, Version, DeleteMode) error {
	return nil
}
func (source *recordingSource) Lookup(_ context.Context, _ Scope, query LookupQuery) (LookupPage, error) {
	source.lookupCalls++
	source.lookupQuery = query
	return source.lookupPage, source.lookupErr
}

type principalStub struct{ err error }

func (stub principalStub) Principal(context.Context) (Principal, error) {
	if stub.err != nil {
		return Principal{}, stub.err
	}
	return Principal{ID: "operator-1", Kind: "operator"}, nil
}

type scopeStub struct {
	scope Scope
	err   error
}

func (stub scopeStub) Scope(context.Context, ResourceKey) (Scope, error) { return stub.scope, stub.err }

type recordingAuthorizer struct {
	deny         map[Action]error
	denyCurrent  error
	currentCalls int
}

func (authorizer *recordingAuthorizer) Authorize(_ context.Context, _ Principal, _ ResourceKey, action Action, current *Record) error {
	if current != nil {
		authorizer.currentCalls++
		return authorizer.denyCurrent
	}
	return authorizer.deny[action]
}

type auditStub struct{}

func (auditStub) Append(context.Context, AuditEvent) error { return nil }

type translatorStub struct{}

func (translatorStub) Message(context.Context, MessageCode, map[string]any) string { return "" }

type clockStub struct{}

func (clockStub) Now() time.Time { return time.Time{} }
