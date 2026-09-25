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
		List:         crud.ListDefinition{Columns: columns, Searchable: columns, Sortable: columns, DefaultSort: []crud.Sort{{Field: columns[0], Direction: crud.SortAscending}}, Pagination: crud.PaginationDefinition{Mode: crud.PageModeOffset, DefaultSize: 25, AllowedSizes: []uint16{25, 50, 100}, Total: true}},
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
