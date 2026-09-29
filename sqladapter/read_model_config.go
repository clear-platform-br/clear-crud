package sqladapter

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	crud "github.com/clear-platform-br/clear-crud"
)

type preparedReadModel struct {
	query      string
	scopes     []readModelScope
	fields     []readModelField
	fieldByKey map[crud.FieldKey]Identifier
	selectList string
}

func completeReadModelDefinition(definition crud.Definition) crud.Definition {
	if len(definition.Grid.Searchable) == 0 {
		definition.Grid.Searchable = append([]crud.FieldKey(nil), definition.Grid.Columns...)
	}
	if len(definition.Grid.Sortable) == 0 {
		definition.Grid.Sortable = append([]crud.FieldKey(nil), definition.Grid.Columns...)
	}
	return definition
}

func completeReadModelConfig(definition crud.Definition, config ReadModelConfig) (ReadModelConfig, error) {
	scopes := make(map[string]Identifier, len(definition.Scope.Keys))
	for _, key := range definition.Scope.Keys {
		scopes[key] = Identifier(key)
	}
	for key, alias := range config.ScopeColumns {
		if _, ok := scopes[key]; !ok {
			return ReadModelConfig{}, fmt.Errorf("sqladapter: read model scope %q is not declared", key)
		}
		scopes[key] = alias
	}
	config.ScopeColumns = scopes
	fields := make(map[crud.FieldKey]Identifier, len(definition.Fields))
	for _, field := range definition.Fields {
		fields[field.Key] = Identifier(field.Key)
	}
	for key, alias := range config.Fields {
		if _, ok := fields[key]; !ok {
			return ReadModelConfig{}, fmt.Errorf("sqladapter: read model field %q is not declared", key)
		}
		fields[key] = alias
	}
	config.Fields = fields
	if len(config.Searchable) == 0 {
		config.Searchable = append([]crud.FieldKey(nil), definition.Grid.Searchable...)
	}
	return config, nil
}

func prepareReadModel(ctx context.Context, db *sql.DB, config ReadModelConfig) (preparedReadModel, error) {
	query := strings.TrimSpace(config.Query)
	upper := strings.ToUpper(query)
	if query == "" || strings.Contains(query, ";") || !(strings.HasPrefix(upper, "SELECT ") || strings.HasPrefix(upper, "WITH ")) {
		return preparedReadModel{}, fmt.Errorf("sqladapter: read model query must be one server-owned SELECT statement")
	}
	if !validIdentifier(config.IDColumn) || len(config.Fields) == 0 {
		return preparedReadModel{}, fmt.Errorf("sqladapter: read model id and fields are required")
	}
	for key, alias := range config.ScopeColumns {
		if !autoKey(key) || !validIdentifier(alias) {
			return preparedReadModel{}, fmt.Errorf("sqladapter: invalid read model scope column")
		}
	}
	seenAliases := map[Identifier]struct{}{config.IDColumn: {}}
	for key, alias := range config.Fields {
		if !autoKey(string(key)) || !validIdentifier(alias) {
			return preparedReadModel{}, fmt.Errorf("sqladapter: invalid read model field")
		}
		if _, exists := seenAliases[alias]; exists {
			return preparedReadModel{}, fmt.Errorf("sqladapter: duplicate read model result alias %q", alias)
		}
		seenAliases[alias] = struct{}{}
	}
	for _, key := range config.Searchable {
		if _, ok := config.Fields[key]; !ok {
			return preparedReadModel{}, fmt.Errorf("sqladapter: read model searchable field %q is not declared", key)
		}
	}
	if err := verifyReadModelColumns(ctx, db, query, config); err != nil {
		return preparedReadModel{}, err
	}

	prepared := preparedReadModel{
		query:      query,
		fieldByKey: make(map[crud.FieldKey]Identifier, len(config.Fields)),
	}
	for _, key := range sortedScopeKeys(config.ScopeColumns) {
		prepared.scopes = append(prepared.scopes, readModelScope{key: key, column: config.ScopeColumns[key]})
	}
	for _, key := range readModelFieldKeys(config.Fields) {
		field := readModelField{key: key, column: config.Fields[key]}
		prepared.fields = append(prepared.fields, field)
		prepared.fieldByKey[key] = field.column
	}
	columns := []string{quoteRaw(readModelAlias) + "." + quote(config.IDColumn)}
	for _, field := range prepared.fields {
		columns = append(columns, quoteRaw(readModelAlias)+"."+quote(field.column))
	}
	prepared.selectList = strings.Join(columns, ", ")
	return prepared, nil
}

func verifyReadModelColumns(ctx context.Context, db *sql.DB, query string, config ReadModelConfig) error {
	rows, err := db.QueryContext(ctx, "SELECT * FROM ("+query+") AS "+quoteRaw(readModelAlias)+" LIMIT 0")
	if err != nil {
		return fmt.Errorf("sqladapter: inspect read model: %w", err)
	}
	columns, columnsErr := rows.Columns()
	closeErr := rows.Close()
	if columnsErr != nil {
		return columnsErr
	}
	if closeErr != nil {
		return closeErr
	}
	available := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		available[strings.ToLower(column)] = struct{}{}
	}
	required := append([]Identifier{config.IDColumn}, readModelScopeAliases(config.ScopeColumns)...)
	required = append(required, readModelFieldAliases(config.Fields)...)
	for _, alias := range required {
		if _, ok := available[strings.ToLower(string(alias))]; !ok {
			return fmt.Errorf("sqladapter: read model query does not expose %q", alias)
		}
	}
	return nil
}

func readModelScopeAliases(columns map[string]Identifier) []Identifier {
	keys := sortedScopeKeys(columns)
	aliases := make([]Identifier, 0, len(keys))
	for _, key := range keys {
		aliases = append(aliases, columns[key])
	}
	return aliases
}

func readModelFieldKeys(fields map[crud.FieldKey]Identifier) []crud.FieldKey {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	result := make([]crud.FieldKey, 0, len(keys))
	for _, key := range keys {
		result = append(result, crud.FieldKey(key))
	}
	return result
}

func readModelFieldAliases(fields map[crud.FieldKey]Identifier) []Identifier {
	keys := readModelFieldKeys(fields)
	aliases := make([]Identifier, 0, len(keys))
	for _, key := range keys {
		aliases = append(aliases, fields[key])
	}
	return aliases
}

func sortedScopeKeys(columns map[string]Identifier) []string {
	keys := make([]string, 0, len(columns))
	for key := range columns {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
