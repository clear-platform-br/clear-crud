package sqladapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

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
