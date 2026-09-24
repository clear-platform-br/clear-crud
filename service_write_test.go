package crud

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceCreateWritesAuditInTheSameUnitOfWork(t *testing.T) {
	t.Parallel()

	source := &mutationSource{record: Record{ID: "new-1", Version: 1, Fields: Fields{"name": "Ana", "active": true}}}
	uow := &transactionUOW{}
	audit := &mutationAudit{}
	afterCommit := false
	service := newMutationService(t, source, uow, audit, func(definition *Definition) {
		definition.Hooks.BeforeCreate = func(ctx context.Context, scope Scope, mutation Mutation) error {
			if ctx.Value(transactionKey{}) != "transaction" || scope["tenant_id"] != "tenant-a" || mutation.Fields["name"] != "Ana" {
				t.Fatal("BeforeCreate did not receive transaction, scope, and mutation")
			}
			return nil
		}
		definition.Hooks.AfterCommit = func(context.Context, MutationEvent) error {
			afterCommit = true
			return nil
		}
	})

	record, err := service.Create(context.Background(), "contact_categories", Mutation{Fields: Fields{"name": "Ana", "active": true}})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if record.ID != "new-1" || source.createCalls != 1 || len(audit.events) != 1 || !uow.committed || !afterCommit {
		t.Fatalf("Create() did not complete atomically: %#v %#v", source, audit)
	}
	event := audit.events[0]
	if event.Action != ActionCreate || event.BeforeVersion != 0 || event.AfterVersion != 1 || event.Scope["tenant_id"] != "tenant-a" {
		t.Fatalf("create audit event = %#v", event)
	}
	if !source.inTransaction || !audit.inTransaction {
		t.Fatal("persistence and audit must share the unit-of-work context")
	}
}

func TestServiceUpdateAndDeleteUseCurrentRecordAndVersion(t *testing.T) {
	t.Parallel()

	source := &mutationSource{
		current: Record{ID: "1", Version: 2, Fields: Fields{"name": "Before", "active": true}},
		record:  Record{ID: "1", Version: 3, Fields: Fields{"name": "After", "active": true}},
	}
	uow, audit := &transactionUOW{}, &mutationAudit{}
	authorizer := &recordingAuthorizer{}
	service := newMutationService(t, source, uow, audit, nil)
	service.authorizer = authorizer

	record, err := service.Update(context.Background(), "contact_categories", "1", 2, Mutation{Fields: Fields{"name": "After", "active": true}})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if record.Version != 3 || source.updateVersion != 2 || authorizer.currentCalls != 1 {
		t.Fatalf("Update() did not use optimistic version/current authorization: %#v", source)
	}
	if event := audit.events[0]; event.Action != ActionUpdate || event.BeforeVersion != 2 || event.AfterVersion != 3 {
		t.Fatalf("update event = %#v", event)
	}

	if err := service.Delete(context.Background(), "contact_categories", "1", 3); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if source.deleteMode != DeleteModeSoftDelete || source.deleteVersion != 3 || authorizer.currentCalls != 2 {
		t.Fatalf("Delete() did not use archive policy/current authorization: %#v", source)
	}
	if event := audit.events[1]; event.Action != ActionDelete || event.BeforeVersion != 3 || event.AfterVersion != 4 {
		t.Fatalf("delete event = %#v", event)
	}
}

func TestServiceMutationFailureIsSafeAndDoesNotRunAfterCommit(t *testing.T) {
	t.Parallel()

	source := &mutationSource{record: Record{ID: "1", Version: 1}}
	uow := &transactionUOW{}
	audit := &mutationAudit{err: errors.New("audit storage unavailable")}
	afterCommit := false
	service := newMutationService(t, source, uow, audit, func(definition *Definition) {
		definition.Hooks.AfterCommit = func(context.Context, MutationEvent) error {
			afterCommit = true
			return errors.New("must be ignored after a successful commit only")
		}
	})
	_, err := service.Create(context.Background(), "contact_categories", Mutation{Fields: Fields{"name": "Ana", "active": true}})
	assertCode(t, err, ErrorTemporarilyUnavailable)
	if uow.committed || afterCommit {
		t.Fatal("failed audit must abort the unit of work and skip AfterCommit")
	}

	_, err = service.Update(context.Background(), "contact_categories", "1", 0, Mutation{})
	assertCode(t, err, ErrorInvalidRequest)
	err = service.Delete(context.Background(), "contact_categories", "", 1)
	assertCode(t, err, ErrorInvalidRequest)
}

func TestNormalizeMutationRejectsUnknownReadOnlyAndInvalidValues(t *testing.T) {
	t.Parallel()

	definition := validDefinition("categories")
	definition.Fields = append(definition.Fields,
		Field{Key: "count", Label: "crud.count", Type: FieldInteger, Visible: true, Minimum: "1", Maximum: "10"},
		Field{Key: "amount", Label: "crud.amount", Type: FieldDecimal, Visible: true, Minimum: "2.5", Maximum: "10.5"},
		Field{Key: "date", Label: "crud.date", Type: FieldDate, Visible: true},
		Field{Key: "email", Label: "crud.email", Type: FieldEmail, Visible: true},
		Field{Key: "system", Label: "crud.system", Type: FieldString, Visible: true, ReadOnly: true},
	)
	definition.Validator = func(_ context.Context, scope Scope, mutation Mutation) FieldErrors {
		if scope["tenant_id"] != "tenant-a" || mutation.Fields["name"] == "blocked" {
			return FieldErrors{"name": "crud.name.blocked"}
		}
		return nil
	}
	valid := Mutation{Fields: Fields{
		"name": "Ana", "active": true, "count": int64(2), "amount": "2.50", "date": "2026-09-24", "email": "ana@example.com",
	}}
	if mutation, err := normalizeMutation(context.Background(), Scope{"tenant_id": "tenant-a"}, definition, valid); err != nil || mutation.Fields["system"] != nil {
		t.Fatalf("normalizeMutation() = %#v, %v", mutation, err)
	}
	for _, mutation := range []Mutation{
		{Fields: Fields{"active": true}},
		{Fields: Fields{"name": "Ana", "active": true, "unknown": "x"}},
		{Fields: Fields{"name": "Ana", "active": true, "system": "x"}},
		{Fields: Fields{"name": "Ana", "active": true, "count": "two"}},
		{Fields: Fields{"name": "Ana", "active": true, "amount": "2.4"}},
		{Fields: Fields{"name": "Ana", "active": true, "date": "24/09/2026"}},
		{Fields: Fields{"name": "Ana", "active": true, "email": "not an email"}},
		{Fields: Fields{"name": "blocked", "active": true}},
	} {
		if _, err := normalizeMutation(context.Background(), Scope{"tenant_id": "tenant-a"}, definition, mutation); err == nil {
			t.Fatalf("normalizeMutation(%#v) accepted invalid input", mutation)
		}
	}
}

func TestMutationValidationCoversFormatsEnumsHooksAndBounds(t *testing.T) {
	t.Parallel()

	definition := validDefinition("strict")
	definition.Fields = append(definition.Fields,
		Field{Key: "status", Label: "crud.status", Type: FieldEnum, Visible: true, Enum: []Option{{Value: "open", Label: "crud.open"}}},
		Field{Key: "short", Label: "crud.short", Type: FieldString, Visible: true, MinLength: 2, MaxLength: 3},
		Field{Key: "at", Label: "crud.at", Type: FieldDateTime, Visible: true, Minimum: "2026-01-01T00:00:00Z", Maximum: "2026-12-31T23:59:59Z"},
	)
	valid := Mutation{Fields: Fields{"name": "Ana", "active": true, "status": "open", "short": "ok", "at": "2026-09-24T12:00:00Z"}}
	if _, err := normalizeMutation(context.Background(), Scope{}, definition, valid); err != nil {
		t.Fatalf("valid mutation error = %v", err)
	}
	for _, mutation := range []Mutation{
		{Fields: Fields{"name": "Ana", "active": true, "status": "closed"}},
		{Fields: Fields{"name": "Ana", "active": true, "status": "open", "short": "x"}},
		{Fields: Fields{"name": "Ana", "active": true, "status": "open", "short": "long"}},
		{Fields: Fields{"name": "Ana", "active": true, "status": "open", "at": "not-rfc3339"}},
		{Fields: Fields{"name": "Ana", "active": true, "status": "open", "at": "2027-01-01T00:00:00Z"}},
	} {
		if _, err := normalizeMutation(context.Background(), Scope{}, definition, mutation); err == nil {
			t.Fatalf("normalizeMutation(%#v) accepted invalid value", mutation)
		}
	}

	definition.Hooks.BeforeValidate = func(context.Context, Mutation) (Mutation, error) {
		return Mutation{Fields: Fields{"name": "Ana", "active": true, "not_allowed": "x"}}, nil
	}
	if _, err := normalizeMutation(context.Background(), Scope{}, definition, valid); err == nil {
		t.Fatal("BeforeValidate must not inject an undeclared field")
	}
	definition.Hooks.BeforeValidate = func(context.Context, Mutation) (Mutation, error) {
		return Mutation{}, errors.New("internal hook error")
	}
	if _, err := normalizeMutation(context.Background(), Scope{}, definition, valid); err == nil {
		t.Fatal("hook failure must be sanitized")
	} else {
		assertCode(t, err, ErrorTemporarilyUnavailable)
	}
}

func TestServiceWriteFailurePathsDoNotAdvance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func(*mutationSource, *recordingAuthorizer, *Definition)
		call   func(*Service) error
		want   ErrorCode
	}{
		{
			name: "preliminary authorization",
			change: func(_ *mutationSource, authorizer *recordingAuthorizer, _ *Definition) {
				authorizer.deny = map[Action]error{ActionCreate: errors.New("denied")}
			},
			call: func(service *Service) error {
				_, err := service.Create(context.Background(), "contact_categories", Mutation{Fields: Fields{"name": "Ana", "active": true}})
				return err
			},
			want: ErrorForbidden,
		},
		{
			name: "current authorization",
			change: func(_ *mutationSource, authorizer *recordingAuthorizer, _ *Definition) {
				authorizer.denyCurrent = errors.New("record denied")
			},
			call: func(service *Service) error {
				_, err := service.Update(context.Background(), "contact_categories", "1", 1, Mutation{Fields: Fields{"name": "Ana", "active": true}})
				return err
			},
			want: ErrorForbidden,
		},
		{
			name: "update persistence",
			change: func(source *mutationSource, _ *recordingAuthorizer, _ *Definition) {
				source.updateErr = errors.New("driver failure")
			},
			call: func(service *Service) error {
				_, err := service.Update(context.Background(), "contact_categories", "1", 1, Mutation{Fields: Fields{"name": "Ana", "active": true}})
				return err
			},
			want: ErrorTemporarilyUnavailable,
		},
		{
			name: "delete persistence",
			change: func(source *mutationSource, _ *recordingAuthorizer, _ *Definition) {
				source.deleteErr = errors.New("driver failure")
			},
			call: func(service *Service) error {
				return service.Delete(context.Background(), "contact_categories", "1", 1)
			},
			want: ErrorTemporarilyUnavailable,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source, uow, audit, authorizer := &mutationSource{current: Record{ID: "1", Version: 1}}, &transactionUOW{}, &mutationAudit{}, &recordingAuthorizer{}
			test.change(source, authorizer, nil)
			service := newMutationService(t, source, uow, audit, nil)
			service.authorizer = authorizer
			assertCode(t, test.call(service), test.want)
			if test.name != "preliminary authorization" {
				// Persistence failures are sanitized as temporary unavailability.
				if test.name == "update persistence" || test.name == "delete persistence" {
					if uow.committed {
						t.Fatal("failed persistence must abort the unit of work")
					}
				}
			}
		})
	}
}

func TestServiceWriteRejectsUnresolvedOrUnavailableContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func(*Service)
		want   ErrorCode
	}{
		{"unknown resource", func(*Service) {}, ErrorNotFound},
		{"principal failure", func(service *Service) { service.principal = principalStub{err: errors.New("identity down")} }, ErrorUnauthenticated},
		{"scope failure", func(service *Service) { service.scope = scopeStub{err: errors.New("scope down")} }, ErrorForbidden},
		{"missing scope", func(service *Service) { service.scope = scopeStub{scope: Scope{}} }, ErrorForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newMutationService(t, &mutationSource{}, &transactionUOW{}, &mutationAudit{}, nil)
			test.change(service)
			key := ResourceKey("contact_categories")
			if test.name == "unknown resource" {
				key = "missing"
			}
			_, err := service.Create(context.Background(), key, Mutation{Fields: Fields{"name": "Ana", "active": true}})
			assertCode(t, err, test.want)
		})
	}
}

func TestServiceRejectsDisabledWriteAction(t *testing.T) {
	t.Parallel()

	service := newMutationService(t, &mutationSource{}, &transactionUOW{}, &mutationAudit{}, func(definition *Definition) {
		definition.Permissions.Create = ""
	})
	_, err := service.Create(context.Background(), "contact_categories", Mutation{Fields: Fields{"name": "Ana", "active": true}})
	assertCode(t, err, ErrorForbidden)
}

func TestServiceWriteSanitizesHooksAndDataSourceFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		change func(*mutationSource, *Definition)
		call   func(*Service) error
	}{
		{
			"before create hook",
			func(_ *mutationSource, definition *Definition) {
				definition.Hooks.BeforeCreate = func(context.Context, Scope, Mutation) error { return errors.New("hook internals") }
			},
			func(service *Service) error {
				_, err := service.Create(context.Background(), "contact_categories", Mutation{Fields: Fields{"name": "Ana", "active": true}})
				return err
			},
		},
		{
			"create data source",
			func(source *mutationSource, _ *Definition) { source.createErr = errors.New("driver internals") },
			func(service *Service) error {
				_, err := service.Create(context.Background(), "contact_categories", Mutation{Fields: Fields{"name": "Ana", "active": true}})
				return err
			},
		},
		{
			"update get data source",
			func(source *mutationSource, _ *Definition) { source.getErr = errors.New("driver internals") },
			func(service *Service) error {
				_, err := service.Update(context.Background(), "contact_categories", "1", 1, Mutation{Fields: Fields{"name": "Ana", "active": true}})
				return err
			},
		},
		{
			"before delete hook",
			func(_ *mutationSource, definition *Definition) {
				definition.Hooks.BeforeDelete = func(context.Context, Scope, Record) error { return errors.New("hook internals") }
			},
			func(service *Service) error {
				return service.Delete(context.Background(), "contact_categories", "1", 1)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source, uow := &mutationSource{current: Record{ID: "1", Version: 1}}, &transactionUOW{}
			service := newMutationService(t, source, uow, &mutationAudit{}, func(definition *Definition) { test.change(source, definition) })
			err := test.call(service)
			assertCode(t, err, ErrorTemporarilyUnavailable)
			if uow.committed {
				t.Fatal("a failed hook or data source must not commit")
			}
		})
	}
}

func TestMutationFieldValidationTypeAndRangeBoundaries(t *testing.T) {
	t.Parallel()

	fields := []Field{
		{Key: "integer", Type: FieldInteger, Minimum: "1", Maximum: "2"},
		{Key: "boolean", Type: FieldBoolean},
		{Key: "decimal", Type: FieldDecimal, Minimum: "2.5", Maximum: "3.5"},
		{Key: "date", Type: FieldDate, Minimum: "2026-01-01", Maximum: "2026-12-31"},
	}
	for _, test := range []struct {
		field Field
		value Value
		ok    bool
	}{
		{fields[0], int64(1), true}, {fields[0], int64(3), false}, {fields[0], true, false},
		{fields[1], true, true}, {fields[1], "true", false},
		{fields[2], "2.50", true}, {fields[2], "2.4", false}, {fields[2], "3.6", false},
		{fields[3], "2026-06-01", true}, {fields[3], "2025-12-31", false},
	} {
		err := validateFieldValue(test.field, test.value)
		if (err == nil) != test.ok {
			t.Fatalf("validateFieldValue(%#v, %#v) = %v, want success %t", test.field, test.value, err, test.ok)
		}
	}
}

func TestServiceHardDeleteAndAfterCommitFailureDoNotChangePublicSuccess(t *testing.T) {
	t.Parallel()

	capabilities := mutableCapabilities()
	capabilities[CapabilityHardDelete] = struct{}{}
	source := &mutationSource{capabilities: capabilities, current: Record{ID: "1", Version: 7}}
	audit := &mutationAudit{}
	service := newMutationService(t, source, &transactionUOW{}, audit, func(definition *Definition) {
		definition.Delete.Mode = DeleteModeHardDelete
		definition.Hooks.AfterCommit = func(context.Context, MutationEvent) error {
			return errors.New("post-commit notification failed")
		}
	})
	if err := service.Delete(context.Background(), "contact_categories", "1", 7); err != nil {
		t.Fatalf("hard Delete() error = %v", err)
	}
	if source.deleteMode != DeleteModeHardDelete || audit.events[0].AfterVersion != 0 {
		t.Fatalf("hard delete audit = %#v, source = %#v", audit.events[0], source)
	}
}

func TestBeforeValidateCanNormalizeWithinTheAllowlist(t *testing.T) {
	t.Parallel()

	definition := validDefinition("normalized")
	definition.Hooks.BeforeValidate = func(_ context.Context, mutation Mutation) (Mutation, error) {
		mutation.Fields["name"] = "Ana"
		return mutation, nil
	}
	mutation, err := normalizeMutation(context.Background(), Scope{}, definition, Mutation{Fields: Fields{"name": " ana ", "active": true}})
	if err != nil || mutation.Fields["name"] != "Ana" {
		t.Fatalf("normalizeMutation() = %#v, %v", mutation, err)
	}
}

func TestServiceUpdateHookAndDeletePublicFailure(t *testing.T) {
	t.Parallel()

	source := &mutationSource{
		current: Record{ID: "1", Version: 1, Fields: Fields{"name": "Before", "active": true}},
		record:  Record{ID: "1", Version: 2, Fields: Fields{"name": "After", "active": true}},
	}
	beforeUpdate := false
	service := newMutationService(t, source, &transactionUOW{}, &mutationAudit{}, func(definition *Definition) {
		definition.Hooks.BeforeUpdate = func(ctx context.Context, scope Scope, current Record, mutation Mutation) error {
			beforeUpdate = ctx.Value(transactionKey{}) == "transaction" && scope["tenant_id"] == "tenant-a" && current.ID == "1" && mutation.Fields["name"] == "After"
			return nil
		}
	})
	record, err := service.Update(context.Background(), "contact_categories", "1", 1, Mutation{Fields: Fields{"name": "After", "active": true}})
	if err != nil || !beforeUpdate || record.Version != 2 {
		t.Fatalf("Update() = %#v, %v, hook=%t", record, err, beforeUpdate)
	}

	publicFailure := publicError(ErrorConflict, "crud.error.conflict", nil)
	source.deleteErr = publicFailure
	err = service.Delete(context.Background(), "contact_categories", "1", 2)
	if !errors.Is(err, publicFailure) {
		t.Fatalf("Delete() error = %v, want preserved conflict", err)
	}
}

func newMutationService(t *testing.T, source *mutationSource, uow *transactionUOW, audit *mutationAudit, change func(*Definition)) *Service {
	t.Helper()
	registry := NewRegistry()
	definition := validDefinition("contact_categories")
	definition.Source, definition.UOW = source, uow
	if change != nil {
		change(&definition)
	}
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	service, err := NewService(Dependencies{
		Registry: registry, Principal: principalStub{}, Scope: scopeStub{scope: Scope{"tenant_id": "tenant-a"}},
		Authorizer: &recordingAuthorizer{}, Audit: audit, Translator: translatorStub{}, Clock: fixedClock{},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

type transactionKey struct{}

type transactionUOW struct {
	committed bool
}

func (uow *transactionUOW) Within(ctx context.Context, operation func(context.Context) error) error {
	err := operation(context.WithValue(ctx, transactionKey{}, "transaction"))
	uow.committed = err == nil
	return err
}

type mutationAudit struct {
	events        []AuditEvent
	err           error
	inTransaction bool
}

func (audit *mutationAudit) Append(ctx context.Context, event AuditEvent) error {
	audit.inTransaction = ctx.Value(transactionKey{}) == "transaction"
	if audit.err != nil {
		return audit.err
	}
	audit.events = append(audit.events, event)
	return nil
}

type mutationSource struct {
	capabilities  Capabilities
	record        Record
	current       Record
	createCalls   int
	updateVersion Version
	deleteVersion Version
	deleteMode    DeleteMode
	inTransaction bool
	createErr     error
	updateErr     error
	deleteErr     error
	getErr        error
}

func (source *mutationSource) Capabilities(context.Context) Capabilities {
	if source.capabilities != nil {
		return source.capabilities
	}
	return mutableCapabilities()
}
func (*mutationSource) List(context.Context, Scope, Query) (Page, error) { return Page{}, nil }
func (source *mutationSource) Get(ctx context.Context, _ Scope, _ RecordID) (Record, error) {
	source.inTransaction = ctx.Value(transactionKey{}) == "transaction"
	return source.current, source.getErr
}
func (source *mutationSource) Create(ctx context.Context, _ Scope, _ Mutation) (Record, error) {
	source.inTransaction = ctx.Value(transactionKey{}) == "transaction"
	source.createCalls++
	return source.record, source.createErr
}
func (source *mutationSource) Update(ctx context.Context, _ Scope, _ RecordID, version Version, _ Mutation) (Record, error) {
	source.inTransaction = ctx.Value(transactionKey{}) == "transaction"
	source.updateVersion = version
	return source.record, source.updateErr
}
func (source *mutationSource) Delete(ctx context.Context, _ Scope, _ RecordID, version Version, mode DeleteMode) error {
	source.inTransaction, source.deleteVersion, source.deleteMode = ctx.Value(transactionKey{}) == "transaction", version, mode
	return source.deleteErr
}
func (*mutationSource) Lookup(context.Context, Scope, LookupQuery) (LookupPage, error) {
	return LookupPage{}, nil
}

type fixedClock struct{}

func (fixedClock) Now() time.Time { return time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC) }
