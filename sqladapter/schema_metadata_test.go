package sqladapter

import (
	"context"
	"reflect"
	"testing"

	crud "github.com/clear-platform-br/clear-crud"
)

func TestSQLiteMetadataReadsEnumChecksForManualTables(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY,
        tenant_id TEXT NOT NULL,
        status TEXT NOT NULL CHECK (status IN ('new', 'review', 'closed')),
        state TEXT NOT NULL CHECK (state = 'or' OR state = 'closed'),
        priority INTEGER NOT NULL CHECK (priority IN (1, 2, 3)),
        notes TEXT CHECK (length(notes) > 0),
        version INTEGER NOT NULL
    )`)
	source, err := NewSimpleTable(database, TableDefinition{
		Table: mustIdentifier("contacts"), IDColumn: mustIdentifier("id"), VersionColumn: mustIdentifier("version"),
		ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")},
		Fields:       map[crud.FieldKey]Identifier{"status": mustIdentifier("status"), "state": mustIdentifier("state"), "priority": mustIdentifier("priority"), "notes": mustIdentifier("notes")},
	})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := source.Metadata(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := metadata.Fields["status"].Enum; len(got) != 3 || got[0] != "new" || got[2] != "closed" {
		t.Fatalf("status metadata = %#v", got)
	}
	if got := metadata.Fields["state"].Enum; len(got) != 2 || got[0] != "or" || got[1] != "closed" {
		t.Fatalf("state metadata = %#v", got)
	}
	if got := metadata.Fields["priority"].Enum; len(got) != 3 || got[0] != int64(1) || got[2] != int64(3) {
		t.Fatalf("priority metadata = %#v", got)
	}
	if _, ok := metadata.Fields["notes"]; ok {
		t.Fatal("arbitrary boolean check was exposed as enum metadata")
	}
}

func TestAutoTableUsesSchemaEnumValuesByDefault(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY,
        tenant_id TEXT NOT NULL,
        status TEXT NOT NULL CHECK (status IN ('new', 'review')),
        version INTEGER NOT NULL,
        archived INTEGER NOT NULL DEFAULT 0
    )`)
	definition, err := AutoTenantTable(context.Background(), database, mustIdentifier("contacts"), WithSoftDelete("archived"))
	if err != nil {
		t.Fatal(err)
	}
	field := fieldByKey(t, definition.Fields, "status")
	if field.Type != crud.FieldEnum || len(field.Enum) != 2 || field.Enum[0].Label != "new" {
		t.Fatalf("automatic enum field = %#v", field)
	}
}

func TestRegistryAppliesSourceEnumMetadataToManualDefinition(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE contacts (
        id INTEGER PRIMARY KEY, tenant_id TEXT NOT NULL,
        status TEXT NOT NULL CHECK (status IN ('new', 'review', 'closed')),
        version INTEGER NOT NULL
    )`)
	source, err := NewSimpleTable(database, TableDefinition{
		Table: mustIdentifier("contacts"), IDColumn: mustIdentifier("id"), VersionColumn: mustIdentifier("version"),
		ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")}, Fields: map[crud.FieldKey]Identifier{"status": mustIdentifier("status")},
	})
	if err != nil {
		t.Fatal(err)
	}
	definition := sqliteCRUDDefinition("contacts", source, []crud.Field{{Key: "status", Label: "crud.status", Type: crud.FieldString, Visible: true}}, []crud.FieldKey{"status"})
	registry := crud.NewRegistry()
	if err := registry.Register(context.Background(), definition); err != nil {
		t.Fatal(err)
	}
	registered, ok := registry.Get("contacts")
	if !ok {
		t.Fatal("registered definition not found")
	}
	field := fieldByKey(t, registered.Fields, "status")
	if field.Type != crud.FieldEnum || len(field.Enum) != 3 || field.Enum[0].Label != "new" {
		t.Fatalf("registered enum field = %#v", field)
	}

	subset := definition
	subset.Fields[0].Type = crud.FieldEnum
	subset.Fields[0].Enum = []crud.Option{{Value: "new", Label: "Novo"}, {Value: "review", Label: "Em análise"}}
	otherRegistry := crud.NewRegistry()
	if err := otherRegistry.Register(context.Background(), subset); err != nil {
		t.Fatal(err)
	}
	registered, _ = otherRegistry.Get("contacts")
	field = fieldByKey(t, registered.Fields, "status")
	if len(field.Enum) != 2 || field.Enum[1].Label != "Em análise" {
		t.Fatalf("explicit enum subset was not preserved = %#v", field.Enum)
	}

	invalid := subset
	invalid.Fields[0].Enum = []crud.Option{{Value: "cancelled", Label: "Cancelado"}}
	if err := crud.NewRegistry().Register(context.Background(), invalid); err == nil {
		t.Fatal("enum value outside schema was accepted")
	}
}

func TestSQLiteEnumParserIgnoresUnsafeOrUnboundedChecks(t *testing.T) {
	constraints, err := sqliteEnumConstraints(`CREATE TABLE sample (
        status TEXT CHECK (status NOT IN ('deleted')),
        other TEXT CHECK (other IN (lower('x'), 'y')),
        quantity INTEGER CHECK (quantity >= 1)
    )`)
	if err != nil {
		t.Fatal(err)
	}
	if len(constraints) != 0 {
		t.Fatalf("unsafe checks inferred as enum = %#v", constraints)
	}
}

func TestSQLiteEnumParserHandlesSupportedLiteralsAndRejectsMalformedExpressions(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		value crud.Value
		ok    bool
	}{
		{name: "single quoted", input: `'it''s'`, value: "it's", ok: true},
		{name: "double quoted", input: `"a""b"`, value: `a"b`, ok: true},
		{name: "integer", input: `-42`, value: int64(-42), ok: true},
		{name: "decimal", input: `1.50`, value: "1.50", ok: true},
		{name: "true", input: `TRUE`, value: true, ok: true},
		{name: "false", input: `FALSE`, value: false, ok: true},
		{name: "unknown", input: `CURRENT_DATE`, ok: false},
		{name: "overflow", input: `999999999999999999999999`, ok: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := sqliteLiteral(test.input)
			if ok != test.ok || ok && !reflect.DeepEqual(got, test.value) {
				t.Fatalf("sqliteLiteral(%q) = %#v, %v; want %#v, %v", test.input, got, ok, test.value, test.ok)
			}
		})
	}

	for _, test := range []struct {
		name  string
		input string
		ok    bool
	}{
		{name: "quoted identifier", input: `"status" IN ('new')`, ok: true},
		{name: "invalid identifier", input: `status.value IN ('new')`, ok: false},
		{name: "trailing expression", input: `status IN ('new') AND 1`, ok: false},
		{name: "missing values", input: `status IN ()`, ok: false},
		{name: "different equality columns", input: `status = 'new' OR state = 'closed'`, ok: false},
		{name: "not equality", input: `status <> 'new'`, ok: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, ok := sqliteEnumExpression(test.input)
			if ok != test.ok {
				t.Fatalf("sqliteEnumExpression(%q) accepted=%v, want %v", test.input, ok, test.ok)
			}
		})
	}

	constraints, err := sqliteEnumConstraints(`CREATE TABLE sample (
        status TEXT CHECK (status IN ('new', 'review'))
        CHECK (status IN ('review', 'closed'))
    )`)
	if err != nil {
		t.Fatal(err)
	}
	if got := constraints["status"]; len(got) != 1 || got[0] != "review" {
		t.Fatalf("intersected enum values = %#v", got)
	}
	if _, err := sqliteEnumConstraints(`CREATE TABLE sample (status TEXT CHECK (status IN ('new')) CHECK (status IN ('closed')))`); err == nil {
		t.Fatal("incompatible enum checks accepted")
	}

	if got := sqliteCheckExpressions(`CREATE TABLE sample (note TEXT DEFAULT 'CHECK (ignored)', -- CHECK (value IN ('bad'))
        /* CHECK (value IN ('also-bad')) */ value TEXT CHECK (value IN ('a'))`); len(got) != 1 {
		t.Fatalf("malformed/quoted CHECK expressions = %#v", got)
	}
}

func TestSQLiteMetadataLabelsAndScannerBranches(t *testing.T) {
	for value, want := range map[crud.Value]string{nil: "∅", "": "∅", "value": "value", true: "true", false: "false", int64(-3): "-3"} {
		if got := schemaEnumLabel(value); got != want {
			t.Fatalf("schemaEnumLabel(%#v) = %q, want %q", value, got, want)
		}
	}
	if got, ok := sqliteColumnIdentifier("`status`"); !ok || got != "status" {
		t.Fatalf("quoted identifier = %q", got)
	}
	if _, ok := sqliteColumnIdentifier("status.value"); ok {
		t.Fatal("qualified identifier accepted")
	}
	if got := findSQLKeyword(`'OR' OR status = 'x'`, "OR"); got < 0 {
		t.Fatal("unquoted keyword was not found")
	}
	if findSQLKeyword(`'OR'`, "OR") >= 0 {
		t.Fatal("keyword inside string was found")
	}
}

func TestSQLiteMetadataRejectsUninitializedSource(t *testing.T) {
	var source *SimpleTable
	if _, err := source.Metadata(context.Background()); err == nil {
		t.Fatal("uninitialized metadata source accepted")
	}
}

func TestSQLiteMetadataHandlesMissingAndClosedDatabase(t *testing.T) {
	database := newAutoDatabase(t, `CREATE TABLE present (id INTEGER PRIMARY KEY, version INTEGER NOT NULL, value TEXT)`)
	source, err := NewSimpleTable(database, TableDefinition{
		Table: mustIdentifier("missing"), IDColumn: mustIdentifier("id"), VersionColumn: mustIdentifier("version"),
		ScopeColumns: map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")}, Fields: map[crud.FieldKey]Identifier{"value": mustIdentifier("value")},
	})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := source.Metadata(context.Background())
	if err != nil || len(metadata.Fields) != 0 {
		t.Fatalf("missing table metadata = %#v, %v", metadata, err)
	}
}
