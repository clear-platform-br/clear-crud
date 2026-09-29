package sqladapter

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	crud "github.com/clear-platform-br/clear-crud"
)

var sqliteIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var sqliteInteger = regexp.MustCompile(`^-?[0-9]+$`)
var sqliteDecimal = regexp.MustCompile(`^-?[0-9]+\.[0-9]+$`)

// --- Adapter metadata bridge ------------------------------------------------

// Metadata exposes enum-like values declared by the SQLite schema. It runs
// during registry registration, never during a CRUD request.
func (source *SimpleTable) Metadata(ctx context.Context) (crud.SourceMetadata, error) {
	if source == nil || source.db == nil {
		return crud.SourceMetadata{}, fmt.Errorf("sqladapter: metadata source is not initialized")
	}
	var statement sql.NullString
	err := source.db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, string(source.table.Table)).Scan(&statement)
	if err == sql.ErrNoRows || !statement.Valid {
		return crud.SourceMetadata{}, nil
	}
	if err != nil {
		return crud.SourceMetadata{}, err
	}
	constraints, err := sqliteEnumConstraints(statement.String)
	if err != nil {
		return crud.SourceMetadata{}, err
	}
	fields := make(map[crud.FieldKey]crud.FieldMetadata)
	for field, column := range source.table.Fields {
		values, ok := constraints[column]
		if !ok || len(values) == 0 {
			continue
		}
		fields[field] = crud.FieldMetadata{Enum: append([]crud.Value(nil), values...)}
	}
	return crud.SourceMetadata{Fields: fields}, nil
}

// --- Conservative enum-expression parser -----------------------------------

func sqliteEnumConstraints(statement string) (map[Identifier][]crud.Value, error) {
	constraints := make(map[Identifier][]crud.Value)
	for _, expression := range sqliteCheckExpressions(statement) {
		column, values, ok := sqliteEnumExpression(expression)
		if !ok {
			continue
		}
		if previous, exists := constraints[column]; exists {
			values = intersectEnumValues(previous, values)
			if len(values) == 0 {
				return nil, fmt.Errorf("sqladapter: incompatible enum checks for column %q", column)
			}
		}
		constraints[column] = values
	}
	return constraints, nil
}

// --- SQL text scanner --------------------------------------------------------

func sqliteCheckExpressions(statement string) []string {
	upper := strings.ToUpper(statement)
	var expressions []string
	var quote byte
	for index := 0; index < len(statement); index++ {
		character := statement[index]
		if quote != 0 {
			if character == quote {
				if index+1 < len(statement) && statement[index+1] == quote {
					index++
				} else {
					quote = 0
				}
			}
			continue
		}
		if character == '-' && index+1 < len(statement) && statement[index+1] == '-' {
			index += 2
			for index < len(statement) && statement[index] != '\n' {
				index++
			}
			continue
		}
		if character == '/' && index+1 < len(statement) && statement[index+1] == '*' {
			index += 2
			for index+1 < len(statement) && !(statement[index] == '*' && statement[index+1] == '/') {
				index++
			}
			if index+1 < len(statement) {
				index++
			}
			continue
		}
		if character == '\'' || character == '"' || character == '`' {
			quote = character
			continue
		}
		if index+len("CHECK") > len(statement) || upper[index:index+len("CHECK")] != "CHECK" || !sqlKeywordBoundary(upper, index, index+len("CHECK")) {
			continue
		}
		open := skipSQLSpace(statement, index+len("CHECK"))
		if open >= len(statement) || statement[open] != '(' {
			index += len("CHECK") - 1
			continue
		}
		close := matchingSQLParen(statement, open)
		if close < 0 {
			break
		}
		expressions = append(expressions, statement[open+1:close])
		index = close
	}
	return expressions
}

func sqliteEnumExpression(expression string) (Identifier, []crud.Value, bool) {
	trimmed := strings.TrimSpace(expression)
	if column, values, ok := sqliteInExpression(trimmed); ok {
		return column, values, true
	}
	parts := splitSQLKeyword(trimmed, "OR")
	if len(parts) < 2 {
		return "", nil, false
	}
	var column Identifier
	values := make([]crud.Value, 0, len(parts))
	for _, part := range parts {
		candidate, value, ok := sqliteEqualityExpression(part)
		if !ok || column != "" && candidate != column {
			return "", nil, false
		}
		column = candidate
		values = append(values, value)
	}
	return column, uniqueEnumValues(values), len(values) > 0
}

func sqliteInExpression(expression string) (Identifier, []crud.Value, bool) {
	keyword := findSQLKeyword(expression, "IN")
	if keyword < 0 {
		return "", nil, false
	}
	column, ok := sqliteColumnIdentifier(strings.TrimSpace(expression[:keyword]))
	if !ok {
		return "", nil, false
	}
	open := skipSQLSpace(expression, keyword+len("IN"))
	if open >= len(expression) || expression[open] != '(' {
		return "", nil, false
	}
	close := matchingSQLParen(expression, open)
	if close < 0 || strings.TrimSpace(expression[close+1:]) != "" {
		return "", nil, false
	}
	parts := splitSQLComma(expression[open+1 : close])
	values := make([]crud.Value, 0, len(parts))
	for _, part := range parts {
		value, ok := sqliteLiteral(part)
		if !ok {
			return "", nil, false
		}
		values = append(values, value)
	}
	return column, uniqueEnumValues(values), len(values) > 0
}

func sqliteEqualityExpression(expression string) (Identifier, crud.Value, bool) {
	operator := findSQLByte(expression, '=')
	if operator < 0 {
		return "", nil, false
	}
	column, ok := sqliteColumnIdentifier(strings.TrimSpace(expression[:operator]))
	if !ok {
		return "", nil, false
	}
	value, ok := sqliteLiteral(strings.TrimSpace(expression[operator+1:]))
	return column, value, ok
}

func sqliteColumnIdentifier(value string) (Identifier, bool) {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '`' && value[len(value)-1] == '`') {
		value = value[1 : len(value)-1]
	}
	if !sqliteIdentifier.MatchString(value) {
		return "", false
	}
	return Identifier(value), true
}

func sqliteLiteral(value string) (crud.Value, bool) {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'"), true
	}
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return strings.ReplaceAll(value[1:len(value)-1], `""`, `"`), true
	}
	if sqliteInteger.MatchString(value) {
		parsed, err := strconv.ParseInt(value, 10, 64)
		return parsed, err == nil
	}
	if sqliteDecimal.MatchString(value) {
		return value, true
	}
	switch strings.ToUpper(value) {
	case "TRUE":
		return true, true
	case "FALSE":
		return false, true
	default:
		return nil, false
	}
}

func uniqueEnumValues(values []crud.Value) []crud.Value {
	unique := make([]crud.Value, 0, len(values))
	for _, value := range values {
		found := false
		for _, existing := range unique {
			if enumValueIdentity(existing) == enumValueIdentity(value) {
				found = true
				break
			}
		}
		if !found {
			unique = append(unique, value)
		}
	}
	return unique
}

func intersectEnumValues(left, right []crud.Value) []crud.Value {
	intersection := make([]crud.Value, 0, len(left))
	for _, candidate := range left {
		for _, allowed := range right {
			if enumValueIdentity(candidate) == enumValueIdentity(allowed) {
				intersection = append(intersection, candidate)
				break
			}
		}
	}
	return uniqueEnumValues(intersection)
}

func enumValueIdentity(value crud.Value) string { return fmt.Sprintf("%T:%v", value, value) }

func splitSQLComma(value string) []string { return splitSQL(value, ',') }

func splitSQLKeyword(value, keyword string) []string {
	parts := make([]string, 0, 2)
	start := 0
	for offset := 0; offset < len(value); {
		index := findSQLKeyword(value[offset:], keyword)
		if index < 0 {
			break
		}
		index += offset
		parts = append(parts, strings.TrimSpace(value[start:index]))
		start = index + len(keyword)
		offset = start
	}
	if len(parts) == 0 {
		return []string{value}
	}
	return append(parts, strings.TrimSpace(value[start:]))
}

func splitSQL(value string, separator byte) []string {
	parts := make([]string, 0, 2)
	start, depth := 0, 0
	var quote byte
	for index := 0; index < len(value); index++ {
		character := value[index]
		if quote != 0 {
			if character == quote {
				if index+1 < len(value) && value[index+1] == quote {
					index++
				} else {
					quote = 0
				}
			}
			continue
		}
		if character == '\'' || character == '"' || character == '`' {
			quote = character
			continue
		}
		switch character {
		case '(':
			depth++
		case ')':
			depth--
		case separator:
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(value[start:index]))
				start = index + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(value[start:]))
	return parts
}

func findSQLKeyword(value, keyword string) int {
	upper := strings.ToUpper(value)
	var quote byte
	for offset := 0; offset+len(keyword) <= len(value); offset++ {
		character := value[offset]
		if quote != 0 {
			if character == quote {
				if offset+1 < len(value) && value[offset+1] == quote {
					offset++
				} else {
					quote = 0
				}
			}
			continue
		}
		if character == '\'' || character == '"' || character == '`' {
			quote = character
			continue
		}
		if upper[offset:offset+len(keyword)] == keyword && sqlKeywordBoundary(upper, offset, offset+len(keyword)) {
			return offset
		}
	}
	return -1
}

func findSQLByte(value string, wanted byte) int {
	var quote byte
	for index := 0; index < len(value); index++ {
		character := value[index]
		if quote != 0 {
			if character == quote {
				if index+1 < len(value) && value[index+1] == quote {
					index++
				} else {
					quote = 0
				}
			}
			continue
		}
		if character == '\'' || character == '"' || character == '`' {
			quote = character
			continue
		}
		if character == wanted {
			return index
		}
	}
	return -1
}

func matchingSQLParen(value string, open int) int {
	depth := 0
	var quote byte
	for index := open; index < len(value); index++ {
		character := value[index]
		if quote != 0 {
			if character == quote {
				if index+1 < len(value) && value[index+1] == quote {
					index++
				} else {
					quote = 0
				}
			}
			continue
		}
		if character == '\'' || character == '"' || character == '`' {
			quote = character
			continue
		}
		switch character {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func skipSQLSpace(value string, offset int) int {
	for offset < len(value) && (value[offset] == ' ' || value[offset] == '\n' || value[offset] == '\r' || value[offset] == '\t') {
		offset++
	}
	return offset
}

func sqlKeywordBoundary(value string, start, end int) bool {
	return (start == 0 || !isSQLIdentifierByte(value[start-1])) && (end >= len(value) || !isSQLIdentifierByte(value[end]))
}

func isSQLIdentifierByte(value byte) bool {
	return value == '_' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' || value >= '0' && value <= '9'
}
