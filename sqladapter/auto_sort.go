package sqladapter

import (
	"context"
	"database/sql"
	"fmt"

	crud "github.com/clear-platform-br/clear-crud"
)

// sqliteIndex is the small, structural part of an index needed to validate
// server-owned collection ordering. Partial and expression indexes are kept
// out because they cannot guarantee the generic CRUD predicate and ordering.
type sqliteIndex struct {
	columns []Identifier
}

func inspectIndexes(ctx context.Context, db *sql.DB, table Identifier) ([]sqliteIndex, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA index_list("+quote(table)+")")
	if err != nil {
		return nil, err
	}

	indexNames := make([]string, 0)
	for rows.Next() {
		var sequence, unique, partial int
		var name, origin string
		if err := rows.Scan(&sequence, &name, &unique, &origin, &partial); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if partial == 0 {
			indexNames = append(indexNames, name)
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	indexes := make([]sqliteIndex, 0, len(indexNames))
	for _, name := range indexNames {
		columns, supported, err := inspectIndexColumns(ctx, db, name)
		if err != nil {
			return nil, err
		}
		if supported && len(columns) > 0 {
			indexes = append(indexes, sqliteIndex{columns: columns})
		}
	}
	return indexes, nil
}

func inspectIndexColumns(ctx context.Context, db *sql.DB, name string) ([]Identifier, bool, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA index_info("+quoteRaw(name)+")")
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	columns := make([]Identifier, 0)
	for rows.Next() {
		var sequence, columnID int
		var name sql.NullString
		if err := rows.Scan(&sequence, &columnID, &name); err != nil {
			return nil, false, err
		}
		if !name.Valid {
			return nil, false, nil
		}
		identifier, err := NewIdentifier(name.String)
		if err != nil {
			return nil, false, nil
		}
		columns = append(columns, identifier)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return columns, true, nil
}

func autoDefaultSort(config AutoTableConfig, table TableDefinition, fields []crud.Field, ordered, listColumns []crud.FieldKey, indexes []sqliteIndex) ([]crud.Sort, error) {
	if len(config.DefaultSort) != 0 {
		if err := validateConfiguredSort(config, table, config.DefaultSort, indexes); err != nil {
			return nil, err
		}
		return append([]crud.Sort(nil), config.DefaultSort...), nil
	}

	candidate := indexedFieldOrder(indexes, table.Fields)
	fromIndex := len(candidate) != 0
	if len(candidate) == 0 {
		candidate = append([]crud.FieldKey(nil), listColumns...)
	}
	if len(candidate) > 0 && fieldType(fields, candidate[0]) == crud.FieldBoolean {
		// Record ordering is independent from the grid projection. In
		// particular, an explicit grid order must not make an indexed
		// boolean sort select a different, unsupported successor column.
		if next := nextSortField(candidate[0], ordered, listColumns); next != "" {
			candidate = []crud.FieldKey{candidate[0], next}
		}
	}
	terms := make([]crud.Sort, 0, len(candidate))
	for _, field := range candidate {
		terms = append(terms, crud.Sort{Field: field, Direction: crud.SortAscending})
	}
	if fromIndex && !supportsIndexOrder(table.Fields, fixedIndexColumns(config), terms, indexes) {
		return nil, fmt.Errorf("sqladapter: automatic default sort is not supported by an index on table %q", config.Table)
	}
	return terms, nil
}

func indexedFieldOrder(indexes []sqliteIndex, fields map[crud.FieldKey]Identifier) []crud.FieldKey {
	byColumn := make(map[Identifier]crud.FieldKey, len(fields))
	for key, column := range fields {
		byColumn[column] = key
	}
	for _, index := range indexes {
		candidate := make([]crud.FieldKey, 0, len(index.columns))
		for _, column := range index.columns {
			if field, ok := byColumn[column]; ok {
				candidate = append(candidate, field)
			}
		}
		if len(candidate) > 0 {
			return candidate
		}
	}
	return nil
}

func nextSortField(first crud.FieldKey, listColumns, ordered []crud.FieldKey) crud.FieldKey {
	for _, field := range listColumns {
		if field != first {
			return field
		}
	}
	for _, field := range ordered {
		if field != first {
			return field
		}
	}
	return ""
}

func fieldType(fields []crud.Field, key crud.FieldKey) crud.FieldType {
	for _, field := range fields {
		if field.Key == key {
			return field.Type
		}
	}
	return ""
}

func validateConfiguredSort(config AutoTableConfig, table TableDefinition, terms []crud.Sort, indexes []sqliteIndex) error {
	if len(terms) == 0 {
		return fmt.Errorf("sqladapter: default sort requires at least one field")
	}
	seen := make(map[crud.FieldKey]struct{}, len(terms))
	for _, term := range terms {
		if !autoKey(string(term.Field)) {
			return fmt.Errorf("sqladapter: invalid default sort field %q", term.Field)
		}
		if _, exists := table.Fields[term.Field]; !exists {
			return fmt.Errorf("sqladapter: default sort field %q does not match a conventional field", term.Field)
		}
		if term.Direction != crud.SortAscending && term.Direction != crud.SortDescending {
			return fmt.Errorf("sqladapter: invalid default sort direction for %q", term.Field)
		}
		if _, exists := seen[term.Field]; exists {
			return fmt.Errorf("sqladapter: duplicate default sort field %q", term.Field)
		}
		seen[term.Field] = struct{}{}
	}
	if !supportsIndexOrder(table.Fields, fixedIndexColumns(config), terms, indexes) {
		return fmt.Errorf("sqladapter: default sort is not supported by an index on table %q", config.Table)
	}
	return nil
}

func fixedIndexColumns(config AutoTableConfig) map[Identifier]struct{} {
	// Scope predicates are strict equalities and can be skipped as an index
	// prefix. The conventional soft-delete predicate is `(archived IS NULL OR
	// archived = 0)`, so an archive column cannot be treated as a fixed prefix
	// without falsely claiming that SQLite can stream the requested order.
	fixed := make(map[Identifier]struct{}, len(config.ScopeColumns))
	for _, column := range config.ScopeColumns {
		fixed[column] = struct{}{}
	}
	return fixed
}

func supportsIndexOrder(fields map[crud.FieldKey]Identifier, fixed map[Identifier]struct{}, terms []crud.Sort, indexes []sqliteIndex) bool {
	if len(terms) == 0 || !sameSortDirection(terms) {
		return false
	}
	columns := make([]Identifier, 0, len(terms))
	for _, term := range terms {
		column, ok := fields[term.Field]
		if !ok {
			return false
		}
		columns = append(columns, column)
	}
	for _, index := range indexes {
		position := 0
		for _, column := range index.columns {
			if position == 0 {
				if column == columns[0] {
					position++
				} else if _, ok := fixed[column]; !ok {
					break
				}
			} else if column == columns[position] {
				position++
			} else {
				break
			}
			if position == len(columns) {
				return true
			}
		}
	}
	return false
}

func sameSortDirection(terms []crud.Sort) bool {
	if len(terms) < 2 {
		return true
	}
	for _, term := range terms[1:] {
		if term.Direction != terms[0].Direction {
			return false
		}
	}
	return true
}
