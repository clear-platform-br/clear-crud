package sqladapter

import (
	"context"
	"database/sql"
	"encoding/json"
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

// NumericFilterRanges returns the scoped minimum and maximum from the complete
// registered read model, independent of the current page, search or filters.
func (source *ReadModelSource) NumericFilterRanges(ctx context.Context, scope crud.Scope, _ bool, fields []crud.FieldKey) (map[crud.FieldKey]crud.NumericFilterRange, error) {
	ranges := make(map[crud.FieldKey]crud.NumericFilterRange, len(fields))
	if source == nil || len(fields) == 0 {
		return ranges, nil
	}
	where, args, err := source.scopeWhere(scope)
	if err != nil {
		return nil, err
	}
	selectItems := make([]string, 0, len(fields)*2)
	for _, key := range fields {
		alias, ok := source.fieldByKey[key]
		if !ok {
			return nil, public(crud.ErrorInvalidRequest)
		}
		column := quoteRaw(readModelAlias) + "." + quote(alias)
		selectItems = append(selectItems, "MIN("+column+")", "MAX("+column+")")
	}
	statement := "SELECT " + strings.Join(selectItems, ", ") + " FROM (" + source.query + ") AS " + quoteRaw(readModelAlias) + " WHERE " + where
	values := make([]any, len(fields)*2)
	destinations := make([]any, len(values))
	for index := range values {
		destinations[index] = &values[index]
	}
	if err := source.db.QueryRowContext(ctx, statement, args...).Scan(destinations...); err != nil {
		return nil, err
	}
	for index, key := range fields {
		minimum, maximum := values[index*2], values[index*2+1]
		if minimum == nil || maximum == nil {
			continue
		}
		ranges[key] = crud.NumericFilterRange{Minimum: numericRangeString(minimum), Maximum: numericRangeString(maximum)}
	}
	return ranges, nil
}

// ListReport returns a normal page and subtotals for the groups represented on
// that page. Every subtotal query reuses the complete scoped and filtered set.
func (source *ReadModelSource) ListReport(ctx context.Context, scope crud.Scope, query crud.Query, report crud.ReportDefinition) (crud.Page, error) {
	if source == nil || query.Page.Mode != crud.PageModeOffset || len(report.Groups) == 0 || len(report.Groups) > 4 || len(report.Measures) == 0 || len(report.Measures) > 8 {
		return crud.Page{}, public(crud.ErrorInvalidRequest)
	}
	page, err := source.List(ctx, scope, query)
	if err != nil {
		return crud.Page{}, err
	}
	summary := &crud.ReportSummary{Subtotals: []crud.ReportSubtotal{}}
	if len(page.Records) == 0 {
		page.Report = summary
		return page, nil
	}
	baseWhere, baseArgs, err := source.where(scope, query)
	if err != nil {
		return crud.Page{}, err
	}
	for level := 1; level <= len(report.Groups); level++ {
		groupAliases := make([]Identifier, level)
		for index := range groupAliases {
			alias, ok := source.fieldByKey[report.Groups[index].Field]
			if !ok {
				return crud.Page{}, public(crud.ErrorInvalidRequest)
			}
			groupAliases[index] = alias
		}
		paths, err := visibleReportPaths(page.Records, report, level)
		if err != nil {
			return crud.Page{}, err
		}
		if len(paths) == 0 {
			continue
		}
		pathPredicate, pathArgs, err := source.reportPathPredicate(groupAliases, paths)
		if err != nil {
			return crud.Page{}, err
		}
		selectItems := make([]string, 0, len(groupAliases)+len(report.Measures))
		groupItems := make([]string, len(groupAliases))
		for index, alias := range groupAliases {
			groupItems[index] = quoteRaw(readModelAlias) + "." + quote(alias)
			selectItems = append(selectItems, groupItems[index])
		}
		measureItems := make([]crud.ReportMeasure, 0, len(report.Measures))
		for _, measure := range report.Measures {
			switch measure.Operation {
			case crud.ReportCount:
				selectItems = append(selectItems, "COUNT(*)")
				measureItems = append(measureItems, measure)
			case crud.ReportSum:
				alias, ok := source.fieldByKey[measure.Field]
				if !ok {
					return crud.Page{}, public(crud.ErrorInvalidRequest)
				}
				selectItems = append(selectItems, "SUM("+quoteRaw(readModelAlias)+"."+quote(alias)+")")
				measureItems = append(measureItems, measure)
			default:
				return crud.Page{}, public(crud.ErrorInvalidRequest)
			}
		}
		statement := "SELECT " + strings.Join(selectItems, ", ") + " FROM (" + source.query + ") AS " + quoteRaw(readModelAlias) + " WHERE " + baseWhere + " AND (" + pathPredicate + ") GROUP BY " + strings.Join(groupItems, ", ") + " ORDER BY " + strings.Join(groupItems, ", ")
		args := append(append([]any(nil), baseArgs...), pathArgs...)
		rows, err := source.db.QueryContext(ctx, statement, args...)
		if err != nil {
			return crud.Page{}, err
		}
		for rows.Next() {
			groupValues := make([]any, level)
			destinations := make([]any, 0, level+len(measureItems))
			for index := range groupValues {
				destinations = append(destinations, &groupValues[index])
			}
			measureValues := make([]any, len(measureItems))
			for index := range measureValues {
				destinations = append(destinations, &measureValues[index])
			}
			if err := rows.Scan(destinations...); err != nil {
				_ = rows.Close()
				return crud.Page{}, err
			}
			values := make([]crud.Value, len(groupValues))
			for index, value := range groupValues {
				values[index] = sqlValue(value)
			}
			measures := make([]crud.ReportMeasureValue, len(measureItems))
			for index, measure := range measureItems {
				measures[index] = crud.ReportMeasureValue{Key: measure.Key, Value: sqlValue(measureValues[index])}
			}
			summary.Subtotals = append(summary.Subtotals, crud.ReportSubtotal{Level: uint8(level), Values: values, Measures: measures})
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return crud.Page{}, err
		}
		if err := rows.Close(); err != nil {
			return crud.Page{}, err
		}
	}
	page.Report = summary
	return page, nil
}

func visibleReportPaths(records []crud.Record, report crud.ReportDefinition, level int) ([][]crud.Value, error) {
	paths := make([][]crud.Value, 0, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		path := make([]crud.Value, level)
		for index := 0; index < level; index++ {
			value, ok := record.Fields[report.Groups[index].Field]
			if !ok {
				return nil, public(crud.ErrorInvalidRequest)
			}
			path[index] = value
		}
		encoded, err := json.Marshal(path)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[string(encoded)]; exists {
			continue
		}
		seen[string(encoded)] = struct{}{}
		paths = append(paths, path)
	}
	return paths, nil
}

func (source *ReadModelSource) reportPathPredicate(columns []Identifier, paths [][]crud.Value) (string, []any, error) {
	if len(columns) == 0 || len(paths) == 0 {
		return "", nil, public(crud.ErrorInvalidRequest)
	}
	terms := make([]string, 0, len(paths))
	args := make([]any, 0, len(paths)*len(columns))
	for _, path := range paths {
		if len(path) != len(columns) {
			return "", nil, public(crud.ErrorInvalidRequest)
		}
		parts := make([]string, len(columns))
		for index, value := range path {
			column := quoteRaw(readModelAlias) + "." + quote(columns[index])
			if value == nil {
				parts[index] = column + " IS NULL"
				continue
			}
			if !reportScalar(value) {
				return "", nil, public(crud.ErrorInvalidRequest)
			}
			parts[index] = column + " = ?"
			args = append(args, value)
		}
		terms = append(terms, "("+strings.Join(parts, " AND ")+")")
	}
	return strings.Join(terms, " OR "), args, nil
}

func reportScalar(value any) bool {
	switch value.(type) {
	case string, bool, int64:
		return true
	default:
		return false
	}
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
