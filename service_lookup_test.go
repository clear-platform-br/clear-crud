package crud

import (
	"context"
	"runtime"
	"sync"
	"testing"
)

func TestServiceLookupUsesRegisteredTargetAndReadThroughCache(t *testing.T) {
	parentSource := &recordingSource{capabilities: capabilitiesWithLookup()}
	total := uint64(1)
	targetSource := &recordingSource{capabilities: Capabilities{CapabilityOffsetPage: {}, CapabilityTotalCount: {}, CapabilityLookup: {}}, lookupPage: LookupPage{Options: []LookupOption{{Value: int64(1), Label: "São Paulo"}}, Total: &total}}
	parent := validDefinition("lookup_contacts")
	parent.Source = parentSource
	parent.Fields = append(parent.Fields, Field{Key: "state_id", Label: "state", Type: FieldLookup, Visible: true, Lookup: &LookupDefinition{Resource: "reference_states", ValueField: "id", LabelField: "name", PageSize: 25}})
	target := validDefinition("reference_states")
	target.Scope = ScopeRequirements{Mode: ScopeModeGlobal}
	target.Permissions = Permissions{Read: "reference.read"}
	target.Delete = DeletePolicy{Mode: DeleteModeNone}
	target.Concurrency = ConcurrencyPolicy{Mode: ConcurrencyNone}
	target.UOW = nil
	target.Source = targetSource
	registry := NewRegistry()
	for _, definition := range []Definition{parent, target} {
		if err := registry.Register(context.Background(), definition); err != nil {
			t.Fatal(err)
		}
	}
	service, err := NewService(Dependencies{Registry: registry, Principal: principalStub{}, Scope: scopeStub{scope: Scope{"tenant_id": "tenant-a"}}, Authorizer: &recordingAuthorizer{}, Audit: auditStub{}, Translator: translatorStub{}, Clock: clockStub{}})
	if err != nil {
		t.Fatal(err)
	}
	query := LookupQuery{Search: "São", Dependencies: Fields{}}
	if _, err := service.Lookup(context.Background(), parent.Key, "state_id", query); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Lookup(context.Background(), parent.Key, "state_id", query); err != nil {
		t.Fatal(err)
	}
	if parentSource.lookupCalls != 0 || targetSource.lookupCalls != 1 {
		t.Fatalf("lookup calls parent=%d target=%d", parentSource.lookupCalls, targetSource.lookupCalls)
	}
	if targetSource.lookupQuery.Resource != "reference_states" || targetSource.lookupQuery.ValueField != "id" || targetSource.lookupQuery.LabelField != "name" {
		t.Fatalf("target metadata was not server-populated: %#v", targetSource.lookupQuery)
	}
}

func TestServiceLookupAppliesServerOwnedFixedFiltersAndSeparatesCache(t *testing.T) {
	parentSource := &recordingSource{capabilities: capabilitiesWithLookup()}
	targetSource := &recordingSource{capabilities: Capabilities{CapabilityOffsetPage: {}, CapabilityTotalCount: {}, CapabilityLookup: {}}, lookupPage: LookupPage{Options: []LookupOption{{Value: int64(1), Label: "São Paulo"}}}}
	parent := validDefinition("fixed_lookup_contacts")
	parent.Source = parentSource
	parent.Fields = append(parent.Fields, Field{Key: "state_id", Label: "state", Type: FieldLookup, Visible: true, Lookup: &LookupDefinition{Resource: "fixed_reference_values", ValueField: "id", LabelField: "name", PageSize: 25, FixedFilters: []FixedLookupFilter{{Field: "kind", Values: []Value{"state", "territory"}}}}})
	target := validDefinition("fixed_reference_values")
	target.Scope = ScopeRequirements{Mode: ScopeModeGlobal}
	target.Permissions = Permissions{Read: "reference.read"}
	target.Delete = DeletePolicy{Mode: DeleteModeNone}
	target.Concurrency = ConcurrencyPolicy{Mode: ConcurrencyNone}
	target.UOW = nil
	target.Source = targetSource
	registry := NewRegistry()
	for _, definition := range []Definition{parent, target} {
		if err := registry.Register(context.Background(), definition); err != nil {
			t.Fatal(err)
		}
	}
	service, err := NewService(Dependencies{Registry: registry, Principal: principalStub{}, Scope: scopeStub{scope: Scope{"tenant_id": "tenant-a"}}, Authorizer: &recordingAuthorizer{}, Audit: auditStub{}, Translator: translatorStub{}, Clock: clockStub{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Lookup(context.Background(), parent.Key, "state_id", LookupQuery{FixedFilters: []FixedLookupFilter{{Field: "kind", Values: []Value{"currency"}}}}); err != nil {
		t.Fatal(err)
	}
	filters := targetSource.lookupQuery.FixedFilters
	if targetSource.lookupCalls != 1 || len(filters) != 1 || filters[0].Field != "kind" || len(filters[0].Values) != 2 || filters[0].Values[0] != "state" || filters[0].Values[1] != "territory" {
		t.Fatalf("fixed filters were not server-owned: %#v", targetSource.lookupQuery)
	}
	if _, err := service.Lookup(context.Background(), parent.Key, "state_id", LookupQuery{}); err != nil {
		t.Fatal(err)
	}
	if targetSource.lookupCalls != 1 {
		t.Fatalf("fixed-filter lookup missed cache: %d calls", targetSource.lookupCalls)
	}
}

func TestPublicDefinitionDoesNotExposeFixedLookupFilters(t *testing.T) {
	definition := validDefinition("private_lookup_filters")
	definition.Fields = append(definition.Fields, Field{Key: "state_id", Label: "state", Type: FieldLookup, Visible: true, Lookup: &LookupDefinition{Resource: "reference_states", ValueField: "id", LabelField: "name", PageSize: 25, FixedFilters: []FixedLookupFilter{{Field: "kind", Values: []Value{"state"}}}}})
	state := readState{definition: definition, principal: Principal{ID: "operator"}}
	public := publicDefinition(context.Background(), state, &recordingAuthorizer{}, NewRegistry())
	field, ok := findField(public.Fields, "state_id")
	if !ok || field.Lookup == nil || len(field.Lookup.FixedFilters) != 0 {
		t.Fatalf("public lookup exposed fixed filters: %#v", field)
	}
}

type blockingLookupSource struct {
	*recordingSource
	start   chan struct{}
	release chan struct{}
	once    sync.Once
}

func (source *blockingLookupSource) Lookup(_ context.Context, _ Scope, query LookupQuery) (LookupPage, error) {
	source.lookupCalls++
	source.lookupQuery = query
	source.once.Do(func() { close(source.start) })
	<-source.release
	return source.lookupPage, source.lookupErr
}

func TestServiceLookupCoalescesConcurrentCacheMisses(t *testing.T) {
	parentSource := &recordingSource{capabilities: capabilitiesWithLookup()}
	targetSource := &blockingLookupSource{recordingSource: &recordingSource{capabilities: Capabilities{CapabilityOffsetPage: {}, CapabilityTotalCount: {}, CapabilityLookup: {}}, lookupPage: LookupPage{Options: []LookupOption{{Value: int64(1), Label: "São Paulo"}}}}, start: make(chan struct{}), release: make(chan struct{})}
	parent := validDefinition("coalesced_contacts")
	parent.Source = parentSource
	parent.Fields = append(parent.Fields, Field{Key: "state_id", Label: "state", Type: FieldLookup, Visible: true, Lookup: &LookupDefinition{Resource: "coalesced_states", ValueField: "id", LabelField: "name", PageSize: 25}})
	target := validDefinition("coalesced_states")
	target.Scope = ScopeRequirements{Mode: ScopeModeGlobal}
	target.Permissions = Permissions{Read: "reference.read"}
	target.Delete = DeletePolicy{Mode: DeleteModeNone}
	target.Concurrency = ConcurrencyPolicy{Mode: ConcurrencyNone}
	target.UOW = nil
	target.Source = targetSource
	registry := NewRegistry()
	if err := registry.Register(context.Background(), parent); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(Dependencies{Registry: registry, Principal: principalStub{}, Scope: scopeStub{scope: Scope{"tenant_id": "tenant-a"}}, Authorizer: &recordingAuthorizer{}, Audit: auditStub{}, Translator: translatorStub{}, Clock: clockStub{}})
	if err != nil {
		t.Fatal(err)
	}
	query := LookupQuery{}
	first := make(chan error, 1)
	go func() {
		_, callErr := service.Lookup(context.Background(), parent.Key, "state_id", query)
		first <- callErr
	}()
	<-targetSource.start
	second := make(chan error, 1)
	go func() {
		_, callErr := service.Lookup(context.Background(), parent.Key, "state_id", query)
		second <- callErr
	}()
	for index := 0; index < 100; index++ {
		runtime.Gosched()
	}
	close(targetSource.release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	if targetSource.lookupCalls != 1 {
		t.Fatalf("coalesced lookup calls = %d, want 1", targetSource.lookupCalls)
	}
}
