package sqladapter

import (
	"reflect"
	"testing"

	crud "github.com/clear-platform-br/clear-crud"
)

func TestSQLiteStaticDefaultAcceptsSafeLiterals(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		fieldType  crud.FieldType
		want       crud.Value
		ok         bool
	}{
		{name: "quoted string", expression: `'new'`, fieldType: crud.FieldString, want: "new", ok: true},
		{name: "escaped quoted string", expression: `'O''Reilly'`, fieldType: crud.FieldString, want: "O'Reilly", ok: true},
		{name: "wrapped string", expression: `('new')`, fieldType: crud.FieldString, want: "new", ok: true},
		{name: "integer", expression: `-7`, fieldType: crud.FieldInteger, want: int64(-7), ok: true},
		{name: "quoted integer", expression: `'7'`, fieldType: crud.FieldInteger, want: int64(7), ok: true},
		{name: "boolean true", expression: `TRUE`, fieldType: crud.FieldBoolean, want: true, ok: true},
		{name: "boolean false", expression: `0`, fieldType: crud.FieldBoolean, want: false, ok: true},
		{name: "quoted boolean", expression: `'false'`, fieldType: crud.FieldBoolean, want: false, ok: true},
		{name: "decimal", expression: `1.25`, fieldType: crud.FieldDecimal, want: "1.25", ok: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := sqliteStaticDefault(test.expression, test.fieldType)
			if ok != test.ok || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("sqliteStaticDefault(%q, %q) = %#v, %t; want %#v, %t", test.expression, test.fieldType, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestSQLiteStaticDefaultRejectsDynamicOrUnsafeExpressions(t *testing.T) {
	for _, expression := range []string{"", "NULL", "CURRENT_TIMESTAMP", "(CURRENT_DATE)", "datetime('now')", "(1) trailing", "'unterminated"} {
		if got, ok := sqliteStaticDefault(expression, crud.FieldString); ok || got != nil {
			t.Fatalf("sqliteStaticDefault(%q) = %#v, %t; want no default", expression, got, ok)
		}
	}
	if got := unwrapSQLiteDefault("((1))"); got != "1" {
		t.Fatalf("unwrap nested default = %q", got)
	}
	if sqliteDefaultParensEncloseAll("(1) trailing") {
		t.Fatal("trailing expression accepted as one parenthesized literal")
	}
	if sqliteDefaultParensEncloseAll("(1") {
		t.Fatal("unclosed expression accepted as one parenthesized literal")
	}
	if value, ok := sqliteDefaultBoolean("maybe"); ok || value != nil {
		t.Fatalf("invalid boolean = %#v, %t", value, ok)
	}
	if value, ok := sqliteDefaultQuotedString("not quoted"); ok || value != "" {
		t.Fatalf("unquoted string = %q, %t", value, ok)
	}
}
