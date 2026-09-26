package sqladapter

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	crud "github.com/clear-platform-br/clear-crud"
)

// AuditTableDefinition maps an application-owned audit table. All columns are
// configured by the server; no table or column name comes from a request.
type AuditTableDefinition struct {
	Table               Identifier
	ScopeColumns        map[string]Identifier
	ResourceColumn      Identifier
	ActionColumn        Identifier
	RecordIDColumn      Identifier
	BeforeVersionColumn Identifier
	AfterVersionColumn  Identifier
	PrincipalIDColumn   Identifier
	PrincipalKindColumn Identifier
	OccurredAtColumn    Identifier
	CorrelationIDColumn *Identifier
}

// AuditSink persists metadata-only mutation audit events. When it receives a
// context created by SimpleTable.Within, it uses that exact transaction.
type AuditSink struct {
	db    *sql.DB
	table AuditTableDefinition
}

// NewAuditSink constructs a SQL audit sink. The database must be the same one
// used by the registered SimpleTable resources so transaction contexts can be
// shared by create, update, delete, children, and their audit events.
func NewAuditSink(db *sql.DB, table AuditTableDefinition) (*AuditSink, error) {
	if db == nil {
		return nil, fmt.Errorf("sqladapter: nil database")
	}
	if err := validateAuditTable(table); err != nil {
		return nil, err
	}
	return &AuditSink{db: db, table: cloneAuditTable(table)}, nil
}

// Append persists the event's technical metadata. Field values are not part of
// AuditEvent and therefore can never be recorded by this adapter.
func (sink *AuditSink) Append(ctx context.Context, event crud.AuditEvent) error {
	if sink == nil || sink.db == nil {
		return fmt.Errorf("sqladapter: nil audit sink")
	}
	columns, values, args, err := sink.values(event)
	if err != nil {
		return err
	}
	statement := "INSERT INTO " + quote(sink.table.Table) + " (" + strings.Join(columns, ", ") + ") VALUES (" + strings.Join(values, ", ") + ")"
	_, err = sink.executor(ctx).ExecContext(ctx, statement, args...)
	return err
}

func (sink *AuditSink) values(event crud.AuditEvent) ([]string, []string, []any, error) {
	keys := make([]string, 0, len(sink.table.ScopeColumns))
	for key := range sink.table.ScopeColumns {
		if event.Scope[key] == "" {
			return nil, nil, nil, fmt.Errorf("sqladapter: missing audit scope %q", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	columns := make([]string, 0, len(keys)+9)
	values := make([]string, 0, len(keys)+9)
	args := make([]any, 0, len(keys)+9)
	appendValue := func(column Identifier, value any) {
		columns = append(columns, quote(column))
		values = append(values, "?")
		args = append(args, value)
	}
	for _, key := range keys {
		appendValue(sink.table.ScopeColumns[key], event.Scope[key])
	}
	appendValue(sink.table.ResourceColumn, string(event.Resource))
	appendValue(sink.table.ActionColumn, string(event.Action))
	appendValue(sink.table.RecordIDColumn, string(event.RecordID))
	appendValue(sink.table.BeforeVersionColumn, int64(event.BeforeVersion))
	appendValue(sink.table.AfterVersionColumn, int64(event.AfterVersion))
	appendValue(sink.table.PrincipalIDColumn, event.Principal.ID)
	appendValue(sink.table.PrincipalKindColumn, event.Principal.Kind)
	appendValue(sink.table.OccurredAtColumn, event.OccurredAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"))
	if sink.table.CorrelationIDColumn != nil {
		appendValue(*sink.table.CorrelationIDColumn, event.CorrelationID)
	}
	return columns, values, args, nil
}

func (sink *AuditSink) executor(ctx context.Context) executor {
	if transaction, ok := ctx.Value(transactionKey{}).(*sql.Tx); ok {
		return transaction
	}
	return sink.db
}

func validateAuditTable(table AuditTableDefinition) error {
	if !validIdentifier(table.Table) || len(table.ScopeColumns) == 0 {
		return fmt.Errorf("sqladapter: audit table and scope columns are required")
	}
	columns := []Identifier{
		table.ResourceColumn, table.ActionColumn, table.RecordIDColumn,
		table.BeforeVersionColumn, table.AfterVersionColumn, table.PrincipalIDColumn,
		table.PrincipalKindColumn, table.OccurredAtColumn,
	}
	if table.CorrelationIDColumn != nil {
		columns = append(columns, *table.CorrelationIDColumn)
	}
	seen := make(map[Identifier]bool, len(columns)+len(table.ScopeColumns))
	for _, column := range columns {
		if !validIdentifier(column) || seen[column] {
			return fmt.Errorf("sqladapter: invalid audit column")
		}
		seen[column] = true
	}
	for key, column := range table.ScopeColumns {
		if key == "" || !validIdentifier(column) || seen[column] {
			return fmt.Errorf("sqladapter: invalid audit scope column")
		}
		seen[column] = true
	}
	return nil
}

func cloneAuditTable(table AuditTableDefinition) AuditTableDefinition {
	clone := table
	clone.ScopeColumns = make(map[string]Identifier, len(table.ScopeColumns))
	for key, value := range table.ScopeColumns {
		clone.ScopeColumns[key] = value
	}
	return clone
}
