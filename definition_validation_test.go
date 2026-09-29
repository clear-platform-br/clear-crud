package crud

import (
	"context"
	"errors"
	"strings"
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
		"enum control unknown": {func(definition *Definition) {
			definition.Fields[0].Type = FieldEnum
			definition.Fields[0].Enum = []Option{{Value: "one", Label: "crud.one"}}
			definition.Fields[0].EnumControl = "chips"
		}, "fields[0].enumControl"},
		"enum control on string": {func(definition *Definition) {
			definition.Fields[0].EnumControl = EnumControlSelect
		}, "fields[0].enumControl"},
		"pattern invalid regex": {func(definition *Definition) {
			definition.Fields[0].Pattern = "["
		}, "fields[0].pattern"},
		"pattern message requires pattern": {func(definition *Definition) {
			definition.Fields[0].PatternMessage = "crud.contact.name.pattern"
		}, "fields[0].patternMessage"},
		"pattern requires string-backed field": {func(definition *Definition) {
			definition.Fields[1].Pattern = "^(yes|no)$"
		}, "fields[1].pattern"},
		"pattern maximum length": {func(definition *Definition) {
			definition.Fields[0].Pattern = strings.Repeat("a", 513)
		}, "fields[0].pattern"},
		"enum on string":               {func(definition *Definition) { definition.Fields[0].Enum = []Option{{Value: "one", Label: "crud.one"}} }, "fields[0].enum"},
		"lookup missing":               {func(definition *Definition) { definition.Fields[0].Type = FieldLookup }, "fields[0].lookup"},
		"lookup invalid resource":      {appendLookup("bad key", "id", "name", nil, 25), "fields[2].lookup.resource"},
		"lookup missing result fields": {appendLookup("categories", "", "", nil, 25), "fields[2].lookup"},
		"lookup invalid page size":     {appendLookup("categories", "id", "name", nil, 101), "fields[2].lookup.pageSize"},
		"lookup fixed filter invalid field": {func(definition *Definition) {
			appendLookup("categories", "id", "name", nil, 25)(definition)
			definition.Fields[len(definition.Fields)-1].Lookup.FixedFilters = []FixedLookupFilter{{Field: "bad-field", Values: []Value{"state"}}}
		}, "fields[2].lookup.fixedFilters[0].field"},
		"lookup fixed filter requires values": {func(definition *Definition) {
			appendLookup("categories", "id", "name", nil, 25)(definition)
			definition.Fields[len(definition.Fields)-1].Lookup.FixedFilters = []FixedLookupFilter{{Field: "kind"}}
		}, "fields[2].lookup.fixedFilters[0].values"},
		"lookup fixed filter nil value": {func(definition *Definition) {
			appendLookup("categories", "id", "name", nil, 25)(definition)
			definition.Fields[len(definition.Fields)-1].Lookup.FixedFilters = []FixedLookupFilter{{Field: "kind", Values: []Value{nil}}}
		}, "fields[2].lookup.fixedFilters[0].values[0]"},
		"lookup fixed filter duplicate values": {func(definition *Definition) {
			appendLookup("categories", "id", "name", nil, 25)(definition)
			definition.Fields[len(definition.Fields)-1].Lookup.FixedFilters = []FixedLookupFilter{{Field: "kind", Values: []Value{"state", "state"}}}
		}, "fields[2].lookup.fixedFilters[0].values"},
		"lookup fixed filters must be unique": {func(definition *Definition) {
			appendLookup("categories", "id", "name", nil, 25)(definition)
			definition.Fields[len(definition.Fields)-1].Lookup.FixedFilters = []FixedLookupFilter{{Field: "kind", Values: []Value{"state"}}, {Field: "kind", Values: []Value{"currency"}}}
		}, "fields[2].lookup.fixedFilters"},
		"lookup search minimum too large": {func(definition *Definition) {
			appendLookup("categories", "id", "name", nil, 25)(definition)
			definition.Fields[len(definition.Fields)-1].Lookup.MinSearchLength = 65
		}, "fields[2].lookup.minSearchLength"},
		"lookup cannot depend on itself":     {appendLookup("categories", "id", "name", []FieldKey{"category"}, 25), "fields[2].lookup.dependencies[0]"},
		"lookup dependencies must be unique": {appendLookup("categories", "id", "name", []FieldKey{"active", "active"}, 25), "fields[2].lookup.dependencies"},
		"duplicate grid column":              {func(definition *Definition) { definition.Grid.Columns = []FieldKey{"name", "name"} }, "grid.columns"},
		"duplicate form field":               {func(definition *Definition) { definition.Form.Fields = []FieldKey{"name", "name"} }, "form.fields"},
		"form field must be visible": {func(definition *Definition) {
			definition.Fields[1].Visible = false
			definition.Form.Fields = []FieldKey{"active"}
		}, "form.fields[0]"},
		"default sort must be sortable": {func(definition *Definition) {
			definition.Grid.DefaultSort = []Sort{{Field: "active", Direction: SortAscending}}
			definition.Grid.Sortable = []FieldKey{"name"}
		}, "grid.defaultSort[0].field"},
		"invalid sort direction":     {func(definition *Definition) { definition.Grid.DefaultSort[0].Direction = "sideways" }, "grid.defaultSort[0].direction"},
		"invalid archive visibility": {func(definition *Definition) { definition.Grid.ArchiveVisibility = "all" }, "grid.archiveVisibility"},
		"boolean display on text": {func(definition *Definition) {
			definition.Fields[0].BooleanDisplay = &BooleanDisplay{True: "✓", False: "×"}
		}, "fields[0].booleanDisplay"},
		"boolean display needs both symbols": {func(definition *Definition) { definition.Fields[1].BooleanDisplay = &BooleanDisplay{True: "✓"} }, "fields[1].booleanDisplay"},
		"invalid page mode":                  {func(definition *Definition) { definition.Grid.Pagination.Mode = "page" }, "grid.pagination.mode"},
		"default page size must be allowed":  {func(definition *Definition) { definition.Grid.Pagination.DefaultSize = 10 }, "grid.pagination.defaultSize"},
		"invalid density":                    {func(definition *Definition) { definition.Presentation.Density = "dense" }, "presentation.density"},
		"title field must exist":             {func(definition *Definition) { definition.Presentation.TitleField = "missing" }, "presentation.title_field"},
		"title field must be visible": {func(definition *Definition) {
			definition.Fields[0].Visible = false
			definition.Presentation.TitleField = "name"
		}, "presentation.title_field"},
		"title field must not be sensitive": {func(definition *Definition) {
			definition.Fields[0].Sensitive = true
			definition.Presentation.TitleField = "name"
		}, "presentation.title_field"},
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
	definition.Grid.Pagination = PaginationDefinition{Mode: PageModeCursor, DefaultSize: 25, AllowedSizes: []uint16{25, 50, 100}}
	definition.Source = fakeSource{capabilities: Capabilities{CapabilityCursorPage: {}}}
	if err := ValidateDefinition(context.Background(), definition); err != nil {
		t.Fatalf("ValidateDefinition() error = %v", err)
	}
}

func TestValidateDefinitionAllowsCustomGridPageSize(t *testing.T) {
	t.Parallel()
	definition := validDefinition("small_page_categories")
	definition.Grid.Pagination.DefaultSize = 10
	definition.Grid.Pagination.AllowedSizes = []uint16{10, 25, 50, 100}
	if err := ValidateDefinition(context.Background(), definition); err != nil {
		t.Fatalf("ValidateDefinition() error = %v", err)
	}
}

func TestValidateDefinitionRejectsUnsafeGridPageSizes(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		change   func(*Definition)
		wantPath string
	}{
		"zero default":      {func(definition *Definition) { definition.Grid.Pagination.DefaultSize = 0 }, "grid.pagination.defaultSize"},
		"oversized default": {func(definition *Definition) { definition.Grid.Pagination.DefaultSize = 101 }, "grid.pagination.defaultSize"},
		"missing allowlist": {func(definition *Definition) { definition.Grid.Pagination.AllowedSizes = nil }, "grid.pagination.allowedSizes"},
		"too many allowed sizes": {func(definition *Definition) {
			definition.Grid.Pagination.AllowedSizes = []uint16{1, 2, 3, 4, 5, 6, 7, 8, 9}
		}, "grid.pagination.allowedSizes"},
		"zero allowed size":    {func(definition *Definition) { definition.Grid.Pagination.AllowedSizes = []uint16{0, 25} }, "grid.pagination.allowedSizes[0]"},
		"descending allowlist": {func(definition *Definition) { definition.Grid.Pagination.AllowedSizes = []uint16{25, 10} }, "grid.pagination.allowedSizes"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			definition := validDefinition("bounded_categories")
			test.change(&definition)
			err := ValidateDefinition(context.Background(), definition)
			var definitionError *DefinitionError
			if !errors.As(err, &definitionError) || definitionError.Path != test.wantPath {
				t.Fatalf("validation error = %v, want path %q", err, test.wantPath)
			}
		})
	}
}

func TestValidateDefinitionSupportsExplicitFormProjection(t *testing.T) {
	t.Parallel()
	definition := validDefinition("form_categories")
	definition.Form.Fields = []FieldKey{"active", "name"}
	if err := ValidateDefinition(context.Background(), definition); err != nil {
		t.Fatalf("ValidateDefinition() error = %v", err)
	}
	definition.Form.Fields = []FieldKey{"missing"}
	err := ValidateDefinition(context.Background(), definition)
	var definitionError *DefinitionError
	if !errors.As(err, &definitionError) || definitionError.Path != "form.fields[0]" {
		t.Fatalf("form projection error = %v, want form.fields[0]", err)
	}
}

func TestValidateDefinitionAllowsVisibleNonSensitiveTitleField(t *testing.T) {
	t.Parallel()
	definition := validDefinition("contextual_categories")
	definition.Presentation.TitleField = "name"
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
