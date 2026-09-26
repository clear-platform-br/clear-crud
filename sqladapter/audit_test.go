package sqladapter

import (
	"context"
	"database/sql"
	"testing"
	"time"

	crud "github.com/clear-platform-br/clear-crud"
	_ "modernc.org/sqlite"
)

func TestAuditSinkPersistsMetadataInSimpleTableTransaction(t *testing.T) {
	database := openAuditDatabase(t, `
CREATE TABLE contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    version INTEGER NOT NULL,
    archived INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id TEXT NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    record_id TEXT NOT NULL,
    before_version INTEGER NOT NULL,
    after_version INTEGER NOT NULL,
    principal_id TEXT NOT NULL,
    principal_kind TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    correlation_id TEXT NOT NULL
)`)
	contacts, err := NewSimpleTable(database, sqliteTable())
	if err != nil {
		t.Fatal(err)
	}
	audit, err := NewAuditSink(database, sqliteAuditTable())
	if err != nil {
		t.Fatal(err)
	}
	scope := crud.Scope{"tenant_id": "tenant-a"}
	when := time.Date(2026, time.September, 26, 12, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
	err = contacts.Within(context.Background(), func(tx context.Context) error {
		record, err := contacts.Create(tx, scope, crud.Mutation{Fields: crud.Fields{"name": "Ana"}})
		if err != nil {
			return err
		}
		return audit.Append(tx, crud.AuditEvent{
			Resource: "contacts", Action: crud.ActionCreate, RecordID: record.ID, AfterVersion: record.Version,
			Principal: crud.Principal{ID: "operator-a", Kind: "operator"}, Scope: scope,
			OccurredAt: when, CorrelationID: "request-81",
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	var tenantID, resource, action, recordID, principalID, principalKind, occurredAt, correlation string
	var before, after int64
	err = database.QueryRow(`SELECT tenant_id, resource, action, record_id, before_version, after_version, principal_id, principal_kind, occurred_at, correlation_id FROM audit_events`).Scan(
		&tenantID, &resource, &action, &recordID, &before, &after, &principalID, &principalKind, &occurredAt, &correlation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if tenantID != "tenant-a" || resource != "contacts" || action != "create" || recordID == "" || before != 0 || after != 1 || principalID != "operator-a" || principalKind != "operator" || occurredAt != "2026-09-26T15:00:00Z" || correlation != "request-81" {
		t.Fatalf("audit row = tenant:%q resource:%q action:%q record:%q before:%d after:%d principal:%q kind:%q at:%q correlation:%q", tenantID, resource, action, recordID, before, after, principalID, principalKind, occurredAt, correlation)
	}
}

func TestAuditSinkFailureRollsBackSimpleTableTransaction(t *testing.T) {
	database := openAuditDatabase(t, `
CREATE TABLE contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    version INTEGER NOT NULL,
    archived INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE audit_events (
    tenant_id TEXT NOT NULL,
    resource TEXT NOT NULL CHECK(resource = 'allowed'),
    action TEXT NOT NULL,
    record_id TEXT NOT NULL,
    before_version INTEGER NOT NULL,
    after_version INTEGER NOT NULL,
    principal_id TEXT NOT NULL,
    principal_kind TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    correlation_id TEXT NOT NULL
)`)
	contacts, err := NewSimpleTable(database, sqliteTable())
	if err != nil {
		t.Fatal(err)
	}
	audit, err := NewAuditSink(database, sqliteAuditTable())
	if err != nil {
		t.Fatal(err)
	}
	err = contacts.Within(context.Background(), func(tx context.Context) error {
		record, err := contacts.Create(tx, crud.Scope{"tenant_id": "tenant-a"}, crud.Mutation{Fields: crud.Fields{"name": "Ana"}})
		if err != nil {
			return err
		}
		return audit.Append(tx, crud.AuditEvent{
			Resource: "contacts", Action: crud.ActionCreate, RecordID: record.ID, AfterVersion: record.Version,
			Principal: crud.Principal{ID: "operator-a", Kind: "operator"}, Scope: crud.Scope{"tenant_id": "tenant-a"}, OccurredAt: time.Now(),
		})
	})
	if err == nil {
		t.Fatal("transaction with rejected audit event succeeded")
	}
	var contactCount int
	if err := database.QueryRow(`SELECT COUNT(*) FROM contacts`).Scan(&contactCount); err != nil {
		t.Fatal(err)
	}
	if contactCount != 0 {
		t.Fatalf("contacts committed after audit failure: %d", contactCount)
	}
}

func TestNewAuditSinkRejectsInvalidConfiguration(t *testing.T) {
	database := openAuditDatabase(t, "")
	if _, err := NewAuditSink(nil, sqliteAuditTable()); err == nil {
		t.Fatal("nil database accepted")
	}
	bad := sqliteAuditTable()
	bad.ScopeColumns = nil
	if _, err := NewAuditSink(database, bad); err == nil {
		t.Fatal("missing scope accepted")
	}
	bad = sqliteAuditTable()
	bad.ActionColumn = bad.ResourceColumn
	if _, err := NewAuditSink(database, bad); err == nil {
		t.Fatal("duplicate audit columns accepted")
	}
	bad = sqliteAuditTable()
	bad.ScopeColumns = map[string]Identifier{"tenant_id": bad.ResourceColumn}
	if _, err := NewAuditSink(database, bad); err == nil {
		t.Fatal("overlapping scope column accepted")
	}
	bad = sqliteAuditTable()
	bad.Table = "audit-events"
	if _, err := NewAuditSink(database, bad); err == nil {
		t.Fatal("unsafe table identifier accepted")
	}
}

func TestAuditSinkRejectsMissingScope(t *testing.T) {
	database := openAuditDatabase(t, "")
	audit, err := NewAuditSink(database, sqliteAuditTable())
	if err != nil {
		t.Fatal(err)
	}
	if err := audit.Append(context.Background(), crud.AuditEvent{}); err == nil {
		t.Fatal("event without required scope accepted")
	}
}

func openAuditDatabase(t testing.TB, schema string) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if schema != "" {
		if _, err := database.Exec(schema); err != nil {
			database.Close()
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func sqliteAuditTable() AuditTableDefinition {
	return AuditTableDefinition{
		Table:               mustIdentifier("audit_events"),
		ScopeColumns:        map[string]Identifier{"tenant_id": mustIdentifier("tenant_id")},
		ResourceColumn:      mustIdentifier("resource"),
		ActionColumn:        mustIdentifier("action"),
		RecordIDColumn:      mustIdentifier("record_id"),
		BeforeVersionColumn: mustIdentifier("before_version"),
		AfterVersionColumn:  mustIdentifier("after_version"),
		PrincipalIDColumn:   mustIdentifier("principal_id"),
		PrincipalKindColumn: mustIdentifier("principal_kind"),
		OccurredAtColumn:    mustIdentifier("occurred_at"),
		CorrelationIDColumn: identifierPointer(mustIdentifier("correlation_id")),
	}
}
