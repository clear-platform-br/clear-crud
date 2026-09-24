package crud

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestRegistryRegisterGetAndSeal(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	definition := validDefinition("contact_categories")
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	definition.Labels.Title = "changed"
	definition.Fields[0].Label = "changed"
	definition.Scope.Keys[0] = "changed"
	definition.List.Columns[0] = "changed"
	definition.List.Pagination.AllowedSizes[0] = 1

	stored, ok := registry.Get("contact_categories")
	if !ok {
		t.Fatal("Get() did not find registered definition")
	}
	if stored.Labels.Title != "crud.contact_categories.title" {
		t.Fatalf("stored title = %q", stored.Labels.Title)
	}
	if stored.Fields[0].Label != "crud.contact_categories.name" {
		t.Fatalf("stored field label = %q", stored.Fields[0].Label)
	}
	if stored.Scope.Keys[0] != "tenant_id" {
		t.Fatalf("stored scope key = %q", stored.Scope.Keys[0])
	}
	if stored.List.Columns[0] != "name" {
		t.Fatalf("stored list column = %q", stored.List.Columns[0])
	}
	if stored.List.Pagination.AllowedSizes[0] != 25 {
		t.Fatalf("stored page size = %d", stored.List.Pagination.AllowedSizes[0])
	}

	stored.Fields[0].Label = "mutated-return-value"
	again, ok := registry.Get("contact_categories")
	if !ok || again.Fields[0].Label != "crud.contact_categories.name" {
		t.Fatal("Get() must return a defensive copy")
	}

	registry.Seal()
	if !registry.Sealed() {
		t.Fatal("registry must report sealed after Seal")
	}
	if err := registry.Register(context.Background(), validDefinition("other_categories")); !errors.Is(err, ErrRegistrySealed) {
		t.Fatalf("Register() after Seal error = %v, want ErrRegistrySealed", err)
	}
}

func TestRegistryRejectsDuplicateResource(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	definition := validDefinition("contact_categories")
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if err := registry.Register(context.Background(), definition); !errors.Is(err, ErrDuplicateResource) {
		t.Fatalf("second Register() error = %v, want ErrDuplicateResource", err)
	}
}

func TestRegistryConcurrentRegistration(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	const resources = 64
	errorsByResource := make(chan error, resources)
	var group sync.WaitGroup
	for index := 0; index < resources; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			errorsByResource <- registry.Register(context.Background(), validDefinition(ResourceKey(fmt.Sprintf("resource_%d", index))))
		}(index)
	}
	group.Wait()
	close(errorsByResource)
	for err := range errorsByResource {
		if err != nil {
			t.Fatalf("concurrent Register() error = %v", err)
		}
	}
	if got := len(registry.Keys()); got != resources {
		t.Fatalf("registered resources = %d, want %d", got, resources)
	}
}

func TestValidateDefinitionRejectsInvalidInvariants(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		change   func(*Definition)
		wantPath string
	}{
		"wrong contract": {
			change:   func(definition *Definition) { definition.Contract = "wrong" },
			wantPath: "contract",
		},
		"invalid resource key": {
			change:   func(definition *Definition) { definition.Key = "Contact Categories" },
			wantPath: "key",
		},
		"read permission missing": {
			change:   func(definition *Definition) { definition.Permissions.Read = "" },
			wantPath: "permissions.read",
		},
		"unknown field type": {
			change:   func(definition *Definition) { definition.Fields[0].Type = "unknown" },
			wantPath: "fields[0].type",
		},
		"duplicate field": {
			change:   func(definition *Definition) { definition.Fields = append(definition.Fields, definition.Fields[0]) },
			wantPath: "fields",
		},
		"unknown list column": {
			change:   func(definition *Definition) { definition.List.Columns = []FieldKey{"missing"} },
			wantPath: "list.columns[0]",
		},
		"unstable default sort": {
			change:   func(definition *Definition) { definition.List.DefaultSort = nil },
			wantPath: "list.defaultSort",
		},
		"invalid page size": {
			change:   func(definition *Definition) { definition.List.Pagination.AllowedSizes = []uint16{25, 99, 100} },
			wantPath: "list.pagination.allowedSizes",
		},
		"mutable without unit of work": {
			change:   func(definition *Definition) { definition.UOW = nil },
			wantPath: "uow",
		},
		"archive without capability": {
			change:   func(definition *Definition) { definition.Source = fakeSource{capabilities: baseCapabilities()} },
			wantPath: "source.capabilities",
		},
		"lookup missing dependency": {
			change: func(definition *Definition) {
				definition.Fields = append(definition.Fields, Field{
					Key:     "category",
					Label:   "crud.contact_categories.category",
					Type:    FieldLookup,
					Visible: true,
					Lookup: &LookupDefinition{
						Resource:     "categories",
						ValueField:   "id",
						LabelField:   "name",
						Dependencies: []FieldKey{"missing"},
						PageSize:     25,
					},
				})
			},
			wantPath: "fields[2].lookup.dependencies[0]",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			definition := validDefinition("contact_categories")
			test.change(&definition)
			err := ValidateDefinition(context.Background(), definition)
			var definitionError *DefinitionError
			if !errors.As(err, &definitionError) {
				t.Fatalf("ValidateDefinition() error = %v, want DefinitionError", err)
			}
			if definitionError.Path != test.wantPath {
				t.Fatalf("error path = %q, want %q", definitionError.Path, test.wantPath)
			}
		})
	}
}

func TestRegistryKeysAreSorted(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	for _, key := range []ResourceKey{"zebra", "alpha", "middle"} {
		if err := registry.Register(context.Background(), validDefinition(key)); err != nil {
			t.Fatalf("Register(%q) error = %v", key, err)
		}
	}
	want := []ResourceKey{"alpha", "middle", "zebra"}
	got := registry.Keys()
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("Keys() = %v, want %v", got, want)
		}
	}
}

func TestNilRegistryIsSafeForReads(t *testing.T) {
	t.Parallel()

	var registry *Registry
	if _, ok := registry.Get("missing"); ok {
		t.Fatal("nil registry must not return a definition")
	}
	if keys := registry.Keys(); keys != nil {
		t.Fatalf("nil registry keys = %v, want nil", keys)
	}
	if !registry.Sealed() {
		t.Fatal("nil registry must be treated as sealed")
	}
	if err := registry.Register(context.Background(), validDefinition("missing")); err == nil {
		t.Fatal("nil registry registration must fail")
	}
}

func TestDefinitionErrorIncludesPathForStartupDiagnostics(t *testing.T) {
	t.Parallel()

	err := &DefinitionError{Path: "fields[0].type", Reason: "is unknown"}
	if got, want := err.Error(), "invalid CRUD definition at fields[0].type: is unknown"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestValidateDefinitionAdditionalInvariants(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		change   func(*Definition)
		wantPath string
	}{
		"duplicate scope": {
			change:   func(definition *Definition) { definition.Scope.Keys = []string{"tenant_id", "tenant_id"} },
			wantPath: "scope.keys",
		},
		"no visible fields": {
			change: func(definition *Definition) {
				for index := range definition.Fields {
					definition.Fields[index].Visible = false
				}
			},
			wantPath: "fields",
		},
		"invalid field length": {
			change:   func(definition *Definition) { definition.Fields[0].MinLength, definition.Fields[0].MaxLength = 10, 2 },
			wantPath: "fields[0].length",
		},
		"enum needs options": {
			change: func(definition *Definition) {
				definition.Fields[0].Type = FieldEnum
				definition.Fields[0].Enum = nil
			},
			wantPath: "fields[0].enum",
		},
		"enum option type": {
			change: func(definition *Definition) {
				definition.Fields[0].Type = FieldEnum
				definition.Fields[0].Enum = []Option{{Value: 1, Label: "crud.one"}}
			},
			wantPath: "fields[0].enum[0].value",
		},
		"enum duplicate option": {
			change: func(definition *Definition) {
				definition.Fields[0].Type = FieldEnum
				definition.Fields[0].Enum = []Option{{Value: "one", Label: "crud.one"}, {Value: "one", Label: "crud.other"}}
			},
			wantPath: "fields[0].enum",
		},
		"lookup only for lookup field": {
			change: func(definition *Definition) {
				definition.Fields[0].Lookup = &LookupDefinition{Resource: "categories", ValueField: "id", LabelField: "name", PageSize: 25}
			},
			wantPath: "fields[0].lookup",
		},
		"lookup requires capability": {
			change: func(definition *Definition) {
				definition.Fields = append(definition.Fields, lookupField())
				definition.Source = fakeSource{capabilities: mutableCapabilities()}
			},
			wantPath: "source.capabilities",
		},
		"invalid presentation": {
			change:   func(definition *Definition) { definition.Presentation.Collection = "grid" },
			wantPath: "presentation.collection",
		},
		"delete permission without mode": {
			change: func(definition *Definition) {
				definition.Delete.Mode = DeleteModeNone
			},
			wantPath: "permissions.delete",
		},
		"hard delete capability missing": {
			change: func(definition *Definition) {
				definition.Delete.Mode = DeleteModeHardDelete
				definition.Source = fakeSource{capabilities: mutableCapabilities()}
			},
			wantPath: "source.capabilities",
		},
		"cursor capability missing": {
			change: func(definition *Definition) {
				definition.List.Pagination.Mode = PageModeCursor
				definition.Source = fakeSource{capabilities: mutableCapabilities()}
			},
			wantPath: "source.capabilities",
		},
		"mutable requires version policy": {
			change:   func(definition *Definition) { definition.Concurrency.Mode = ConcurrencyNone },
			wantPath: "concurrency.mode",
		},
		"read only does not require write guarantees": {
			change: func(definition *Definition) {
				definition.Permissions.Create = ""
				definition.Permissions.Update = ""
				definition.Permissions.Delete = ""
				definition.Delete.Mode = DeleteModeNone
				definition.Concurrency.Mode = ConcurrencyNone
				definition.UOW = nil
				definition.Source = fakeSource{capabilities: Capabilities{CapabilityOffsetPage: {}, CapabilityTotalCount: {}}}
			},
			wantPath: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			definition := validDefinition("contact_categories")
			test.change(&definition)
			err := ValidateDefinition(context.Background(), definition)
			if test.wantPath == "" {
				if err != nil {
					t.Fatalf("ValidateDefinition() error = %v", err)
				}
				return
			}
			var definitionError *DefinitionError
			if !errors.As(err, &definitionError) {
				t.Fatalf("ValidateDefinition() error = %v, want DefinitionError", err)
			}
			if definitionError.Path != test.wantPath {
				t.Fatalf("error path = %q, want %q", definitionError.Path, test.wantPath)
			}
		})
	}
}

func validDefinition(key ResourceKey) Definition {
	return Definition{
		Contract: ContractDefinitionV1,
		Key:      key,
		Labels: Labels{
			Title:    "crud.contact_categories.title",
			Singular: "crud.contact_categories.singular",
		},
		Scope: ScopeRequirements{Keys: []string{"tenant_id"}},
		Permissions: Permissions{
			Create: "crud.contact_categories.create",
			Read:   "crud.contact_categories.read",
			Update: "crud.contact_categories.update",
			Delete: "crud.contact_categories.delete",
		},
		Fields: []Field{
			{Key: "name", Label: "crud.contact_categories.name", Type: FieldString, Required: true, Visible: true, MaxLength: 80},
			{Key: "active", Label: "crud.contact_categories.active", Type: FieldBoolean, Visible: true},
		},
		List: ListDefinition{
			Columns:     []FieldKey{"name", "active"},
			Searchable:  []FieldKey{"name"},
			Sortable:    []FieldKey{"name", "active"},
			DefaultSort: []Sort{{Field: "name", Direction: SortAscending}},
			Pagination: PaginationDefinition{
				Mode:         PageModeOffset,
				DefaultSize:  25,
				AllowedSizes: []uint16{25, 50, 100},
				Total:        true,
			},
		},
		Presentation: Presentation{Collection: CollectionAuto, Density: DensityComfortable},
		Source:       fakeSource{capabilities: mutableCapabilities()},
		UOW:          fakeUnitOfWork{},
		Delete:       DeletePolicy{Mode: DeleteModeArchive},
		Concurrency:  ConcurrencyPolicy{Mode: ConcurrencyVersion},
	}
}

func baseCapabilities() Capabilities {
	return Capabilities{
		CapabilityOffsetPage:    {},
		CapabilityTotalCount:    {},
		CapabilityAtomicVersion: {},
		CapabilityUnitOfWork:    {},
	}
}

func mutableCapabilities() Capabilities {
	capabilities := baseCapabilities()
	capabilities[CapabilityArchive] = struct{}{}
	return capabilities
}

func lookupField() Field {
	return Field{
		Key:     "category",
		Label:   "crud.contact_categories.category",
		Type:    FieldLookup,
		Visible: true,
		Lookup: &LookupDefinition{
			Resource:     "categories",
			ValueField:   "id",
			LabelField:   "name",
			Dependencies: []FieldKey{"active"},
			PageSize:     25,
		},
	}
}

type fakeSource struct {
	capabilities Capabilities
}

func (source fakeSource) Capabilities(context.Context) Capabilities        { return source.capabilities }
func (fakeSource) List(context.Context, Scope, Query) (Page, error)        { return Page{}, nil }
func (fakeSource) Get(context.Context, Scope, RecordID) (Record, error)    { return Record{}, nil }
func (fakeSource) Create(context.Context, Scope, Mutation) (Record, error) { return Record{}, nil }
func (fakeSource) Update(context.Context, Scope, RecordID, Version, Mutation) (Record, error) {
	return Record{}, nil
}
func (fakeSource) Delete(context.Context, Scope, RecordID, Version, DeleteMode) error { return nil }
func (fakeSource) Lookup(context.Context, Scope, LookupQuery) (LookupPage, error) {
	return LookupPage{}, nil
}

type fakeUnitOfWork struct{}

func (fakeUnitOfWork) Within(ctx context.Context, operation func(context.Context) error) error {
	return operation(ctx)
}
