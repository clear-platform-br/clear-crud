package sqladapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	crud "github.com/clear-platform-br/clear-crud"
	"github.com/clear-platform-br/clear-crud/conformance"
	_ "modernc.org/sqlite"
)

func TestSQLiteSimpleTableConformance(t *testing.T) {
	conformance.TestDataSource(t, newSQLiteFixture)
}

func TestNewIdentifier(t *testing.T) {
	for _, value := range []string{"contacts", "tenant_id", "field_2"} {
		if _, err := NewIdentifier(value); err != nil {
			t.Fatalf("NewIdentifier(%q): %v", value, err)
		}
	}
	for _, value := range []string{"", "contacts; DROP TABLE contacts", "tenant-id", `"contacts"`} {
		if _, err := NewIdentifier(value); err == nil {
			t.Fatalf("NewIdentifier(%q) accepted unsafe identifier", value)
		}
	}
}

func TestSimpleTableArchiveMetadataNormalizesConventionalValues(t *testing.T) {
	source := &SimpleTable{table: sqliteTable()}
	for _, test := range []struct {
		name     string
		value    any
		archived bool
	}{
		{name: "nil", value: nil},
		{name: "false", value: false},
		{name: "true", value: true, archived: true},
		{name: "zero integer", value: int64(0)},
		{name: "nonzero integer", value: int64(1), archived: true},
		{name: "zero native integer", value: int(0)},
		{name: "nonzero native integer", value: int(2), archived: true},
		{name: "zero text", value: "0"},
		{name: "nonzero text", value: "1", archived: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := source.isArchivedValue(test.value); got != test.archived {
				t.Fatalf("isArchivedValue(%#v) = %v, want %v", test.value, got, test.archived)
			}
		})
	}

	state := sqliteTable()
	state.ArchiveColumn = nil
	state.ArchiveState = &ArchiveState{Column: mustIdentifier("status"), ActiveValue: "active", ArchivedValue: "archived"}
	stateSource := &SimpleTable{table: state}
	for _, test := range []struct {
		value    any
		archived bool
	}{
		{value: "active"},
		{value: "archived", archived: true},
		{value: "other"},
	} {
		if got := stateSource.isArchivedValue(test.value); got != test.archived {
			t.Fatalf("state isArchivedValue(%#v) = %v, want %v", test.value, got, test.archived)
		}
	}
	if !archiveValuesEqual(int64(4), int64(4)) || archiveValuesEqual(int64(4), int64(5)) {
		t.Fatal("integer archive values were compared incorrectly")
	}
	if !archiveValuesEqual(true, true) || archiveValuesEqual(true, false) {
		t.Fatal("boolean archive values were compared incorrectly")
	}
	if !archiveValuesEqual(nil, nil) || archiveValuesEqual(nil, "") {
		t.Fatal("nil archive values were compared incorrectly")
	}
}

func TestNewSimpleTableRejectsInvalidConfig(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := NewSimpleTable(nil, sqliteTable()); err == nil {
		t.Fatal("nil database accepted")
	}
	bad := sqliteTable()
	bad.Fields = nil
	if _, err := NewSimpleTable(database, bad); err == nil {
		t.Fatal("missing fields accepted")
	}
	bad = sqliteTable()
	bad.IDField = "name"
	if _, err := NewSimpleTable(database, bad); err == nil {
		t.Fatal("id field mapped to another column accepted")
	}
	bad = sqliteTable()
	bad.ArchiveState = &ArchiveState{Column: mustIdentifier("archived"), ActiveValue: "active", ArchivedValue: "archived"}
	if _, err := NewSimpleTable(database, bad); err == nil {
		t.Fatal("two archive mappings accepted")
	}
}

func TestSimpleTableSupportsDeclaredIdentityStateArchiveAndManagedColumns(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.SetMaxOpenConns(1)
	if _, err = database.Exec(`CREATE TABLE contacts (
        id TEXT PRIMARY KEY,
        tenant_id TEXT NOT NULL,
        name TEXT NOT NULL,
        status TEXT NOT NULL,
        version INTEGER NOT NULL,
        created_at TEXT NOT NULL,
        created_by TEXT NOT NULL,
        updated_at TEXT NOT NULL,
        updated_by TEXT NOT NULL
    )`); err != nil {
		t.Fatal(err)
	}
	source, err := NewSimpleTable(database, TableDefinition{
		Table: mustIdentifier("contacts"), IDColumn: mustIdentifier("id"), IDField: "code", VersionColumn: mustIdentifier("version"),
		ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")},
		ArchiveState: &ArchiveState{Column: mustIdentifier("status"), ActiveValue: "active", ArchivedValue: "archived"},
		Managed: []ManagedColumn{
			{Column: mustIdentifier("created_at"), OnCreate: ManagedUTCNow()},
			{Column: mustIdentifier("created_by"), OnCreate: ManagedFromScope("actor_id")},
			{Column: mustIdentifier("updated_at"), OnCreate: ManagedUTCNow(), OnUpdate: ManagedUTCNow()},
			{Column: mustIdentifier("updated_by"), OnCreate: ManagedFromScope("actor_id"), OnUpdate: ManagedFromScope("actor_id")},
		},
		Fields: map[crud.FieldKey]Identifier{"code": mustIdentifier("id"), "name": mustIdentifier("name")},
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := crud.Scope{"tenant_id": "tenant-a", "actor_id": "operator-a"}
	created, err := source.Create(context.Background(), scope, crud.Mutation{Fields: crud.Fields{"code": "contact-64", "name": "Ana"}})
	if err != nil || created.ID != "contact-64" || created.Fields["code"] != "contact-64" {
		t.Fatalf("Create() = %#v, %v", created, err)
	}
	updated, err := source.Update(context.Background(), scope, created.ID, created.Version, crud.Mutation{Fields: crud.Fields{"name": "Ana Maria"}})
	if err != nil || updated.Version != 2 || updated.Fields["code"] != "contact-64" {
		t.Fatalf("Update() = %#v, %v", updated, err)
	}
	if err := source.Delete(context.Background(), scope, created.ID, updated.Version, crud.DeleteModeArchive); err != nil {
		t.Fatal(err)
	}
	var status, createdBy, updatedBy, createdAt, updatedAt string
	if err := database.QueryRow(`SELECT status, created_by, updated_by, created_at, updated_at FROM contacts WHERE id='contact-64'`).Scan(&status, &createdBy, &updatedBy, &createdAt, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if status != "archived" || createdBy != "operator-a" || updatedBy != "operator-a" || createdAt == "" || updatedAt == "" {
		t.Fatalf("technical columns = status:%q createdBy:%q updatedBy:%q createdAt:%q updatedAt:%q", status, createdBy, updatedBy, createdAt, updatedAt)
	}
	_, err = source.Get(context.Background(), scope, created.ID)
	requireCode(t, err, crud.ErrorNotFound)
	archivedPage, err := source.List(context.Background(), scope, crud.Query{IncludeArchived: true, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}})
	if err != nil || len(archivedPage.Records) != 1 || !archivedPage.Records[0].Archived || archivedPage.Records[0].Fields["name"] != "Ana Maria" {
		t.Fatalf("List(include archived) = %#v, %v", archivedPage, err)
	}
	_, err = source.Create(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.Mutation{Fields: crud.Fields{"code": "contact-65", "name": "Bia"}})
	requireCode(t, err, crud.ErrorForbidden)
}

func TestSimpleTableQueryAndPolicies(t *testing.T) {
	fixture := newSQLiteFixture(t)
	defer fixture.Close()
	source := fixture.Source.(*SimpleTable)
	ctx := context.Background()
	for _, name := range []string{"Ana", "Bruno", "Carla"} {
		if _, err := source.Create(ctx, fixture.ScopeA, fixture.NewMutation(name)); err != nil {
			t.Fatal(err)
		}
	}
	queries := []crud.Query{
		{Search: "an", Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}},
		{Filters: []crud.Filter{{Field: "name", Operator: crud.FilterEqual, Value: "Ana"}}, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}},
		{Filters: []crud.Filter{{Field: "name", Operator: crud.FilterIn, Values: []crud.Value{"Ana", "Bruno"}}}, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}},
		{Filters: []crud.Filter{{Field: "name", Operator: crud.FilterNotEqual, Value: "Ana"}}, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}},
		{Filters: []crud.Filter{{Field: "name", Operator: crud.FilterContains, Value: "r"}}, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}},
		{Filters: []crud.Filter{{Field: "name", Operator: crud.FilterPrefix, Value: "A"}}, Sort: []crud.Sort{{Field: "name", Direction: crud.SortDescending}}, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}},
		{Filters: []crud.Filter{{Field: "name", Operator: crud.FilterIsNull}}, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 10}},
	}
	for _, query := range queries {
		page, err := source.List(ctx, fixture.ScopeA, query)
		if err != nil || page.Total == nil {
			t.Fatalf("List(%#v): page=%#v err=%v", query, page, err)
		}
	}
	_, err := source.List(ctx, fixture.ScopeA, crud.Query{Filters: []crud.Filter{{Field: "unknown", Operator: crud.FilterEqual, Value: "x"}}, Page: crud.PageRequest{Mode: crud.PageModeOffset, Number: 1, Size: 1}})
	requireCode(t, err, crud.ErrorInvalidRequest)
	_, err = source.Lookup(ctx, fixture.ScopeA, crud.LookupQuery{})
	requireCode(t, err, crud.ErrorInvalidRequest)
	_, err = source.Get(ctx, crud.Scope{}, "1")
	requireCode(t, err, crud.ErrorForbidden)

	withoutArchive := sqliteTable()
	withoutArchive.ArchiveColumn = nil
	plain, err := NewSimpleTable(source.db, withoutArchive)
	if err != nil {
		t.Fatal(err)
	}
	record, err := plain.Create(ctx, fixture.ScopeA, fixture.NewMutation("Dora"))
	if err != nil {
		t.Fatal(err)
	}
	requireCode(t, plain.Delete(ctx, fixture.ScopeA, record.ID, record.Version, crud.DeleteModeArchive), crud.ErrorDeleteRestricted)
	requireCode(t, plain.Delete(ctx, fixture.ScopeA, record.ID, record.Version+1, crud.DeleteModeHardDelete), crud.ErrorConflict)
}

func TestSimpleTableHardDeleteLetsTheDatabaseEnforceForeignKeys(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.SetMaxOpenConns(1)
	if _, err := database.Exec(`PRAGMA foreign_keys = ON;
        CREATE TABLE parents (id INTEGER PRIMARY KEY AUTOINCREMENT, tenant_id TEXT NOT NULL, name TEXT NOT NULL, version INTEGER NOT NULL);
        CREATE TABLE children (id INTEGER PRIMARY KEY AUTOINCREMENT, parent_id INTEGER NOT NULL REFERENCES parents(id), note TEXT NOT NULL);`); err != nil {
		t.Fatal(err)
	}
	source, err := NewSimpleTable(database, TableDefinition{
		Table: mustIdentifier("parents"), IDColumn: mustIdentifier("id"), VersionColumn: mustIdentifier("version"),
		ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")},
		Fields:       map[crud.FieldKey]Identifier{"name": mustIdentifier("name")},
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := crud.Scope{"tenant_id": "tenant-a"}
	parent, err := source.Create(context.Background(), scope, crud.Mutation{Fields: crud.Fields{"name": "parent"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO children (parent_id, note) VALUES (?, 'dependent')`, parent.ID); err != nil {
		t.Fatal(err)
	}
	if err := source.Delete(context.Background(), scope, parent.ID, parent.Version, crud.DeleteModeHardDelete); !errors.Is(err, &crud.Error{Code: crud.ErrorDeleteRestricted}) {
		t.Fatalf("hard delete with dependent row = %v, want delete restricted", err)
	}
	if _, err := database.Exec(`DELETE FROM children WHERE parent_id = ?`, parent.ID); err != nil {
		t.Fatal(err)
	}
	if err := source.Delete(context.Background(), scope, parent.ID, parent.Version, crud.DeleteModeHardDelete); err != nil {
		t.Fatalf("hard delete after dependent removal = %v", err)
	}
}

func TestForeignKeyViolationDetectionIsConservative(t *testing.T) {
	if isForeignKeyViolation(nil) {
		t.Fatal("nil error classified as foreign-key violation")
	}
	if !isForeignKeyViolation(errors.New("foreign key constraint violation")) {
		t.Fatal("foreign-key violation message was not recognized")
	}
	if isForeignKeyViolation(errors.New("unique constraint failed")) {
		t.Fatal("unrelated constraint classified as foreign-key violation")
	}
}

func TestSimpleTableLookupUsesDeclaredDependenciesAndCursor(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.SetMaxOpenConns(1)
	if _, err := database.Exec(`CREATE TABLE reference_regions (
        id INTEGER PRIMARY KEY, state_id INTEGER NOT NULL, name TEXT NOT NULL, version INTEGER NOT NULL
    ); CREATE INDEX reference_regions_name ON reference_regions (name);
    INSERT INTO reference_regions (id, state_id, name, version) VALUES (11, 1, 'Alpha', 1), (12, 1, 'Beta', 1), (21, 2, 'Gamma', 1)`); err != nil {
		t.Fatal(err)
	}
	source, err := NewSimpleTable(database, TableDefinition{Table: mustIdentifier("reference_regions"), IDColumn: mustIdentifier("id"), VersionColumn: mustIdentifier("version"), Global: true, Fields: map[crud.FieldKey]Identifier{"state_id": "state_id", "name": "name"}})
	if err != nil {
		t.Fatal(err)
	}
	page, err := source.Lookup(context.Background(), crud.Scope{"tenant_id": "ignored"}, crud.LookupQuery{ValueField: "id", LabelField: "name", Search: "a", Size: 1, Dependencies: crud.Fields{"state_id": int64(1)}})
	if err != nil || len(page.Options) != 1 || page.NextCursor == "" || page.Options[0].Value != int64(11) {
		t.Fatalf("first lookup = %#v, %v", page, err)
	}
	next, err := source.Lookup(context.Background(), nil, crud.LookupQuery{ValueField: "id", LabelField: "name", Size: 2, Cursor: page.NextCursor, Dependencies: crud.Fields{"state_id": int64(1)}})
	if err != nil || len(next.Options) != 1 || next.Options[0].Value != int64(12) || next.NextCursor != "" {
		t.Fatalf("cursor lookup = %#v, %v", next, err)
	}
	exact, err := source.Lookup(context.Background(), nil, crud.LookupQuery{ValueField: "id", LabelField: "name", Search: "21", Size: 2, Dependencies: crud.Fields{"state_id": int64(2)}})
	if err != nil || len(exact.Options) != 1 || exact.Options[0].Value != int64(21) || exact.Options[0].Label != "Gamma" {
		t.Fatalf("exact value lookup = %#v, %v", exact, err)
	}
	for _, query := range []crud.LookupQuery{{ValueField: "id", LabelField: "name"}, {ValueField: "missing", LabelField: "name", Size: 1}, {ValueField: "id", LabelField: "missing", Size: 1}, {ValueField: "id", LabelField: "name", Size: 1, Cursor: "bad"}, {ValueField: "id", LabelField: "name", Size: 1, Dependencies: crud.Fields{"missing": int64(1)}}} {
		if _, err := source.Lookup(context.Background(), nil, query); err == nil {
			t.Fatalf("invalid lookup accepted: %#v", query)
		}
	}
	if _, err := source.Create(context.Background(), nil, crud.Mutation{}); err == nil {
		t.Fatal("global lookup table accepted a mutation")
	}
	if stringValue([]byte("bytes")) != "bytes" || sqlValue([]byte("bytes")) != "bytes" {
		t.Fatal("byte values were not normalized")
	}
}

func TestSimpleTableLookupCombinesFixedFiltersScopeArchiveAndDependencies(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.SetMaxOpenConns(1)
	if _, err := database.Exec(`CREATE TABLE reference_values (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL, kind TEXT NOT NULL,
        state_id INTEGER NOT NULL, name TEXT NOT NULL, version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    ); CREATE INDEX reference_values_lookup ON reference_values (tenant_id, kind, state_id, name);
    INSERT INTO reference_values (id, tenant_id, kind, state_id, name, version, archived) VALUES
        (1, 'tenant-a', 'state', 10, 'Visible', 1, 0),
        (2, 'tenant-a', 'currency', 10, 'Wrong kind', 1, 0),
        (3, 'tenant-a', 'state', 20, 'Wrong dependency', 1, 0),
        (4, 'tenant-b', 'state', 10, 'Wrong tenant', 1, 0),
        (5, 'tenant-a', 'state', 10, 'Archived', 1, 1),
        (6, 'tenant-a', 'territory', 10, 'Territory', 1, 0)`); err != nil {
		t.Fatal(err)
	}
	archiveColumn := mustIdentifier("archived")
	source, err := NewSimpleTable(database, TableDefinition{Table: mustIdentifier("reference_values"), IDColumn: mustIdentifier("id"), VersionColumn: mustIdentifier("version"), ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")}, ArchiveColumn: &archiveColumn, Fields: map[crud.FieldKey]Identifier{"kind": mustIdentifier("kind"), "state_id": mustIdentifier("state_id"), "name": mustIdentifier("name")}})
	if err != nil {
		t.Fatal(err)
	}
	page, err := source.Lookup(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.LookupQuery{ValueField: "id", LabelField: "name", Size: 10, Dependencies: crud.Fields{"state_id": int64(10)}, FixedFilters: []crud.FixedLookupFilter{{Field: "kind", Values: []crud.Value{"state"}}}})
	if err != nil || len(page.Options) != 1 || page.Options[0].Value != int64(1) || page.Options[0].Label != "Visible" {
		t.Fatalf("combined lookup = %#v, %v", page, err)
	}
	page, err = source.Lookup(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.LookupQuery{ValueField: "id", LabelField: "name", Size: 10, Dependencies: crud.Fields{"state_id": int64(10)}, FixedFilters: []crud.FixedLookupFilter{{Field: "kind", Values: []crud.Value{"state", "territory"}}}})
	if err != nil || len(page.Options) != 2 || page.Options[0].Value != int64(6) || page.Options[1].Value != int64(1) {
		t.Fatalf("membership lookup = %#v, %v", page, err)
	}
	if _, err := source.Lookup(context.Background(), crud.Scope{"tenant_id": "tenant-a"}, crud.LookupQuery{ValueField: "id", LabelField: "name", Size: 10, FixedFilters: []crud.FixedLookupFilter{{Field: "kind", Values: []crud.Value{nil}}}}); err == nil {
		t.Fatal("lookup accepted an invalid fixed-filter value")
	}
}

func TestSQLiteSimpleTablesShareMasterDetailTransaction(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.SetMaxOpenConns(1)
	if _, err := database.Exec(`CREATE TABLE contacts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        tenant_id TEXT NOT NULL,
        name TEXT NOT NULL,
        version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    ); CREATE TABLE contact_destinations (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        tenant_id TEXT NOT NULL,
        contact_id TEXT NOT NULL,
        address TEXT NOT NULL CHECK(address <> 'broken'),
        version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`); err != nil {
		t.Fatal(err)
	}
	contacts, err := NewSimpleTable(database, TableDefinition{
		Table: mustIdentifier("contacts"), IDColumn: mustIdentifier("id"), VersionColumn: mustIdentifier("version"),
		ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")}, ArchiveColumn: identifierPointer(mustIdentifier("archived")),
		Fields: map[crud.FieldKey]Identifier{"name": mustIdentifier("name")},
	})
	if err != nil {
		t.Fatal(err)
	}
	destinations, err := NewSimpleTable(database, TableDefinition{
		Table: mustIdentifier("contact_destinations"), IDColumn: mustIdentifier("id"), VersionColumn: mustIdentifier("version"),
		ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")}, ArchiveColumn: identifierPointer(mustIdentifier("archived")),
		Fields: map[crud.FieldKey]Identifier{"contact_id": mustIdentifier("contact_id"), "address": mustIdentifier("address")},
	})
	if err != nil {
		t.Fatal(err)
	}
	registry := crud.NewRegistry()
	parent := sqliteCRUDDefinition("contacts", contacts, []crud.Field{{Key: "name", Label: "crud.contact.name", Type: crud.FieldString, Required: true, Visible: true}}, []crud.FieldKey{"name"})
	parent.Details = []crud.DetailDefinition{{Key: "destinations", Resource: "contact_destinations", ParentField: "contact_id", Minimum: 1, Maximum: 2, AllowCreate: true, AllowUpdate: true, AllowDelete: true}}
	child := sqliteCRUDDefinition("contact_destinations", destinations, []crud.Field{
		{Key: "address", Label: "crud.destination.address", Type: crud.FieldString, Required: true, Visible: true},
		{Key: "contact_id", Label: "crud.destination.contact", Type: crud.FieldString, Required: true, ReadOnly: true, Visible: false},
	}, []crud.FieldKey{"address"})
	for _, definition := range []crud.Definition{parent, child} {
		if err := registry.Register(context.Background(), definition); err != nil {
			t.Fatal(err)
		}
	}
	service, err := crud.NewService(crud.Dependencies{
		Registry: registry, Principal: sqlPrincipal{}, Scope: sqlScope{}, Authorizer: sqlAuthorizer{}, Audit: sqlAudit{}, Translator: sqlTranslator{}, Clock: sqlClock{},
	})
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.Create(context.Background(), "contacts", crud.Mutation{
		Fields:  crud.Fields{"name": "Ana"},
		Details: crud.DetailMutations{"destinations": {{Fields: crud.Fields{"address": "ana@example.com"}}}},
	})
	if err != nil || len(created.Details["destinations"]) != 1 {
		t.Fatalf("Create() = %#v, %v", created, err)
	}
	var contactID, contactCount, destinationCount string
	if err := database.QueryRow(`SELECT id, (SELECT COUNT(*) FROM contacts), (SELECT COUNT(*) FROM contact_destinations) FROM contacts`).Scan(&contactID, &contactCount, &destinationCount); err != nil {
		t.Fatal(err)
	}
	if contactCount != "1" || destinationCount != "1" {
		t.Fatalf("successful transaction counts = contacts:%s destinations:%s", contactCount, destinationCount)
	}
	var storedParentID string
	if err := database.QueryRow(`SELECT contact_id FROM contact_destinations`).Scan(&storedParentID); err != nil {
		t.Fatal(err)
	}
	if storedParentID != contactID {
		t.Fatalf("stored parent id = %q, want %q", storedParentID, contactID)
	}
	_, err = service.Create(context.Background(), "contacts", crud.Mutation{
		Fields:  crud.Fields{"name": "Bruno"},
		Details: crud.DetailMutations{"destinations": {{Fields: crud.Fields{"address": "broken"}}}},
	})
	var public *crud.Error
	if !errors.As(err, &public) || public.Code != crud.ErrorTemporarilyUnavailable {
		t.Fatalf("failed child write error = %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM contacts`).Scan(&contactCount); err != nil {
		t.Fatal(err)
	}
	if contactCount != "1" {
		t.Fatalf("failed child write committed its parent: contacts=%s", contactCount)
	}
}

func TestFilterOperators(t *testing.T) {
	valid := []crud.FilterOperator{crud.FilterEqual, crud.FilterNotEqual, crud.FilterLessThan, crud.FilterLessOrEqual, crud.FilterGreaterThan, crud.FilterGreaterOrEqual, crud.FilterContains, crud.FilterPrefix, crud.FilterIsNull}
	for _, operator := range valid {
		if _, ok := filterOperator(operator); !ok {
			t.Fatalf("operator %q rejected", operator)
		}
	}
	if _, ok := filterOperator("bad"); ok {
		t.Fatal("invalid operator accepted")
	}
}

func requireCode(t *testing.T, err error, want crud.ErrorCode) {
	t.Helper()
	var publicError *crud.Error
	if !errors.As(err, &publicError) || publicError.Code != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func newSQLiteFixture(t testing.TB) conformance.Fixture {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	database.SetMaxOpenConns(1)
	if _, err := database.Exec(`CREATE TABLE contacts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        tenant_id TEXT NOT NULL,
        name TEXT,
        version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	source, err := NewSimpleTable(database, sqliteTable())
	if err != nil {
		t.Fatalf("new simple table: %v", err)
	}
	return conformance.Fixture{
		Source: source, UnitOfWork: source,
		ScopeA: crud.Scope{"tenant_id": "tenant-a"}, ScopeB: crud.Scope{"tenant_id": "tenant-b"},
		LookupQuery: &crud.LookupQuery{ValueField: "id", LabelField: "name", Size: 10},
		NewMutation: func(label string) crud.Mutation { return crud.Mutation{Fields: crud.Fields{"name": label}} },
		Close:       database.Close,
	}
}

func sqliteTable() TableDefinition {
	return TableDefinition{
		Table: mustIdentifier("contacts"), IDColumn: mustIdentifier("id"), VersionColumn: mustIdentifier("version"),
		ScopeColumns:  map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")},
		ArchiveColumn: identifierPointer(mustIdentifier("archived")),
		Fields:        map[crud.FieldKey]Identifier{"name": mustIdentifier("name")},
	}
}

func mustIdentifier(value string) Identifier {
	identifier, err := NewIdentifier(value)
	if err != nil {
		panic(fmt.Sprintf("test identifier %q: %v", value, err))
	}
	return identifier
}

func identifierPointer(value Identifier) *Identifier { return &value }

func sqliteCRUDDefinition(key crud.ResourceKey, source *SimpleTable, fields []crud.Field, columns []crud.FieldKey) crud.Definition {
	return crud.Definition{
		Contract: crud.ContractDefinitionV1, Key: key,
		Labels:       crud.Labels{Title: crud.MessageCode("crud." + string(key) + ".title"), Singular: crud.MessageCode("crud." + string(key) + ".singular")},
		Scope:        crud.ScopeRequirements{Keys: []string{"tenant_id"}},
		Permissions:  crud.Permissions{Create: "create", Read: "read", Update: "update", Delete: "delete"},
		Fields:       fields,
		Grid:         crud.GridDefinition{Columns: columns, Searchable: columns, Sortable: columns, DefaultSort: []crud.Sort{{Field: columns[0], Direction: crud.SortAscending}}, Pagination: crud.PaginationDefinition{Mode: crud.PageModeOffset, DefaultSize: 25, AllowedSizes: []uint16{25, 50, 100}, Total: true}},
		Presentation: crud.Presentation{Collection: crud.CollectionAuto, Density: crud.DensityCompact},
		Source:       source, UOW: source, Delete: crud.DeletePolicy{Mode: crud.DeleteModeArchive}, Concurrency: crud.ConcurrencyPolicy{Mode: crud.ConcurrencyVersion},
	}
}

type sqlPrincipal struct{}

func (sqlPrincipal) Principal(context.Context) (crud.Principal, error) {
	return crud.Principal{ID: "operator"}, nil
}

type sqlScope struct{}

func (sqlScope) Scope(context.Context, crud.ResourceKey) (crud.Scope, error) {
	return crud.Scope{"tenant_id": "tenant-a"}, nil
}

type sqlAuthorizer struct{}

func (sqlAuthorizer) Authorize(context.Context, crud.Principal, crud.ResourceKey, crud.Action, *crud.Record) error {
	return nil
}

type sqlAudit struct{}

func (sqlAudit) Append(context.Context, crud.AuditEvent) error { return nil }

type sqlTranslator struct{}

func (sqlTranslator) Message(_ context.Context, code crud.MessageCode, _ map[string]any) string {
	return string(code)
}

type sqlClock struct{}

func (sqlClock) Now() time.Time { return time.Time{} }
