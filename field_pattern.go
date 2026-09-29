package crud

import (
	"regexp"
	"strconv"
)

const maxFieldPatternLength = 512

func validatePatternDefinition(field Field, path string) error {
	if field.PatternMessage != "" && field.Pattern == "" {
		return invalidDefinition(path+".patternMessage", "requires a pattern")
	}
	if field.Pattern == "" {
		return nil
	}
	if len(field.Pattern) > maxFieldPatternLength {
		return invalidDefinition(path+".pattern", "must not exceed 512 characters")
	}
	if !patternFieldType(field.Type) {
		return invalidDefinition(path+".pattern", "requires a string-backed field")
	}
	if _, err := regexp.Compile(field.Pattern); err != nil {
		return invalidDefinition(path+".pattern", "must be a valid RE2 expression")
	}
	return nil
}

func patternFieldType(fieldType FieldType) bool {
	switch fieldType {
	case FieldString, FieldText, FieldDecimal, FieldDate, FieldDateTime, FieldEmail, FieldPhone:
		return true
	default:
		return false
	}
}

func prepareDefinitionPatterns(definition Definition) (Definition, error) {
	patterns := make(map[FieldKey]*regexp.Regexp)
	for index, field := range definition.Fields {
		if field.Pattern == "" {
			continue
		}
		compiled, err := regexp.Compile(field.Pattern)
		if err != nil {
			return Definition{}, invalidDefinition("fields["+strconv.Itoa(index)+"].pattern", "must be a valid RE2 expression")
		}
		patterns[field.Key] = compiled
	}
	if len(patterns) == 0 {
		definition.fieldPatterns = nil
	} else {
		definition.fieldPatterns = patterns
	}
	return definition, nil
}

func validateFieldPattern(field Field, value string, compiled *regexp.Regexp) error {
	if field.Pattern == "" {
		return nil
	}
	if compiled == nil {
		var err error
		compiled, err = regexp.Compile(field.Pattern)
		if err != nil {
			return invalidMutation(FieldErrors{field.Key: patternMessage(field)})
		}
	}
	if !compiled.MatchString(value) {
		return invalidMutation(FieldErrors{field.Key: patternMessage(field)})
	}
	return nil
}

func patternMessage(field Field) MessageCode {
	if field.PatternMessage != "" {
		return field.PatternMessage
	}
	return "crud.field.pattern"
}
