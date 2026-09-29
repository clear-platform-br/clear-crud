package sqladapter

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	crud "github.com/clear-platform-br/clear-crud"
	_ "modernc.org/sqlite"
)

func TestAutoTableBuildsConventionalSQLiteDefinition(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name VARCHAR(80) NOT NULL,
        notes TEXT, enabled BOOLEAN NOT NULL, balance DECIMAL(10,2),
        birthday DATE, updated_at DATETIME, version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	definition, err := AutoTable(context.Background(), database, AutoTableConfig{
		Table: mustIdentifier("contacts"), ArchiveColumn: identifierPointer(mustIdentifier("archived")), ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")},
		Permissions: crud.Permissions{Read: "contacts.read", Create: "contacts.create", Update: "contacts.update", Delete: "contacts.delete"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if definition.Key != "contacts" || definition.Delete.Mode != crud.DeleteModeArchive || definition.Concurrency.Mode != crud.ConcurrencyVersion {
		t.Fatalf("definition defaults = %#v", definition)
	}
	if got := definition.Grid.Pagination; got.Mode != crud.PageModeOffset || got.DefaultSize != 25 || !got.Total {
		t.Fatalf("pagination = %#v", got)
	}
	if got := fieldByKey(t, definition.Fields, "name"); got.Type != crud.FieldString || got.MaxLength != 80 || !got.IsRequired() {
		t.Fatalf("name field = %#v", got)
	}
	if got := fieldByKey(t, definition.Fields, "notes"); got.MaxLength != defaultAutoTextMaxLength {
		t.Fatalf("notes maximum = %d, want %d", got.MaxLength, defaultAutoTextMaxLength)
	}
	wantFieldOrder := []crud.FieldKey{"name", "notes", "enabled", "balance", "birthday", "updated_at"}
	for index, want := range wantFieldOrder {
		if got := definition.Fields[index].Key; got != want {
			t.Fatalf("field order[%d] = %q, want %q", index, got, want)
		}
	}
	if !fieldByKey(t, definition.Fields, "notes").IsRequired() {
		t.Fatal("notes field should be required by default")
	}
	for key, want := range map[crud.FieldKey]crud.FieldType{"notes": crud.FieldString, "enabled": crud.FieldBoolean, "balance": crud.FieldDecimal, "birthday": crud.FieldDate, "updated_at": crud.FieldDateTime} {
		if got := fieldByKey(t, definition.Fields, key); got.Type != want {
			t.Fatalf("%s type = %q, want %q", key, got.Type, want)
		}
	}
	if containsField(definition.Fields, "tenant_id") || containsField(definition.Fields, "id") || containsField(definition.Fields, "version") || containsField(definition.Fields, "archived") {
		t.Fatalf("technical field exposed: %#v", definition.Fields)
	}
	if !containsKey(definition.Grid.Searchable, "name") || !containsKey(definition.Grid.Searchable, "notes") || containsKey(definition.Grid.Searchable, "balance") {
		t.Fatalf("searchable = %#v", definition.Grid.Searchable)
	}
	if err := crud.ValidateDefinition(context.Background(), definition); err != nil {
		t.Fatalf("invalid generated definition: %v", err)
	}
	source := definition.Source.(*SimpleTable)
	scope := crud.Scope{"tenant_id": "tenant-a"}
	record, err := source.Create(context.Background(), scope, crud.Mutation{Fields: crud.Fields{"name": "Ana", "notes": "note", "enabled": true, "balance": "1.00", "birthday": "2026-09-26", "updated_at": "2026-09-26T12:00:00Z"}})
	if err != nil {
		t.Fatalf("create through auto table: %v", err)
	}
	if err := source.Delete(context.Background(), scope, record.ID, record.Version, crud.DeleteModeArchive); err != nil {
		t.Fatalf("archive through auto table: %v", err)
	}
	_, err = source.Get(context.Background(), scope, record.ID)
	requireCode(t, err, crud.ErrorNotFound)
	page, err := source.List(context.Background(), scope, crud.Query{Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 25}})
	if err != nil || len(page.Records) != 0 || page.Total == nil || *page.Total != 0 {
		t.Fatalf("soft-deleted record was visible in normal list: %#v, %v", page, err)
	}
}

func TestSimpleTableBooleanInFilterAcceptsBothStates(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, enabled BOOLEAN NOT NULL, version INTEGER NOT NULL
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"))
	if err != nil {
		t.Fatal(err)
	}
	source := definition.Source.(*SimpleTable)
	scope := crud.Scope{"tenant_id": "tenant-a"}
	for _, enabled := range []bool{true, false} {
		if _, err := source.Create(context.Background(), scope, crud.Mutation{Fields: crud.Fields{"enabled": enabled}}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := source.List(context.Background(), scope, crud.Query{Filters: []crud.Filter{{Field: "enabled", Operator: crud.FilterIn, Values: []crud.Value{true, false}}}, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}})
	if err != nil || len(page.Records) != 2 {
		t.Fatalf("boolean IN filter = %#v, %v", page, err)
	}
}

func TestWithArchiveVisibilityValidatesAndStoresTheContract(t *testing.T) {
	config := AutoTableConfig{}
	if err := WithArchiveVisibility(crud.ArchiveVisibilityActiveAndArchived)(&config); err != nil {
		t.Fatalf("WithArchiveVisibility(active_and_archived): %v", err)
	}
	if config.ArchiveVisibility != crud.ArchiveVisibilityActiveAndArchived {
		t.Fatalf("archive visibility = %q, want active_and_archived", config.ArchiveVisibility)
	}
	if err := WithArchiveVisibility(crud.ArchiveVisibilityActiveOnly)(&config); err != nil {
		t.Fatalf("WithArchiveVisibility(active_only): %v", err)
	}
	if config.ArchiveVisibility != crud.ArchiveVisibilityActiveOnly {
		t.Fatalf("archive visibility = %q, want active_only", config.ArchiveVisibility)
	}
	if err := WithArchiveVisibility("unknown")(&config); err == nil {
		t.Fatal("unknown archive visibility accepted")
	}
}

func TestAutoTablePreservesPhysicalColumnOrderWhenNoProjectionIsDeclared(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, zulu TEXT, alpha TEXT,
        version INTEGER NOT NULL, middle TEXT
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"))
	if err != nil {
		t.Fatal(err)
	}
	wantFields := []crud.FieldKey{"zulu", "alpha", "middle"}
	wantGrid := []crud.FieldKey{"zulu", "alpha", "middle"}
	if len(definition.Fields) != len(wantFields) || len(definition.Grid.Columns) != len(wantGrid) {
		t.Fatalf("generated projection = fields:%#v grid:%#v, want fields:%#v grid:%#v", definition.Fields, definition.Grid.Columns, wantFields, wantGrid)
	}
	for index, key := range wantFields {
		if definition.Fields[index].Key != key {
			t.Fatalf("physical field order[%d] = %q, want %q", index, definition.Fields[index].Key, key)
		}
	}
	for index, key := range wantGrid {
		if definition.Grid.Columns[index] != key {
			t.Fatalf("default grid order[%d] = %q, want %q", index, definition.Grid.Columns[index], key)
		}
	}
}

func TestAutoTableAppliesDeclarativeFieldPattern(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL,
        name TEXT NOT NULL
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithPattern("name", `^[A-Z].*$`, "crud.contact.name.pattern"))
	if err != nil {
		t.Fatalf("AutoTenantTable() error = %v", err)
	}
	field := fieldByKey(t, definition.Fields, "name")
	if field.Pattern != `^[A-Z].*$` || field.PatternMessage != "crud.contact.name.pattern" {
		t.Fatalf("pattern metadata = %#v", field)
	}
	if err := crud.ValidateDefinition(context.Background(), definition); err != nil {
		t.Fatalf("generated patterned definition is invalid: %v", err)
	}
}

func TestAutoTableReadsStaticSQLiteDefaultsAndAllowsExplicitOverride(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL,
        status TEXT NOT NULL DEFAULT 'new', enabled BOOLEAN NOT NULL DEFAULT 1,
        count INTEGER NOT NULL DEFAULT 7, amount DECIMAL NOT NULL DEFAULT 1.25,
        label TEXT DEFAULT 'O''Reilly', dynamic TEXT DEFAULT CURRENT_TIMESTAMP,
        version INTEGER NOT NULL
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithDefault("status", "review"))
	if err != nil {
		t.Fatal(err)
	}
	if got := fieldByKey(t, definition.Fields, "status").Default; got != "review" {
		t.Fatalf("explicit status default = %#v", got)
	}
	if got := fieldByKey(t, definition.Fields, "enabled").Default; got != true {
		t.Fatalf("boolean schema default = %#v", got)
	}
	if got := fieldByKey(t, definition.Fields, "count").Default; got != int64(7) {
		t.Fatalf("integer schema default = %#v", got)
	}
	if got := fieldByKey(t, definition.Fields, "amount").Default; got != "1.25" {
		t.Fatalf("decimal schema default = %#v", got)
	}
	if got := fieldByKey(t, definition.Fields, "label").Default; got != "O'Reilly" {
		t.Fatalf("quoted schema default = %#v", got)
	}
	if got := fieldByKey(t, definition.Fields, "dynamic").Default; got != nil {
		t.Fatalf("dynamic schema default was exposed: %#v", got)
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithDefault("status", 1)); err == nil {
		t.Fatal("non-scalar default accepted")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithDefault("missing", "x")); err == nil {
		t.Fatal("default for unknown field accepted")
	}
}

func TestAutoTableRejectsUnsafeAndAmbiguousSchema(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL, name TEXT)`)
	config := AutoTableConfig{Table: mustIdentifier("contacts"), ArchiveColumn: identifierPointer(mustIdentifier("archived")), ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")}, Permissions: crud.Permissions{Read: "contacts.read"}}
	for name, change := range map[string]func(*AutoTableConfig){
		"missing trusted scope":   func(c *AutoTableConfig) { c.ScopeColumns = nil },
		"missing read permission": func(c *AutoTableConfig) { c.Permissions.Read = "" },
		"missing scope column": func(c *AutoTableConfig) {
			c.ScopeColumns = map[string]Identifier{"tenant_id": mustIdentifier("missing")}
		},
		"archive needs permission": func(c *AutoTableConfig) { c.Permissions.Delete = "contacts.delete" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := config
			change(&candidate)
			if _, err := AutoTable(context.Background(), database, candidate); err == nil {
				t.Fatal("unsafe configuration accepted")
			}
		})
	}
	badID := newAutoDatabase(t, `CREATE TABLE bad_id (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL, name TEXT)`)
	if _, err := AutoTable(context.Background(), badID, AutoTableConfig{Table: mustIdentifier("bad_id"), ScopeColumns: config.ScopeColumns, Permissions: config.Permissions}); err == nil {
		t.Fatal("text primary id accepted")
	}
	unsupported := newAutoDatabase(t, `CREATE TABLE blobs (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL, payload BLOB)`)
	if _, err := AutoTable(context.Background(), unsupported, AutoTableConfig{Table: mustIdentifier("blobs"), ScopeColumns: config.ScopeColumns, Permissions: config.Permissions}); err == nil {
		t.Fatal("blob field accepted")
	}
}

func TestAutoTableDoesNotInferSecurityFromNames(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL, email TEXT)`)
	_, err := AutoTable(context.Background(), database, AutoTableConfig{Table: mustIdentifier("contacts"), Permissions: crud.Permissions{Read: "contacts.read"}})
	if err == nil || !strings.Contains(err.Error(), "trusted scope") {
		t.Fatalf("error = %v", err)
	}
}

func TestAutoTableLimitsGeneratedGridColumns(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL,
        field01 TEXT, field02 TEXT, field03 TEXT, field04 TEXT, field05 TEXT,
        field06 TEXT, field07 TEXT, field08 TEXT, field09 TEXT, field10 TEXT,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	definition, err := AutoTable(context.Background(), database, AutoTableConfig{
		Table: mustIdentifier("contacts"), ArchiveColumn: identifierPointer(mustIdentifier("archived")),
		ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")},
		Permissions:  crud.Permissions{Read: "contacts.read", Delete: "contacts.delete"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(definition.Fields) != 10 {
		t.Fatalf("generated fields = %d, want all 10", len(definition.Fields))
	}
	if len(definition.Grid.Columns) != defaultAutoGridColumns {
		t.Fatalf("default grid columns = %d, want %d", len(definition.Grid.Columns), defaultAutoGridColumns)
	}
	if got := definition.Grid.Columns[defaultAutoGridColumns-1]; got != "field08" {
		t.Fatalf("last default grid column = %q, want field08", got)
	}
}

func TestAutoTenantTableAllowsExplicitGridPageSize(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL,
        name TEXT, archived INTEGER NOT NULL DEFAULT 0
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"), WithGridPageSize(10))
	if err != nil {
		t.Fatal(err)
	}
	if got := definition.Grid.Pagination; got.DefaultSize != 10 || len(got.AllowedSizes) != 4 || got.AllowedSizes[0] != 10 || got.AllowedSizes[3] != 100 {
		t.Fatalf("grid pagination = %#v, want default 10 and bounded sizes", got)
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithGridPageSize(0)); err == nil {
		t.Fatal("zero grid page size accepted")
	}
}

func TestAutoTenantTableTableOnlyConventionOmitsDeleteWithoutArchive(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        version INTEGER NOT NULL
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"))
	if err != nil {
		t.Fatal(err)
	}
	if definition.Permissions.Delete != "" || definition.Delete.Mode != crud.DeleteModeNone {
		t.Fatalf("table-only delete policy = permissions:%q policy:%q", definition.Permissions.Delete, definition.Delete.Mode)
	}
}

func TestAutoTenantTableEnablesPhysicalDeleteOnlyWhenExplicit(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY AUTOINCREMENT, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        version INTEGER NOT NULL
    ); CREATE INDEX contacts_listing ON contacts (tenant_id, name)`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithHardDelete())
	if err != nil {
		t.Fatal(err)
	}
	if definition.Delete.Mode != crud.DeleteModeHardDelete || definition.Permissions.Delete == "" {
		t.Fatalf("hard-delete definition = %#v", definition)
	}
	source := definition.Source.(*SimpleTable)
	record, err := source.Create(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.Mutation{Fields: crud.Fields{"name": "physical"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := source.Delete(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, record.ID, record.Version, crud.DeleteModeHardDelete); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Get(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, record.ID); err == nil {
		t.Fatal("physically deleted record remained readable")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithHardDelete(), WithSoftDelete("missing")); err == nil {
		t.Fatal("hard-delete and soft-delete accepted together")
	}
	registry := crud.NewRegistry()
	if err := RegisterAutoTenantTable(context.Background(), database, registry, mustIdentifier("contacts"), WithHardDelete()); err != nil {
		t.Fatalf("RegisterAutoTenantTable() error = %v", err)
	}
	if _, ok := registry.Get("contacts"); !ok {
		t.Fatal("hard-delete tenant resource was not registered")
	}
}

func TestAutoTableComposesBooleanDefaultSortFromIndex(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        enabled BOOLEAN NOT NULL, version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	if _, err := database.Exec(`CREATE INDEX contacts_listing ON contacts (tenant_id, enabled, name)`); err != nil {
		t.Fatal(err)
	}
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"))
	if err != nil {
		t.Fatal(err)
	}
	want := []crud.Sort{{Field: "enabled", Direction: crud.SortAscending}, {Field: "name", Direction: crud.SortAscending}}
	if got := definition.Grid.DefaultSort; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("default sort = %#v, want %#v", got, want)
	}
}

func TestAutoTableKeepsIndexedSortWhenGridProjectionIsExplicitlyReordered(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        notes TEXT, status TEXT NOT NULL, enabled BOOLEAN NOT NULL,
        version INTEGER NOT NULL
    )`)
	if _, err := database.Exec(`CREATE INDEX contacts_listing ON contacts (tenant_id, enabled, name)`); err != nil {
		t.Fatal(err)
	}
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"),
		WithGridColumns("status", "name", "enabled", "notes"),
		WithFormFields("status", "name", "enabled", "notes"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := definition.Grid.Columns; !reflect.DeepEqual(got, []crud.FieldKey{"status", "name", "enabled", "notes"}) {
		t.Fatalf("explicit grid columns = %#v", got)
	}
	if got := definition.Form.Fields; !reflect.DeepEqual(got, []crud.FieldKey{"status", "name", "enabled", "notes"}) {
		t.Fatalf("explicit form fields = %#v", got)
	}
	wantSort := []crud.Sort{{Field: "enabled", Direction: crud.SortAscending}, {Field: "name", Direction: crud.SortAscending}}
	if got := definition.Grid.DefaultSort; !reflect.DeepEqual(got, wantSort) {
		t.Fatalf("default sort = %#v, want %#v", got, wantSort)
	}
}

func TestAutoTableRejectsArchivePrefixThatCannotSupportOrdering(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        enabled BOOLEAN NOT NULL, version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	if _, err := database.Exec(`CREATE INDEX contacts_archive_listing ON contacts (tenant_id, archived, enabled, name)`); err != nil {
		t.Fatal(err)
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived")); err == nil {
		t.Fatal("archive-prefixed index accepted as support for active ordering")
	}
}

func TestAutoTenantTableValidatesExplicitSortAgainstIndex(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        enabled BOOLEAN NOT NULL, version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	if _, err := database.Exec(`CREATE INDEX contacts_name ON contacts (tenant_id, name)`); err != nil {
		t.Fatal(err)
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"), WithDefaultSort(crud.Sort{Field: "name", Direction: crud.SortAscending})); err != nil {
		t.Fatalf("compatible explicit sort rejected: %v", err)
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"), WithDefaultSort(
		crud.Sort{Field: "enabled", Direction: crud.SortAscending},
		crud.Sort{Field: "name", Direction: crud.SortAscending},
	)); err == nil {
		t.Fatal("unsupported explicit composite sort accepted")
	}
	partialIndexDatabase := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        enabled BOOLEAN NOT NULL, version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	if _, err := partialIndexDatabase.Exec(`CREATE INDEX contacts_enabled ON contacts (tenant_id, enabled)`); err != nil {
		t.Fatal(err)
	}
	if _, err := AutoTenantTable(context.Background(), partialIndexDatabase, mustIdentifier("contacts"), WithSoftDelete("archived")); err == nil {
		t.Fatal("automatic boolean sort without a supporting secondary field index accepted")
	}
}

func TestAutoTableComposesBooleanDefaultSortWithoutIndex(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        enabled BOOLEAN NOT NULL, version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"))
	if err != nil {
		t.Fatal(err)
	}
	if got := definition.Grid.DefaultSort; len(got) != 2 || got[0].Field != "enabled" || got[1].Field != "name" {
		t.Fatalf("default sort = %#v, want enabled then name", got)
	}
}

func TestAutoTableUsesOrderedSuccessorWhenBooleanIsSoleListColumn(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        enabled BOOLEAN NOT NULL, version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"), WithGridColumns("enabled"))
	if err != nil {
		t.Fatal(err)
	}
	if got := definition.Grid.DefaultSort; len(got) != 2 || got[0].Field != "enabled" || got[1].Field != "name" {
		t.Fatalf("default sort = %#v, want enabled then name", got)
	}
}

func TestAutoTableKeepsSingleBooleanSortWhenItHasNoSuccessor(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL,
        enabled BOOLEAN NOT NULL, version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"))
	if err != nil {
		t.Fatal(err)
	}
	if got := definition.Grid.DefaultSort; len(got) != 1 || got[0].Field != "enabled" {
		t.Fatalf("default sort = %#v, want enabled only", got)
	}
}

func TestAutoTableRejectsInvalidDefaultSortConfiguration(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        enabled BOOLEAN NOT NULL, version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	if _, err := database.Exec(`CREATE INDEX contacts_name ON contacts (tenant_id, name)`); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]crud.Sort{
		"unknown field":     {{Field: "missing", Direction: crud.SortAscending}},
		"invalid key":       {{Field: "Name", Direction: crud.SortAscending}},
		"invalid direction": {{Field: "name", Direction: "sideways"}},
		"duplicate field":   {{Field: "name", Direction: crud.SortAscending}, {Field: "name", Direction: crud.SortAscending}},
		"mixed direction":   {{Field: "enabled", Direction: crud.SortAscending}, {Field: "name", Direction: crud.SortDescending}},
	}
	for name, terms := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"), WithDefaultSort(terms...)); err == nil {
				t.Fatal("invalid default sort accepted")
			}
		})
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithDefaultSort()); err == nil {
		t.Fatal("empty default sort accepted")
	}
}

func TestAutoTableIgnoresPartialIndexesForAutomaticOrdering(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, name TEXT NOT NULL,
        enabled BOOLEAN NOT NULL, version INTEGER NOT NULL
    )`)
	if _, err := database.Exec(`CREATE INDEX contacts_partial ON contacts (tenant_id, enabled, name) WHERE enabled = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`CREATE INDEX contacts_expression ON contacts (lower(name))`); err != nil {
		t.Fatal(err)
	}
	definition, err := AutoTable(context.Background(), database, AutoTableConfig{
		Table: mustIdentifier("contacts"), ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")},
		Permissions: crud.Permissions{Read: "contacts.read"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := definition.Grid.DefaultSort; len(got) != 2 || got[0].Field != "enabled" || got[1].Field != "name" {
		t.Fatalf("default sort = %#v, want fallback enabled then name", got)
	}
}

func TestAutoTenantTableAllowsExplicitGridColumns(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL,
        field01 TEXT, field02 TEXT, field03 TEXT, archived INTEGER NOT NULL DEFAULT 0
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"), WithGridColumns("field03", "field01"))
	if err != nil {
		t.Fatal(err)
	}
	if got := definition.Grid.Columns; len(got) != 2 || got[0] != "field03" || got[1] != "field01" {
		t.Fatalf("explicit grid columns = %#v", got)
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"), WithGridColumns("missing")); err == nil {
		t.Fatal("unknown explicit grid column accepted")
	}
}

func TestAutoTenantTableAllowsExplicitFormFields(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL,
        name TEXT, enabled BOOLEAN NOT NULL, notes TEXT, archived INTEGER NOT NULL DEFAULT 0
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"), WithFormFields("enabled", "name"))
	if err != nil {
		t.Fatal(err)
	}
	if got := definition.Form.Fields; len(got) != 2 || got[0] != "enabled" || got[1] != "name" {
		t.Fatalf("explicit form fields = %#v", got)
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"), WithFormFields("missing")); err == nil {
		t.Fatal("unknown explicit form field accepted")
	}
}

func TestAutoTenantTableRegistersOneLineConvention(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL,
        name TEXT, enabled BOOLEAN NOT NULL, archived INTEGER NOT NULL DEFAULT 0
    )`)
	registry := crud.NewRegistry()
	if err := RegisterAutoTenantTable(context.Background(), database, registry, mustIdentifier("contacts"), WithSoftDelete("archived"), WithMaxLength("name", 50), WithOptional("name"), WithBooleanDisplay("enabled", "●", "○")); err != nil {
		t.Fatal(err)
	}
	definition, ok := registry.Get("contacts")
	if !ok || len(definition.Scope.Keys) != 1 || definition.Scope.Keys[0] != "tenant_id" {
		t.Fatalf("definition scope = %#v", definition.Scope)
	}
	if definition.Permissions.Read != "crud.contacts.read" || definition.Permissions.Delete != "crud.contacts.delete" {
		t.Fatalf("permissions = %#v", definition.Permissions)
	}
	if got := fieldByKey(t, definition.Fields, "name").MaxLength; got != 50 {
		t.Fatalf("name maximum = %d, want 50", got)
	}
	if fieldByKey(t, definition.Fields, "name").IsRequired() {
		t.Fatal("explicit optional field is still required")
	}
	if got := fieldByKey(t, definition.Fields, "enabled").BooleanDisplay; got == nil || got.True != "●" || got.False != "○" {
		t.Fatalf("boolean display = %#v", got)
	}
}

func TestAutoTenantTableAllowsEnumAndAlternateResourceKey(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL,
        name TEXT NOT NULL, status TEXT NOT NULL CHECK (status IN ('new', 'closed', 'review')), archived INTEGER NOT NULL DEFAULT 0
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"),
		WithResourceKey("contacts_status"),
		WithSoftDelete("archived"),
		WithEnum("status", crud.Option{Value: "new", Label: "crud.status.new"}, crud.Option{Value: "closed", Label: "crud.status.closed"}),
		WithEnumControl("status", crud.EnumControlRadio),
	)
	if err != nil {
		t.Fatal(err)
	}
	if definition.Key != "contacts_status" || definition.Permissions.Read != "crud.contacts_status.read" {
		t.Fatalf("alternate resource key = %q, permissions = %#v", definition.Key, definition.Permissions)
	}
	field := fieldByKey(t, definition.Fields, "status")
	if field.Type != crud.FieldEnum || field.EnumControl != crud.EnumControlRadio || len(field.Enum) != 2 || field.Enum[0].Value != "new" {
		t.Fatalf("status enum = %#v", field)
	}
	if err := crud.ValidateDefinition(context.Background(), definition); err != nil {
		t.Fatalf("generated enum definition is invalid: %v", err)
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithEnum("status")); err == nil {
		t.Fatal("empty enum accepted")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithEnum("status", crud.Option{Value: "new", Label: "crud.status.new"}, crud.Option{Value: "new", Label: "crud.status.duplicate"})); err == nil {
		t.Fatal("duplicate enum value accepted")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithEnumControl("status", "chips")); err == nil {
		t.Fatal("invalid enum control accepted")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithEnum("status", crud.Option{Value: "cancelled", Label: "crud.status.cancelled"})); err == nil {
		t.Fatal("enum value outside schema accepted")
	}
}

func TestAutoTablesDeclareLookupAndGlobalReferenceDefaults(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL,
        name TEXT NOT NULL, state_id INTEGER NOT NULL
    ); CREATE INDEX contacts_listing ON contacts (tenant_id, name);
    CREATE TABLE reference_states (
        id INTEGER PRIMARY KEY, name TEXT NOT NULL, version INTEGER NOT NULL
    ); CREATE INDEX reference_states_name ON reference_states (name)`)
	tenant, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"),
		WithLookup("state_id", crud.LookupDefinition{Resource: "reference_states", ValueField: "id", LabelField: "name", PageSize: 25}),
	)
	if err != nil {
		t.Fatal(err)
	}
	lookup := fieldByKey(t, tenant.Fields, "state_id")
	if lookup.Type != crud.FieldLookup || lookup.Lookup == nil || lookup.Lookup.Resource != "reference_states" {
		t.Fatalf("lookup field = %#v", lookup)
	}
	global, err := AutoGlobalTable(context.Background(), database, mustIdentifier("reference_states"))
	if err != nil {
		t.Fatal(err)
	}
	if global.Scope.Mode != crud.ScopeModeGlobal || global.Permissions.Create != "" || global.Permissions.Update != "" || global.Permissions.Delete != "" || global.Delete.Mode != crud.DeleteModeNone {
		t.Fatalf("global definition = %#v", global)
	}
	if _, ok := global.Source.(*SimpleTable); !ok || !global.Source.(*SimpleTable).table.Global {
		t.Fatal("global source did not preserve explicit global mode")
	}
	registry := crud.NewRegistry()
	if err := RegisterAutoGlobalTable(context.Background(), database, registry, mustIdentifier("reference_states")); err != nil {
		t.Fatalf("RegisterAutoGlobalTable() error = %v", err)
	}
	if _, ok := registry.Get("reference_states"); !ok {
		t.Fatal("global resource was not registered")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithLookup("state_id", crud.LookupDefinition{Resource: "reference_states", ValueField: "id", LabelField: "name", Dependencies: []crud.FieldKey{"state_id", "state_id"}, PageSize: 25})); err == nil {
		t.Fatal("duplicate lookup dependencies accepted")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithLookup("state_id", crud.LookupDefinition{Resource: "reference_states", ValueField: "id", LabelField: "name", FixedFilters: []crud.FixedLookupFilter{{Field: "kind", Values: []crud.Value{"state"}}, {Field: "kind", Values: []crud.Value{"currency"}}}, PageSize: 25})); err == nil {
		t.Fatal("duplicate fixed lookup filters accepted")
	}
	if _, err := AutoGlobalTable(context.Background(), database, mustIdentifier("reference_states"), WithHardDelete()); err == nil {
		t.Fatal("global table accepted physical deletion")
	}
	if _, err := AutoGlobalTable(context.Background(), database, mustIdentifier("reference_states"), nil); err == nil {
		t.Fatal("nil global table option accepted")
	}
}

func TestAutoTableExplicitLookupWinsOverSchemaEnum(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL,
        name TEXT NOT NULL, channel TEXT NOT NULL CHECK (channel IN ('email', 'phone'))
    );
    CREATE TABLE reference_channels (
        id INTEGER PRIMARY KEY, name TEXT NOT NULL, version INTEGER NOT NULL
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"),
		WithLookup("channel", crud.LookupDefinition{Resource: "reference_channels", ValueField: "id", LabelField: "name", PageSize: 25}),
	)
	if err != nil {
		t.Fatal(err)
	}
	field := fieldByKey(t, definition.Fields, "channel")
	if field.Type != crud.FieldLookup || field.Lookup == nil || field.Lookup.Resource != "reference_channels" || len(field.Enum) != 0 {
		t.Fatalf("explicit lookup was replaced by schema enum: %#v", field)
	}
}

func TestWithLookupCopiesFixedFilterValues(t *testing.T) {
	lookup := crud.LookupDefinition{
		Resource:     "reference_values",
		ValueField:   "id",
		LabelField:   "name",
		PageSize:     25,
		FixedFilters: []crud.FixedLookupFilter{{Field: "kind", Values: []crud.Value{"state", "territory"}}},
	}
	config := &AutoTableConfig{}
	if err := WithLookup("state_id", lookup)(config); err != nil {
		t.Fatal(err)
	}
	lookup.FixedFilters[0].Values[0] = "currency"
	stored := config.FieldOverrides["state_id"].Lookup
	if stored == nil || stored.FixedFilters[0].Values[0] != "state" {
		t.Fatalf("fixed lookup filter was not copied: %#v", stored)
	}
	if err := WithLookup("state_id", crud.LookupDefinition{Resource: "reference_values", ValueField: "id", LabelField: "name", PageSize: 25, FixedFilters: []crud.FixedLookupFilter{{Field: "kind", Values: []crud.Value{1}}}})(&AutoTableConfig{}); err == nil {
		t.Fatal("lookup accepted an unsupported fixed-filter value")
	}
}

func TestAutoTableRejectsInvalidFieldOverrides(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, version INTEGER NOT NULL,
        name TEXT, enabled BOOLEAN NOT NULL
    )`)
	config := AutoTableConfig{
		Table: mustIdentifier("contacts"), ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")},
		Permissions: crud.Permissions{Read: "contacts.read"},
	}
	for name, overrides := range map[string]map[crud.FieldKey]FieldOverride{
		"zero bound":              {"name": {MaxLength: 0}},
		"oversized bound":         {"name": {MaxLength: maximumAutoTextMaxLength + 1}},
		"unknown field":           {"unknown": {MaxLength: 50}},
		"non-text field":          {"enabled": {MaxLength: 50}},
		"optional not null":       {"enabled": {Optional: true}},
		"boolean display on text": {"name": {BooleanDisplay: &crud.BooleanDisplay{True: "●", False: "○"}}},
		"pattern on boolean":      {"enabled": {Pattern: `^(yes|no)$`, PatternMessage: "crud.enabled.pattern"}},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := config
			candidate.FieldOverrides = overrides
			if _, err := AutoTable(context.Background(), database, candidate); err == nil {
				t.Fatal("invalid field override accepted")
			}
		})
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithMaxLength("name", 0)); err == nil {
		t.Fatal("invalid registration option accepted")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithBooleanDisplay("enabled", "", "○")); err == nil {
		t.Fatal("invalid boolean display accepted")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithPattern("name", "[", "crud.name.pattern")); err == nil {
		t.Fatal("invalid pattern accepted")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithPattern("name", "", "crud.name.pattern")); err == nil {
		t.Fatal("empty pattern accepted")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithPattern("name", "^[A-Z].*$", "")); err == nil {
		t.Fatal("empty pattern message accepted")
	}
	if _, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithPattern("name", strings.Repeat("a", maximumAutoPatternLength+1), "crud.name.pattern")); err == nil {
		t.Fatal("oversized pattern accepted")
	}
}

func newAutoDatabase(t *testing.T, schema string) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return database
}

func fieldByKey(t *testing.T, fields []crud.Field, key crud.FieldKey) crud.Field {
	t.Helper()
	for _, field := range fields {
		if field.Key == key {
			return field
		}
	}
	t.Fatalf("field %q absent", key)
	return crud.Field{}
}

func containsField(fields []crud.Field, key crud.FieldKey) bool {
	for _, field := range fields {
		if field.Key == key {
			return true
		}
	}
	return false
}

func containsKey(keys []crud.FieldKey, wanted crud.FieldKey) bool {
	for _, key := range keys {
		if key == wanted {
			return true
		}
	}
	return false
}
