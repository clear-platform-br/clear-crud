package sqladapter

import (
	"context"
	"database/sql"
	"strings"

	crud "github.com/clear-platform-br/clear-crud"
)

// List returns a bounded page from the registered read model.
func (source *ReadModelSource) List(ctx context.Context, scope crud.Scope, query crud.Query) (crud.Page, error) {
	if source == nil || query.IncludeArchived || query.Page.Mode != crud.PageModeOffset || query.Page.Number == 0 || query.Page.Size == 0 || query.Page.Size > 100 {
		return crud.Page{}, public(crud.ErrorInvalidRequest)
	}
	where, args, err := source.where(scope, query)
	if err != nil {
		return crud.Page{}, err
	}
	order, err := source.order(query.Sort)
	if err != nil {
		return crud.Page{}, err
	}
	statement := "SELECT " + source.selectList + " FROM (" + source.query + ") AS " + quoteRaw(readModelAlias) + " WHERE " + where + order + " LIMIT ? OFFSET ?"
	listArgs := append([]any(nil), args...)
	listArgs = append(listArgs, int64(query.Page.Size), int64((query.Page.Number-1)*uint64(query.Page.Size)))
	rows, err := source.db.QueryContext(ctx, statement, listArgs...)
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
	countStatement := "SELECT COUNT(*) FROM (" + source.query + ") AS " + quoteRaw(readModelAlias) + " WHERE " + where
	var total uint64
	if err := source.db.QueryRowContext(ctx, countStatement, args...).Scan(&total); err != nil {
		return crud.Page{}, err
	}
	return crud.Page{Records: records, Page: query.Page.Number, Size: query.Page.Size, Total: &total}, nil
}

// Get returns one read-model row in the supplied trusted scope.
func (source *ReadModelSource) Get(ctx context.Context, scope crud.Scope, id crud.RecordID) (crud.Record, error) {
	if source == nil || id == "" {
		return crud.Record{}, public(crud.ErrorInvalidRequest)
	}
	where, args, err := source.scopeWhere(scope)
	if err != nil {
		return crud.Record{}, err
	}
	where += " AND " + quoteRaw(readModelAlias) + "." + quote(source.idColumn) + " = ?"
	args = append(args, id)
	statement := "SELECT " + source.selectList + " FROM (" + source.query + ") AS " + quoteRaw(readModelAlias) + " WHERE " + where + " LIMIT 1"
	record, err := source.scanRecord(source.db.QueryRowContext(ctx, statement, args...))
	if err == sql.ErrNoRows {
		return crud.Record{}, public(crud.ErrorNotFound)
	}
	return record, err
}

func (source *ReadModelSource) where(scope crud.Scope, query crud.Query) (string, []any, error) {
	where, args, err := source.scopeWhere(scope)
	if err != nil {
		return "", nil, err
	}
	for _, filter := range query.Filters {
		alias, ok := source.fieldByKey[filter.Field]
		if !ok {
			return "", nil, public(crud.ErrorInvalidRequest)
		}
		column := quoteRaw(readModelAlias) + "." + quote(alias)
		if filter.Operator == crud.FilterIn {
			if len(filter.Values) == 0 || len(filter.Values) > 100 {
				return "", nil, public(crud.ErrorInvalidRequest)
			}
			where += " AND " + column + " IN (" + lookupPlaceholders(len(filter.Values)) + ")"
			args = append(args, filter.Values...)
			continue
		}
		op, ok := filterOperator(filter.Operator)
		if !ok {
			return "", nil, public(crud.ErrorInvalidRequest)
		}
		where += " AND " + column + " " + op
		if filter.Operator != crud.FilterIsNull {
			args = append(args, filter.Value)
		}
	}
	if query.Search == "" {
		return where, args, nil
	}
	if len(source.searchable) == 0 {
		return "", nil, public(crud.ErrorInvalidRequest)
	}
	terms := make([]string, 0, len(source.searchable))
	for _, field := range source.searchable {
		alias, ok := source.fieldByKey[field]
		if !ok {
			return "", nil, public(crud.ErrorInvalidRequest)
		}
		terms = append(terms, "LOWER(CAST("+quoteRaw(readModelAlias)+"."+quote(alias)+" AS TEXT)) LIKE LOWER(?)")
		args = append(args, "%"+query.Search+"%")
	}
	return where + " AND (" + strings.Join(terms, " OR ") + ")", args, nil
}

func (source *ReadModelSource) scopeWhere(scope crud.Scope) (string, []any, error) {
	parts := []string{"1=1"}
	args := make([]any, 0, len(source.scopes))
	for _, column := range source.scopes {
		value := scope[column.key]
		if value == "" {
			return "", nil, public(crud.ErrorForbidden)
		}
		parts = append(parts, quoteRaw(readModelAlias)+"."+quote(column.column)+" = ?")
		args = append(args, value)
	}
	return strings.Join(parts, " AND "), args, nil
}

func (source *ReadModelSource) order(terms []crud.Sort) (string, error) {
	if len(terms) == 0 {
		terms = []crud.Sort{{Field: "id", Direction: crud.SortAscending}}
	}
	parts := make([]string, 0, len(terms))
	for _, term := range terms {
		alias := source.idColumn
		if term.Field != "id" {
			var ok bool
			alias, ok = source.fieldByKey[term.Field]
			if !ok {
				return "", public(crud.ErrorInvalidRequest)
			}
		}
		direction := "ASC"
		if term.Direction == crud.SortDescending {
			direction = "DESC"
		}
		parts = append(parts, quoteRaw(readModelAlias)+"."+quote(alias)+" "+direction)
	}
	return " ORDER BY " + strings.Join(parts, ", "), nil
}

func (source *ReadModelSource) records(rows *sql.Rows) ([]crud.Record, error) {
	var records []crud.Record
	for rows.Next() {
		record, err := source.scanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

type readModelScanner interface{ Scan(...any) error }

func (source *ReadModelSource) scanRecord(row readModelScanner) (crud.Record, error) {
	values := make([]any, len(source.fields))
	destinations := make([]any, 0, len(values)+1)
	var id any
	destinations = append(destinations, &id)
	for index := range values {
		destinations = append(destinations, &values[index])
	}
	if err := row.Scan(destinations...); err != nil {
		return crud.Record{}, err
	}
	fields := make(crud.Fields, len(source.fields))
	for index, field := range source.fields {
		fields[field.key] = sqlValue(values[index])
	}
	return crud.Record{ID: crud.RecordID(stringValue(id)), Version: 1, Fields: fields}, nil
}
