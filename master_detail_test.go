package crud

import (
	"context"
	"errors"
	"testing"
)

func TestMasterDetailCreateIsAtomicAndHidesParentLink(t *testing.T) {
	parentSource := &detailSource{created: Record{ID: "contact-1", Version: 1, Fields: Fields{"name": "Ana", "active": true}}}
	childSource := &detailSource{
		created: Record{ID: "destination-1", Version: 1},
		list: Page{Records: []Record{{ID: "destination-1", Version: 1, Fields: Fields{
			"parent_id": "contact-1", "address": "ana@example.com", "active": true,
		}, Details: DetailRecords{"untrusted": {{ID: "leak"}}}}}, Total: total(1)},
	}
	uow, audit := &transactionUOW{}, &mutationAudit{}
	service := newMasterDetailService(t, parentSource, childSource, uow, audit, 1, 2, &detailAuthorizer{})

	record, err := service.Create(context.Background(), "contacts", Mutation{
		Fields:  Fields{"name": "Ana", "active": true},
		Details: DetailMutations{"destinations": {{Fields: Fields{"address": "ana@example.com"}}}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !uow.committed || len(audit.events) != 2 || parentSource.createCalls != 1 || childSource.createCalls != 1 {
		t.Fatalf("Create() was not atomic: uow=%#v audit=%#v parent=%#v child=%#v", uow, audit, parentSource, childSource)
	}
	if got := childSource.createMutations[0].Fields["parent_id"]; got != "contact-1" {
		t.Fatalf("child parent link = %#v, want injected parent id", got)
	}
	if childSource.listQuery.Filters[0] != (Filter{Field: "parent_id", Operator: FilterEqual, Value: "contact-1"}) {
		t.Fatalf("detail list query = %#v", childSource.listQuery)
	}
	got := record.Details["destinations"]
	if len(got) != 1 || got[0].Fields["address"] != "ana@example.com" {
		t.Fatalf("returned details = %#v", record.Details)
	}
	if _, leaked := got[0].Fields["parent_id"]; leaked {
		t.Fatal("internal child parent link crossed the public boundary")
	}
	if got[0].Details != nil {
		t.Fatal("undeclared child details crossed the public boundary")
	}
}

func TestMasterDetailRejectsUnauthorizedChildAndRollsBack(t *testing.T) {
	parentSource := &detailSource{created: Record{ID: "contact-1", Version: 1, Fields: Fields{"name": "Ana", "active": true}}}
	childSource := &detailSource{}
	uow, audit := &transactionUOW{}, &mutationAudit{}
	authorizer := &detailAuthorizer{deny: map[ResourceKey]map[Action]error{"contact_destinations": {ActionCreate: errors.New("denied")}}}
	service := newMasterDetailService(t, parentSource, childSource, uow, audit, 0, 2, authorizer)

	_, err := service.Create(context.Background(), "contacts", Mutation{
		Fields:  Fields{"name": "Ana", "active": true},
		Details: DetailMutations{"destinations": {{Fields: Fields{"address": "ana@example.com"}}}},
	})
	assertCode(t, err, ErrorForbidden)
	if uow.committed || childSource.createCalls != 0 {
		t.Fatalf("unauthorized child change must abort: uow=%#v child=%#v", uow, childSource)
	}
}

func TestMasterDetailQualifiesChildValidationErrorsForRenderer(t *testing.T) {
	service := newMasterDetailService(t, &detailSource{}, &detailSource{}, &transactionUOW{}, &mutationAudit{}, 0, 2, &detailAuthorizer{})
	_, err := service.Create(context.Background(), "contacts", Mutation{
		Fields:  Fields{"name": "Ana", "active": true},
		Details: DetailMutations{"destinations": {{Fields: Fields{"address": ""}}}},
	})
	var public *Error
	if !errors.As(err, &public) || public.Code != ErrorValidationFailed || public.Fields["destinations.address"] == "" {
		t.Fatalf("Create() child validation error = %#v, want qualified destination field", public)
	}
	if _, leaked := public.Fields["address"]; leaked {
		t.Fatalf("child validation error must not use ambiguous field key: %#v", public.Fields)
	}
}

func TestMasterDetailEnforcesMinimumAndLoadsDetailsOnGet(t *testing.T) {
	parentSource := &detailSource{
		created: Record{ID: "contact-1", Version: 1, Fields: Fields{"name": "Ana", "active": true}},
		current: Record{ID: "contact-1", Version: 1, Fields: Fields{"name": "Ana", "active": true}},
	}
	childSource := &detailSource{list: Page{
		Records: []Record{{ID: "destination-1", Version: 1, Fields: Fields{
			"parent_id": "contact-1", "address": "ana@example.com", "active": true,
		}}},
		Total: total(1),
	}}
	service := newMasterDetailService(t, parentSource, childSource, &transactionUOW{}, &mutationAudit{}, 1, 2, &detailAuthorizer{})

	got, err := service.Get(context.Background(), "contacts", "contact-1")
	if err != nil || len(got.Details["destinations"]) != 1 {
		t.Fatalf("Get() = %#v, %v", got, err)
	}

	childSource.list = Page{Total: total(0)}
	_, err = service.Create(context.Background(), "contacts", Mutation{Fields: Fields{"name": "Ana", "active": true}})
	assertCode(t, err, ErrorValidationFailed)
}

func TestMasterDetailStartupRejectsUnsafeRelationship(t *testing.T) {
	registry := NewRegistry()
	parent := validDefinition("contacts")
	parent.Details = []DetailDefinition{{Key: "destinations", Resource: "contact_destinations", ParentField: "parent_id", Maximum: 2, AllowCreate: true}}
	child := validDefinition("contact_destinations")
	child.Fields = append(child.Fields, Field{Key: "parent_id", Label: "crud.parent", Type: FieldString, Visible: true, ReadOnly: true})
	if err := registry.Register(context.Background(), parent); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(context.Background(), child); err != nil {
		t.Fatal(err)
	}
	_, err := NewService(Dependencies{Registry: registry, Principal: principalStub{}, Scope: scopeStub{scope: Scope{"tenant_id": "tenant-a"}}, Authorizer: &detailAuthorizer{}, Audit: auditStub{}, Translator: translatorStub{}, Clock: clockStub{}})
	var definitionError *DefinitionError
	if !errors.As(err, &definitionError) || definitionError.Path != "details[0].parent_field" {
		t.Fatalf("NewService() error = %v", err)
	}
}

func TestMasterDetailUpdateAndDeleteUseBoundChildAndAudit(t *testing.T) {
	parentSource := &detailSource{
		current: Record{ID: "contact-1", Version: 1, Fields: Fields{"name": "Ana", "active": true}},
		updated: Record{ID: "contact-1", Version: 2, Fields: Fields{"name": "Ana Maria", "active": true}},
	}
	childSource := &detailSource{
		current: Record{ID: "destination-1", Version: 1, Fields: Fields{"parent_id": "contact-1", "address": "ana@example.com"}},
		updated: Record{ID: "destination-1", Version: 2, Fields: Fields{"parent_id": "contact-1", "address": "new@example.com"}},
		list:    Page{Total: total(0)},
	}
	uow, audit := &transactionUOW{}, &mutationAudit{}
	service := newMasterDetailService(t, parentSource, childSource, uow, audit, 0, 2, &detailAuthorizer{})

	_, err := service.Update(context.Background(), "contacts", "contact-1", 1, Mutation{
		Fields: Fields{"name": "Ana Maria", "active": true},
		Details: DetailMutations{"destinations": {
			{ID: "destination-1", Version: 1, Fields: Fields{"address": "new@example.com"}},
			{ID: "destination-1", Version: 2, Delete: true},
		}},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !uow.committed || parentSource.updateCalls != 1 || childSource.updateCalls != 1 || childSource.deleteCalls != 1 || len(audit.events) != 3 {
		t.Fatalf("Update() did not apply all mutations: parent=%#v child=%#v audit=%#v", parentSource, childSource, audit)
	}
	if got := childSource.updateMutations[0].Fields["parent_id"]; got != "contact-1" {
		t.Fatalf("updated child parent link = %#v", got)
	}
	if childSource.deleteMode != DeleteModeArchive || childSource.deleteVersion != 2 {
		t.Fatalf("child delete = %#v", childSource)
	}
}

func TestMasterDetailRejectsChildFromAnotherParent(t *testing.T) {
	parentSource := &detailSource{
		current: Record{ID: "contact-1", Version: 1, Fields: Fields{"name": "Ana", "active": true}},
		updated: Record{ID: "contact-1", Version: 2, Fields: Fields{"name": "Ana", "active": true}},
	}
	childSource := &detailSource{current: Record{ID: "destination-2", Version: 1, Fields: Fields{"parent_id": "contact-2", "address": "other@example.com"}}}
	uow := &transactionUOW{}
	service := newMasterDetailService(t, parentSource, childSource, uow, &mutationAudit{}, 0, 2, &detailAuthorizer{})
	_, err := service.Update(context.Background(), "contacts", "contact-1", 1, Mutation{
		Fields:  Fields{"name": "Ana", "active": true},
		Details: DetailMutations{"destinations": {{ID: "destination-2", Version: 1, Fields: Fields{"address": "new@example.com"}}}},
	})
	assertCode(t, err, ErrorNotFound)
	if uow.committed || childSource.updateCalls != 0 {
		t.Fatalf("foreign child must not be changed: uow=%#v child=%#v", uow, childSource)
	}
}

func TestMasterDetailDefinitionExposesOnlyAuthorizedVisibleChildMetadata(t *testing.T) {
	service := newMasterDetailService(t, &detailSource{}, &detailSource{}, &transactionUOW{}, &mutationAudit{}, 0, 2, &detailAuthorizer{})
	definition, err := service.Definition(context.Background(), "contacts")
	if err != nil || len(definition.Details) != 1 {
		t.Fatalf("Definition() = %#v, %v", definition, err)
	}
	detail := definition.Details[0]
	if definition.Delete.Mode != DeleteModeArchive || detail.Labels.Title != "crud.contact_categories.title" || !detail.AllowCreate || !detail.AllowUpdate || !detail.AllowDelete || len(detail.Fields) != 1 || detail.Fields[0].Key != "address" {
		t.Fatalf("public detail = %#v", detail)
	}

	denied := &detailAuthorizer{deny: map[ResourceKey]map[Action]error{"contact_destinations": {ActionRead: errors.New("denied")}}}
	service = newMasterDetailService(t, &detailSource{}, &detailSource{}, &transactionUOW{}, &mutationAudit{}, 0, 2, denied)
	definition, err = service.Definition(context.Background(), "contacts")
	if err != nil || len(definition.Details) != 0 {
		t.Fatalf("Definition() leaked unauthorized child metadata: %#v, %v", definition, err)
	}
}

func TestDetailDefinitionValidationRejectsInvalidBoundsAndKeys(t *testing.T) {
	for _, detail := range []DetailDefinition{
		{Key: "Bad", Resource: "child", ParentField: "parent_id", Maximum: 1},
		{Key: "items", Resource: "child", ParentField: "parent_id", Maximum: 0},
		{Key: "items", Resource: "child", ParentField: "parent_id", Minimum: 2, Maximum: 1},
	} {
		definition := validDefinition("parent")
		definition.Details = []DetailDefinition{detail}
		if err := ValidateDefinition(context.Background(), definition); err == nil {
			t.Fatalf("ValidateDefinition(%#v) accepted invalid detail", detail)
		}
	}
	definition := validDefinition("parent")
	definition.Details = []DetailDefinition{
		{Key: "items", Resource: "child", ParentField: "parent_id", Maximum: 1},
		{Key: "items", Resource: "other", ParentField: "parent_id", Maximum: 1},
	}
	if err := ValidateDefinition(context.Background(), definition); err == nil {
		t.Fatal("duplicate detail key accepted")
	}
}

func TestMasterDetailStartupRejectsMissingChildResource(t *testing.T) {
	registry := NewRegistry()
	parent := validDefinition("parent")
	parent.Details = []DetailDefinition{{Key: "items", Resource: "missing", ParentField: "parent_id", Maximum: 1}}
	if err := registry.Register(context.Background(), parent); err != nil {
		t.Fatal(err)
	}
	_, err := NewService(Dependencies{Registry: registry, Principal: principalStub{}, Scope: scopeStub{scope: Scope{"tenant_id": "tenant-a"}}, Authorizer: &detailAuthorizer{}, Audit: auditStub{}, Translator: translatorStub{}, Clock: clockStub{}})
	var definitionError *DefinitionError
	if !errors.As(err, &definitionError) || definitionError.Path != "details[0].resource" {
		t.Fatalf("NewService() error = %v", err)
	}
}

func newMasterDetailService(t *testing.T, parentSource, childSource *detailSource, uow *transactionUOW, audit *mutationAudit, minimum, maximum uint16, authorizer Authorizer) *Service {
	t.Helper()
	registry := NewRegistry()
	parent := validDefinition("contacts")
	parent.Source, parent.UOW = parentSource, uow
	parent.Details = []DetailDefinition{{Key: "destinations", Resource: "contact_destinations", ParentField: "parent_id", Minimum: minimum, Maximum: maximum, AllowCreate: true, AllowUpdate: true, AllowDelete: true}}
	child := validDefinition("contact_destinations")
	child.Source, child.UOW = childSource, uow
	child.Fields = []Field{
		Field{Key: "address", Label: "crud.destination.address", Type: FieldEmail, Required: true, Visible: true},
		Field{Key: "parent_id", Label: "crud.destination.parent", Type: FieldString, Required: true, ReadOnly: true, Visible: false},
	}
	child.List.Columns = []FieldKey{"address"}
	child.List.Searchable = []FieldKey{"address"}
	child.List.Sortable = []FieldKey{"address"}
	child.List.DefaultSort = []Sort{{Field: "address", Direction: SortAscending}}
	if err := registry.Register(context.Background(), parent); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(context.Background(), child); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(Dependencies{Registry: registry, Principal: principalStub{}, Scope: scopeStub{scope: Scope{"tenant_id": "tenant-a"}}, Authorizer: authorizer, Audit: audit, Translator: translatorStub{}, Clock: fixedClock{}})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func total(value uint64) *uint64 { return &value }

type detailAuthorizer struct {
	deny map[ResourceKey]map[Action]error
}

func (authorizer *detailAuthorizer) Authorize(_ context.Context, _ Principal, key ResourceKey, action Action, _ *Record) error {
	return authorizer.deny[key][action]
}

type detailSource struct {
	created         Record
	current         Record
	updated         Record
	list            Page
	createCalls     int
	createMutations []Mutation
	updateCalls     int
	updateMutations []Mutation
	deleteCalls     int
	deleteVersion   Version
	deleteMode      DeleteMode
	listQuery       Query
}

func (*detailSource) Capabilities(context.Context) Capabilities { return mutableCapabilities() }
func (source *detailSource) List(_ context.Context, _ Scope, query Query) (Page, error) {
	source.listQuery = query
	return source.list, nil
}
func (source *detailSource) Get(_ context.Context, _ Scope, _ RecordID) (Record, error) {
	return source.current, nil
}
func (source *detailSource) Create(_ context.Context, _ Scope, mutation Mutation) (Record, error) {
	source.createCalls++
	source.createMutations = append(source.createMutations, mutation)
	record := source.created
	if record.Fields == nil {
		record.Fields = cloneFields(mutation.Fields)
	}
	return record, nil
}
func (source *detailSource) Update(_ context.Context, _ Scope, _ RecordID, _ Version, mutation Mutation) (Record, error) {
	source.updateCalls++
	source.updateMutations = append(source.updateMutations, mutation)
	return source.updated, nil
}
func (source *detailSource) Delete(_ context.Context, _ Scope, _ RecordID, version Version, mode DeleteMode) error {
	source.deleteCalls++
	source.deleteVersion, source.deleteMode = version, mode
	return nil
}
func (*detailSource) Lookup(context.Context, Scope, LookupQuery) (LookupPage, error) {
	return LookupPage{}, nil
}
