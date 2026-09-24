package conformance

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"

	crud "github.com/clear-platform-br/clear-crud"
)

func TestDataSourceWithReferenceFixture(t *testing.T) {
	TestDataSource(t, newReferenceFixture)
}

func TestFixtureHelpers(t *testing.T) {
	t.Run("scope equality", func(t *testing.T) {
		if !sameScope(crud.Scope{"tenant_id": "a"}, crud.Scope{"tenant_id": "a"}) {
			t.Fatal("equal scopes were different")
		}
		if sameScope(crud.Scope{"tenant_id": "a"}, crud.Scope{"tenant_id": "b"}) {
			t.Fatal("different scope values compared equal")
		}
		if sameScope(crud.Scope{"tenant_id": "a"}, crud.Scope{"tenant_id": "a", "region": "br"}) {
			t.Fatal("different scope sizes compared equal")
		}
	})

	t.Run("public error code", func(t *testing.T) {
		if got := codeOf(publicError(crud.ErrorNotFound)); got != crud.ErrorNotFound {
			t.Fatalf("codeOf public error = %q", got)
		}
		if got := codeOf(fmt.Errorf("internal")); got != "" {
			t.Fatalf("codeOf internal error = %q", got)
		}
	})
}

func newReferenceFixture(testing.TB) Fixture {
	source := &referenceSource{
		records: make(map[string]map[crud.RecordID]crud.Record),
		capabilities: crud.Capabilities{
			crud.CapabilityTotalCount:    {},
			crud.CapabilityOffsetPage:    {},
			crud.CapabilityCursorPage:    {},
			crud.CapabilityAtomicVersion: {},
			crud.CapabilityUnitOfWork:    {},
			crud.CapabilityArchive:       {},
			crud.CapabilityHardDelete:    {},
			crud.CapabilityLookup:        {},
		},
	}
	return Fixture{
		Source:      source,
		UnitOfWork:  referenceUnitOfWork{source: source},
		ScopeA:      crud.Scope{"tenant_id": "tenant-a"},
		ScopeB:      crud.Scope{"tenant_id": "tenant-b"},
		ListQuery:   crud.Query{},
		LookupQuery: &crud.LookupQuery{Size: 10},
		NewMutation: func(label string) crud.Mutation {
			return crud.Mutation{Fields: crud.Fields{"name": label}}
		},
	}
}

type referenceUnitOfWork struct {
	source *referenceSource
}

func (unit referenceUnitOfWork) Within(ctx context.Context, operation func(context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	unit.source.mu.Lock()
	snapshot := unit.source.snapshot()
	unit.source.mu.Unlock()
	if err := operation(ctx); err != nil {
		unit.source.mu.Lock()
		unit.source.records = snapshot.records
		unit.source.next = snapshot.next
		unit.source.mu.Unlock()
		return err
	}
	return nil
}

type referenceSource struct {
	mu           sync.Mutex
	next         uint64
	records      map[string]map[crud.RecordID]crud.Record
	capabilities crud.Capabilities
}

func (source *referenceSource) Capabilities(context.Context) crud.Capabilities {
	return source.capabilities
}

func (source *referenceSource) List(ctx context.Context, scope crud.Scope, query crud.Query) (crud.Page, error) {
	if err := ctx.Err(); err != nil {
		return crud.Page{}, err
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	records := source.recordsFor(scope)
	ids := make([]string, 0, len(records))
	for id := range records {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	all := make([]crud.Record, 0, len(ids))
	for _, id := range ids {
		all = append(all, cloneRecord(records[crud.RecordID(id)]))
	}
	total := uint64(len(all))
	page := crud.Page{Records: all, Total: &total}
	if query.Page.Size == 0 {
		return page, nil
	}
	page.Size = query.Page.Size
	switch query.Page.Mode {
	case crud.PageModeOffset:
		page.Page = query.Page.Number
		start := int((query.Page.Number - 1) * uint64(query.Page.Size))
		if start >= len(all) {
			page.Records = nil
			return page, nil
		}
		end := min(start+int(query.Page.Size), len(all))
		page.Records = all[start:end]
	case crud.PageModeCursor:
		page.Records = all[:min(int(query.Page.Size), len(all))]
		if len(all) > len(page.Records) {
			page.NextCursor = "next"
		}
	}
	return page, nil
}

func (source *referenceSource) Get(ctx context.Context, scope crud.Scope, id crud.RecordID) (crud.Record, error) {
	if err := ctx.Err(); err != nil {
		return crud.Record{}, err
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	record, ok := source.recordsFor(scope)[id]
	if !ok {
		return crud.Record{}, publicError(crud.ErrorNotFound)
	}
	return cloneRecord(record), nil
}

func (source *referenceSource) Create(ctx context.Context, scope crud.Scope, mutation crud.Mutation) (crud.Record, error) {
	if err := ctx.Err(); err != nil {
		return crud.Record{}, err
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	source.next++
	record := crud.Record{
		ID:      crud.RecordID(fmt.Sprintf("record-%d", source.next)),
		Version: 1,
		Fields:  cloneFields(mutation.Fields),
	}
	source.recordsFor(scope)[record.ID] = record
	return cloneRecord(record), nil
}

func (source *referenceSource) Update(ctx context.Context, scope crud.Scope, id crud.RecordID, version crud.Version, mutation crud.Mutation) (crud.Record, error) {
	if err := ctx.Err(); err != nil {
		return crud.Record{}, err
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	record, ok := source.recordsFor(scope)[id]
	if !ok {
		return crud.Record{}, publicError(crud.ErrorNotFound)
	}
	if record.Version != version {
		return crud.Record{}, publicError(crud.ErrorConflict)
	}
	record.Version++
	record.Fields = cloneFields(mutation.Fields)
	source.recordsFor(scope)[id] = record
	return cloneRecord(record), nil
}

func (source *referenceSource) Delete(ctx context.Context, scope crud.Scope, id crud.RecordID, version crud.Version, mode crud.DeleteMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if mode == crud.DeleteModeNone {
		return publicError(crud.ErrorDeleteRestricted)
	}
	source.mu.Lock()
	defer source.mu.Unlock()
	record, ok := source.recordsFor(scope)[id]
	if !ok {
		return publicError(crud.ErrorNotFound)
	}
	if record.Version != version {
		return publicError(crud.ErrorConflict)
	}
	delete(source.recordsFor(scope), id)
	return nil
}

func (source *referenceSource) Lookup(ctx context.Context, scope crud.Scope, query crud.LookupQuery) (crud.LookupPage, error) {
	if err := ctx.Err(); err != nil {
		return crud.LookupPage{}, err
	}
	if len(scope) == 0 || query.Size == 0 {
		return crud.LookupPage{}, publicError(crud.ErrorInvalidRequest)
	}
	total := uint64(1)
	return crud.LookupPage{
		Options: []crud.LookupOption{{Value: "option-1", Label: "Option 1"}},
		Total:   &total,
	}, nil
}

func (source *referenceSource) recordsFor(scope crud.Scope) map[crud.RecordID]crud.Record {
	key := scope["tenant_id"]
	records, ok := source.records[key]
	if !ok {
		records = make(map[crud.RecordID]crud.Record)
		source.records[key] = records
	}
	return records
}

type referenceSnapshot struct {
	next    uint64
	records map[string]map[crud.RecordID]crud.Record
}

func (source *referenceSource) snapshot() referenceSnapshot {
	copy := make(map[string]map[crud.RecordID]crud.Record, len(source.records))
	for scope, records := range source.records {
		copy[scope] = make(map[crud.RecordID]crud.Record, len(records))
		for id, record := range records {
			copy[scope][id] = cloneRecord(record)
		}
	}
	return referenceSnapshot{next: source.next, records: copy}
}

func publicError(code crud.ErrorCode) *crud.Error {
	return &crud.Error{Code: code, Message: crud.MessageCode("crud.error." + string(code))}
}

func cloneRecord(record crud.Record) crud.Record {
	record.Fields = cloneFields(record.Fields)
	return record
}

func cloneFields(fields crud.Fields) crud.Fields {
	clone := make(crud.Fields, len(fields))
	for key, value := range fields {
		clone[key] = value
	}
	return clone
}
