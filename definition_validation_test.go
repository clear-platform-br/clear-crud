package crud

import (
	"context"
	"errors"
	"testing"
)

func TestValidateDefinitionCoversStartupBoundaries(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		change   func(*Definition)
		wantPath string
	}{
		"missing title":     {func(definition *Definition) { definition.Labels.Title = "" }, "labels.title"},
		"missing singular":  {func(definition *Definition) { definition.Labels.Singular = "" }, "labels.singular"},
		"missing source":    {func(definition *Definition) { definition.Source = nil }, "source"},
		"typed nil source":  {func(definition *Definition) { var source *nilSource; definition.Source = source }, "source"},
		"invalid scope key": {func(definition *Definition) { definition.Scope.Keys = []string{"tenant-id"} }, "scope.keys[0]"},
		"too long resource key": {func(definition *Definition) {
			definition.Key = "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm"
		}, "key"},
		"missing field label": {func(definition *Definition) { definition.Fields[0].Label = "" }, "fields[0].label"},
		"invalid field key":   {func(definition *Definition) { definition.Fields[0].Key = "not-a-key" }, "fields[0].key"},
		"enum label missing": {func(definition *Definition) {
			definition.Fields[0].Type = FieldEnum
			definition.Fields[0].Enum = []Option{{Value: "one"}}
		}, "fields[0].enum[0].label"},
		"enum on string":                     {func(definition *Definition) { definition.Fields[0].Enum = []Option{{Value: "one", Label: "crud.one"}} }, "fields[0].enum"},
		"lookup missing":                     {func(definition *Definition) { definition.Fields[0].Type = FieldLookup }, "fields[0].lookup"},
		"lookup invalid resource":            {appendLookup("bad key", "id", "name", nil, 25), "fields[2].lookup.resource"},
		"lookup missing result fields":       {appendLookup("categories", "", "", nil, 25), "fields[2].lookup"},
		"lookup invalid page size":           {appendLookup("categories", "id", "name", nil, 101), "fields[2].lookup.pageSize"},
		"lookup cannot depend on itself":     {appendLookup("categories", "id", "name", []FieldKey{"category"}, 25), "fields[2].lookup.dependencies[0]"},
		"lookup dependencies must be unique": {appendLookup("categories", "id", "name", []FieldKey{"active", "active"}, 25), "fields[2].lookup.dependencies"},
		"duplicate list column":              {func(definition *Definition) { definition.List.Columns = []FieldKey{"name", "name"} }, "list.columns"},
		"default sort must be sortable": {func(definition *Definition) {
			definition.List.DefaultSort = []Sort{{Field: "active", Direction: SortAscending}}
			definition.List.Sortable = []FieldKey{"name"}
		}, "list.defaultSort[0].field"},
		"invalid sort direction":          {func(definition *Definition) { definition.List.DefaultSort[0].Direction = "sideways" }, "list.defaultSort[0].direction"},
		"invalid page mode":               {func(definition *Definition) { definition.List.Pagination.Mode = "page" }, "list.pagination.mode"},
		"default page size remains fixed": {func(definition *Definition) { definition.List.Pagination.DefaultSize = 50 }, "list.pagination.defaultSize"},
		"invalid density":                 {func(definition *Definition) { definition.Presentation.Density = "dense" }, "presentation.density"},
		"invalid delete mode":             {func(definition *Definition) { definition.Delete.Mode = "soft" }, "delete.mode"},
		"delete mode requires permission": {func(definition *Definition) { definition.Permissions.Delete = "" }, "permissions.delete"},
		"offset capability required":      {func(definition *Definition) { definition.Source = fakeSource{capabilities: Capabilities{}} }, "source.capabilities"},
		"total capability required": {func(definition *Definition) {
			definition.Source = fakeSource{capabilities: Capabilities{CapabilityOffsetPage: {}, CapabilityAtomicVersion: {}, CapabilityUnitOfWork: {}, CapabilityArchive: {}}}
		}, "source.capabilities"},
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
				t.Fatalf("definition error path = %q, want %q", definitionError.Path, test.wantPath)
			}
		})
	}
}

func TestValidateDefinitionSupportsCursorReadOnlyResources(t *testing.T) {
	t.Parallel()
	definition := validDefinition("read_only_categories")
	definition.Permissions = Permissions{Read: "crud.categories.read"}
	definition.Delete.Mode = DeleteModeNone
	definition.Concurrency.Mode = ConcurrencyNone
	definition.UOW = nil
	definition.List.Pagination = PaginationDefinition{Mode: PageModeCursor, DefaultSize: 25, AllowedSizes: []uint16{25, 50, 100}}
	definition.Source = fakeSource{capabilities: Capabilities{CapabilityCursorPage: {}}}
	if err := ValidateDefinition(context.Background(), definition); err != nil {
		t.Fatalf("ValidateDefinition() error = %v", err)
	}
}

func TestDefinitionErrorHandlesNilReceiver(t *testing.T) {
	t.Parallel()
	var err *DefinitionError
	if got := err.Error(); got != "" {
		t.Fatalf("nil DefinitionError Error() = %q", got)
	}
}

func appendLookup(resource ResourceKey, value, label FieldKey, dependencies []FieldKey, size uint16) func(*Definition) {
	return func(definition *Definition) {
		definition.Fields = append(definition.Fields, Field{Key: "category", Label: "crud.category", Type: FieldLookup, Visible: true, Lookup: &LookupDefinition{Resource: resource, ValueField: value, LabelField: label, Dependencies: dependencies, PageSize: size}})
	}
}

type nilSource struct{}

func (*nilSource) Capabilities(context.Context) Capabilities               { return nil }
func (*nilSource) List(context.Context, Scope, Query) (Page, error)        { return Page{}, nil }
func (*nilSource) Get(context.Context, Scope, RecordID) (Record, error)    { return Record{}, nil }
func (*nilSource) Create(context.Context, Scope, Mutation) (Record, error) { return Record{}, nil }
func (*nilSource) Update(context.Context, Scope, RecordID, Version, Mutation) (Record, error) {
	return Record{}, nil
}
func (*nilSource) Delete(context.Context, Scope, RecordID, Version, DeleteMode) error { return nil }
func (*nilSource) Lookup(context.Context, Scope, LookupQuery) (LookupPage, error) {
	return LookupPage{}, nil
}
