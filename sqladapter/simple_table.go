// Package sqladapter provides SQL persistence adapters for clear.crud.
// It imports database/sql but never imports or selects a database driver.
package sqladapter

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	crud "github.com/clear-platform-br/clear-crud"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Identifier is a server-validated SQL identifier. It is never built from an
// HTTP value, field value, or tenant value.
type Identifier string

// NewIdentifier validates an SQL table or column identifier for simple_table.
func NewIdentifier(value string) (Identifier, error) {
	if !identifierPattern.MatchString(value) {
		return "", fmt.Errorf("sqladapter: invalid identifier %q", value)
	}
	return Identifier(value), nil
}

// TableDefinition maps one conventional table to a clear.crud DataSource.
type TableDefinition struct {
	Table         Identifier
	IDColumn      Identifier
	VersionColumn Identifier
	ScopeColumns  map[string]Identifier
	ArchiveColumn *Identifier
	Fields        map[crud.FieldKey]Identifier
}

// SimpleTable implements a SQLite simple_table resource and its UnitOfWork.
// Its config is server-owned and validated before it can serve a request.
type SimpleTable struct {
	db    *sql.DB
	table TableDefinition
}

// NewSimpleTable constructs a SQLite adapter without importing a SQLite driver.
func NewSimpleTable(db *sql.DB, table TableDefinition) (*SimpleTable, error) {
	if db == nil {
		return nil, fmt.Errorf("sqladapter: nil database")
	}
	if err := validateTable(table); err != nil {
		return nil, err
	}
	return &SimpleTable{db: db, table: cloneTable(table)}, nil
}

// Capabilities reports guarantees supplied by SQLite simple_table.
func (source *SimpleTable) Capabilities(context.Context) crud.Capabilities {
	return crud.Capabilities{
		crud.CapabilityTotalCount:     {},
		crud.CapabilityOffsetPage:     {},
		crud.CapabilityAtomicVersion:  {},
		crud.CapabilityUnitOfWork:     {},
		crud.CapabilityContainsSearch: {},
		crud.CapabilityArchive:        {},
		crud.CapabilityHardDelete:     {},
	}
}

// Within starts a transaction and makes it available to source operations
// through their context. A returned error rolls all mutations back.
func (source *SimpleTable) Within(ctx context.Context, operation func(context.Context) error) error {
	transaction, err := source.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	transactionContext := context.WithValue(ctx, transactionKey{}, transaction)
	if err := operation(transactionContext); err != nil {
		_ = transaction.Rollback()
		return err
	}
	return transaction.Commit()
}

// List returns records from exactly one trusted scope.
func (source *SimpleTable) List(ctx context.Context, scope crud.Scope, query crud.Query) (crud.Page, error) {
	where, args, err := source.where(scope, query)
	if err != nil {
		return crud.Page{}, err
	}
	columns := source.selectColumns()
	statement := "SELECT " + columns + " FROM " + quote(source.table.Table) + " WHERE " + where + source.order(query.Sort)
	args = append(args, int64(query.Page.Size), int64((query.Page.Number-1)*uint64(query.Page.Size)))
	statement += " LIMIT ? OFFSET ?"
	rows, err := source.executor(ctx).QueryContext(ctx, statement, args...)
	if err != nil {
		return crud.Page{}, err
	}
	defer rows.Close()
	records, err := source.records(rows)
	if err != nil {
		return crud.Page{}, err
	}
	if err := rows.Err(); err != nil {
		return crud.Page{}, err
	}
	countStatement := "SELECT COUNT(*) FROM " + quote(source.table.Table) + " WHERE " + where
	var count uint64
	if err := source.executor(ctx).QueryRowContext(ctx, countStatement, args[:len(args)-2]...).Scan(&count); err != nil {
		return crud.Page{}, err
	}
	return crud.Page{Records: records, Page: query.Page.Number, Size: query.Page.Size, Total: &count}, nil
}

// Get loads one record in the supplied trusted scope.
func (source *SimpleTable) Get(ctx context.Context, scope crud.Scope, id crud.RecordID) (crud.Record, error) {
	where, args, err := source.scopeWhere(scope)
	if err != nil {
		return crud.Record{}, err
	}
	args = append([]any{id}, args...)
	statement := "SELECT " + source.selectColumns() + " FROM " + quote(source.table.Table) + " WHERE " + quote(source.table.IDColumn) + " = ? AND " + where + source.activeWhere()
	row := source.executor(ctx).QueryRowContext(ctx, statement, args...)
	record, err := source.scanRecord(row)
	if err == sql.ErrNoRows {
		return crud.Record{}, public(crud.ErrorNotFound)
	}
	return record, err
}

// Create inserts a full mutation and reads the generated opaque record ID.
func (source *SimpleTable) Create(ctx context.Context, scope crud.Scope, mutation crud.Mutation) (crud.Record, error) {
	columns, values, args, err := source.mutationValues(scope, mutation)
	if err != nil {
		return crud.Record{}, err
	}
	columns = append(columns, quote(source.table.VersionColumn))
	values = append(values, "?")
	args = append(args, int64(1))
	statement := "INSERT INTO " + quote(source.table.Table) + " (" + strings.Join(columns, ", ") + ") VALUES (" + strings.Join(values, ", ") + ")"
	result, err := source.executor(ctx).ExecContext(ctx, statement, args...)
	if err != nil {
		return crud.Record{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return crud.Record{}, err
	}
	return source.Get(ctx, scope, crud.RecordID(strconv.FormatInt(id, 10)))
}

// Update replaces a full mutation only while its version matches.
func (source *SimpleTable) Update(ctx context.Context, scope crud.Scope, id crud.RecordID, version crud.Version, mutation crud.Mutation) (crud.Record, error) {
	_, _, allArgs, err := source.mutationValues(scope, mutation)
	if err != nil {
		return crud.Record{}, err
	}
	args := append([]any(nil), allArgs[len(source.table.ScopeColumns):]...)
	assignments := source.assignments()
	where, scopeArgs, err := source.scopeWhere(scope)
	if err != nil {
		return crud.Record{}, err
	}
	args = append(args, id)
	args = append(args, scopeArgs...)
	args = append(args, version)
	statement := "UPDATE " + quote(source.table.Table) + " SET " + assignments + " WHERE " + quote(source.table.IDColumn) + " = ? AND " + where + source.activeWhere() + " AND " + quote(source.table.VersionColumn) + " = ?"
	result, err := source.executor(ctx).ExecContext(ctx, statement, args...)
	if err != nil {
		return crud.Record{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return crud.Record{}, err
	}
	if changed != 1 {
		return crud.Record{}, public(crud.ErrorConflict)
	}
	return source.Get(ctx, scope, id)
}

// Delete archives or physically deletes one versioned record.
func (source *SimpleTable) Delete(ctx context.Context, scope crud.Scope, id crud.RecordID, version crud.Version, mode crud.DeleteMode) error {
	if mode == crud.DeleteModeNone {
		return public(crud.ErrorDeleteRestricted)
	}
	where, args, err := source.scopeWhere(scope)
	if err != nil {
		return err
	}
	args = append([]any{id}, args...)
	args = append(args, version)
	var statement string
	switch mode {
	case crud.DeleteModeArchive:
		if source.table.ArchiveColumn == nil {
			return public(crud.ErrorDeleteRestricted)
		}
		statement = "UPDATE " + quote(source.table.Table) + " SET " + quote(*source.table.ArchiveColumn) + " = 1, " + quote(source.table.VersionColumn) + " = " + quote(source.table.VersionColumn) + " + 1 WHERE " + quote(source.table.IDColumn) + " = ? AND " + where + source.activeWhere() + " AND " + quote(source.table.VersionColumn) + " = ?"
	case crud.DeleteModeHardDelete:
		statement = "DELETE FROM " + quote(source.table.Table) + " WHERE " + quote(source.table.IDColumn) + " = ? AND " + where + source.activeWhere() + " AND " + quote(source.table.VersionColumn) + " = ?"
	default:
		return public(crud.ErrorDeleteRestricted)
	}
	result, err := source.executor(ctx).ExecContext(ctx, statement, args...)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return public(crud.ErrorConflict)
	}
	return nil
}

// Lookup is not part of SQLite simple_table v0.1; resources with lookups use
// the dedicated SQL resource adapter added in a later increment.
func (source *SimpleTable) Lookup(context.Context, crud.Scope, crud.LookupQuery) (crud.LookupPage, error) {
	return crud.LookupPage{}, public(crud.ErrorInvalidRequest)
}

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type transactionKey struct{}

func (source *SimpleTable) executor(ctx context.Context) executor {
	if transaction, ok := ctx.Value(transactionKey{}).(*sql.Tx); ok {
		return transaction
	}
	return source.db
}

func validateTable(table TableDefinition) error {
	if table.Table == "" || table.IDColumn == "" || table.VersionColumn == "" || len(table.ScopeColumns) == 0 || len(table.Fields) == 0 {
		return fmt.Errorf("sqladapter: table, id, version, scope and fields are required")
	}
	for key, column := range table.ScopeColumns {
		if key == "" || !validIdentifier(column) {
			return fmt.Errorf("sqladapter: invalid scope column")
		}
	}
	for key, column := range table.Fields {
		if key == "" || !validIdentifier(column) {
			return fmt.Errorf("sqladapter: invalid field column")
		}
	}
	if !validIdentifier(table.Table) || !validIdentifier(table.IDColumn) || !validIdentifier(table.VersionColumn) || table.ArchiveColumn != nil && !validIdentifier(*table.ArchiveColumn) {
		return fmt.Errorf("sqladapter: invalid table mapping")
	}
	return nil
}

func validIdentifier(identifier Identifier) bool {
	return identifierPattern.MatchString(string(identifier))
}

func cloneTable(table TableDefinition) TableDefinition {
	clone := table
	clone.ScopeColumns = make(map[string]Identifier, len(table.ScopeColumns))
	for key, value := range table.ScopeColumns {
		clone.ScopeColumns[key] = value
	}
	clone.Fields = make(map[crud.FieldKey]Identifier, len(table.Fields))
	for key, value := range table.Fields {
		clone.Fields[key] = value
	}
	return clone
}

func (source *SimpleTable) where(scope crud.Scope, query crud.Query) (string, []any, error) {
	where, args, err := source.scopeWhere(scope)
	if err != nil {
		return "", nil, err
	}
	where += source.activeWhere()
	for _, filter := range query.Filters {
		column, ok := source.table.Fields[filter.Field]
		if !ok {
			return "", nil, public(crud.ErrorInvalidRequest)
		}
		operator, ok := filterOperator(filter.Operator)
		if !ok {
			return "", nil, public(crud.ErrorInvalidRequest)
		}
		where += " AND " + quote(column) + " " + operator
		if filter.Operator != crud.FilterIsNull {
			args = append(args, filter.Value)
		}
	}
	if query.Search != "" {
		fields := source.fieldColumns()
		if len(fields) == 0 {
			return "", nil, public(crud.ErrorInvalidRequest)
		}
		terms := make([]string, 0, len(fields))
		for _, column := range fields {
			terms = append(terms, "LOWER(CAST("+quote(column)+" AS TEXT)) LIKE LOWER(?)")
			args = append(args, "%"+query.Search+"%")
		}
		where += " AND (" + strings.Join(terms, " OR ") + ")"
	}
	return where, args, nil
}

func (source *SimpleTable) scopeWhere(scope crud.Scope) (string, []any, error) {
	keys := make([]string, 0, len(source.table.ScopeColumns))
	for key := range source.table.ScopeColumns {
		if scope[key] == "" {
			return "", nil, public(crud.ErrorForbidden)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, quote(source.table.ScopeColumns[key])+" = ?")
		args = append(args, scope[key])
	}
	return strings.Join(parts, " AND "), args, nil
}

func (source *SimpleTable) activeWhere() string {
	if source.table.ArchiveColumn == nil {
		return ""
	}
	return " AND (" + quote(*source.table.ArchiveColumn) + " IS NULL OR " + quote(*source.table.ArchiveColumn) + " = 0)"
}

func (source *SimpleTable) selectColumns() string {
	columns := []string{quote(source.table.IDColumn), quote(source.table.VersionColumn)}
	for _, field := range source.fieldColumns() {
		columns = append(columns, quote(field))
	}
	return strings.Join(columns, ", ")
}

func (source *SimpleTable) fieldColumns() []Identifier {
	keys := make([]string, 0, len(source.table.Fields))
	for key := range source.table.Fields {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	columns := make([]Identifier, 0, len(keys))
	for _, key := range keys {
		columns = append(columns, source.table.Fields[crud.FieldKey(key)])
	}
	return columns
}

func (source *SimpleTable) fieldKeys() []crud.FieldKey {
	keys := make([]string, 0, len(source.table.Fields))
	for key := range source.table.Fields {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	fields := make([]crud.FieldKey, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, crud.FieldKey(key))
	}
	return fields
}

func (source *SimpleTable) order(sortTerms []crud.Sort) string {
	terms := make([]string, 0, len(sortTerms))
	for _, term := range sortTerms {
		column := source.table.IDColumn
		if term.Field != "id" {
			column = source.table.Fields[term.Field]
		}
		if column == "" {
			return ""
		}
		direction := "ASC"
		if term.Direction == crud.SortDescending {
			direction = "DESC"
		}
		terms = append(terms, quote(column)+" "+direction)
	}
	if len(terms) == 0 {
		terms = append(terms, quote(source.table.IDColumn)+" ASC")
	}
	return " ORDER BY " + strings.Join(terms, ", ")
}

func (source *SimpleTable) mutationValues(scope crud.Scope, mutation crud.Mutation) ([]string, []string, []any, error) {
	where, scopeArgs, err := source.scopeWhere(scope)
	if err != nil {
		return nil, nil, nil, err
	}
	_ = where
	columns := make([]string, 0, len(scopeArgs)+len(source.table.Fields))
	values := make([]string, 0, cap(columns))
	args := make([]any, 0, cap(columns))
	scopeKeys := make([]string, 0, len(source.table.ScopeColumns))
	for key := range source.table.ScopeColumns {
		scopeKeys = append(scopeKeys, key)
	}
	sort.Strings(scopeKeys)
	for _, key := range scopeKeys {
		columns = append(columns, quote(source.table.ScopeColumns[key]))
		values = append(values, "?")
		args = append(args, scope[key])
	}
	for _, key := range source.fieldKeys() {
		columns = append(columns, quote(source.table.Fields[key]))
		values = append(values, "?")
		args = append(args, mutation.Fields[key])
	}
	return columns, values, args, nil
}

func (source *SimpleTable) assignments() string {
	parts := make([]string, 0, len(source.table.Fields)+1)
	for _, key := range source.fieldKeys() {
		parts = append(parts, quote(source.table.Fields[key])+" = ?")
	}
	parts = append(parts, quote(source.table.VersionColumn)+" = "+quote(source.table.VersionColumn)+" + 1")
	return strings.Join(parts, ", ")
}

func (source *SimpleTable) records(rows *sql.Rows) ([]crud.Record, error) {
	var records []crud.Record
	for rows.Next() {
		record, err := source.scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

type scanner interface{ Scan(...any) error }

func (source *SimpleTable) scanRecord(row scanner) (crud.Record, error) {
	var id any
	var version int64
	keys := source.fieldKeys()
	values := make([]any, len(keys))
	destinations := make([]any, 0, len(keys)+2)
	destinations = append(destinations, &id, &version)
	for index := range values {
		destinations = append(destinations, &values[index])
	}
	if err := row.Scan(destinations...); err != nil {
		return crud.Record{}, err
	}
	fields := make(crud.Fields, len(keys))
	for index, key := range keys {
		fields[key] = sqlValue(values[index])
	}
	return crud.Record{ID: crud.RecordID(stringValue(id)), Version: crud.Version(version), Fields: fields}, nil
}

func filterOperator(operator crud.FilterOperator) (string, bool) {
	switch operator {
	case crud.FilterEqual:
		return "= ?", true
	case crud.FilterNotEqual:
		return "<> ?", true
	case crud.FilterLessThan:
		return "< ?", true
	case crud.FilterLessOrEqual:
		return "<= ?", true
	case crud.FilterGreaterThan:
		return "> ?", true
	case crud.FilterGreaterOrEqual:
		return ">= ?", true
	case crud.FilterContains:
		return "LIKE '%' || ? || '%'", true
	case crud.FilterPrefix:
		return "LIKE ? || '%'", true
	case crud.FilterIsNull:
		return "IS NULL", true
	default:
		return "", false
	}
}

func quote(identifier Identifier) string { return `"` + string(identifier) + `"` }
func public(code crud.ErrorCode) *crud.Error {
	return &crud.Error{Code: code, Message: crud.MessageCode("crud.error." + string(code))}
}
func stringValue(value any) string {
	if bytes, ok := value.([]byte); ok {
		return string(bytes)
	}
	return fmt.Sprint(value)
}
func sqlValue(value any) any {
	if bytes, ok := value.([]byte); ok {
		return string(bytes)
	}
	return value
}
