package sqladapter

import (
	"strconv"
	"strings"

	crud "github.com/clear-platform-br/clear-crud"
)

// sqliteStaticDefault converts only literal SQLite defaults into public scalar
// values. Expressions such as CURRENT_TIMESTAMP stay with the database and are
// deliberately not exposed as renderer defaults.
func sqliteStaticDefault(expression string, fieldType crud.FieldType) (crud.Value, bool) {
	raw := unwrapSQLiteDefault(expression)
	if raw == "" || strings.EqualFold(raw, "NULL") {
		return nil, false
	}
	if quoted, ok := sqliteDefaultQuotedString(raw); ok {
		switch fieldType {
		case crud.FieldInteger:
			value, err := strconv.ParseInt(quoted, 10, 64)
			return value, err == nil
		case crud.FieldBoolean:
			return sqliteDefaultBoolean(quoted)
		default:
			return quoted, true
		}
	}
	switch fieldType {
	case crud.FieldInteger:
		value, err := strconv.ParseInt(raw, 10, 64)
		return value, err == nil
	case crud.FieldBoolean:
		return sqliteDefaultBoolean(raw)
	case crud.FieldDecimal:
		if sqliteInteger.MatchString(raw) || sqliteDecimal.MatchString(raw) {
			return raw, true
		}
	}
	return nil, false
}

func unwrapSQLiteDefault(expression string) string {
	raw := strings.TrimSpace(expression)
	for len(raw) >= 2 && raw[0] == '(' && raw[len(raw)-1] == ')' && sqliteDefaultParensEncloseAll(raw) {
		raw = strings.TrimSpace(raw[1 : len(raw)-1])
	}
	return raw
}

func sqliteDefaultParensEncloseAll(expression string) bool {
	depth := 0
	var quote byte
	for index := 0; index < len(expression); index++ {
		character := expression[index]
		if quote != 0 {
			if character == quote {
				if index+1 < len(expression) && expression[index+1] == quote {
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
			if depth == 0 && index != len(expression)-1 {
				return false
			}
		}
	}
	return depth == 0 && quote == 0
}

func sqliteDefaultQuotedString(expression string) (string, bool) {
	if len(expression) < 2 || expression[0] != '\'' || expression[len(expression)-1] != '\'' {
		return "", false
	}
	return strings.ReplaceAll(expression[1:len(expression)-1], "''", "'"), true
}

func sqliteDefaultBoolean(expression string) (crud.Value, bool) {
	switch strings.ToUpper(strings.TrimSpace(expression)) {
	case "1", "TRUE":
		return true, true
	case "0", "FALSE":
		return false, true
	default:
		return nil, false
	}
}
