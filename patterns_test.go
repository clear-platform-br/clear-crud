package crud

import (
	"regexp"
	"testing"
)

func TestPatternCatalogCompilesAsRE2(t *testing.T) {
	for name, pattern := range patternCatalog() {
		name, pattern := name, pattern
		t.Run(name, func(t *testing.T) {
			if len(pattern) > maxFieldPatternLength {
				t.Fatalf("pattern length = %d, want at most %d", len(pattern), maxFieldPatternLength)
			}
			if _, err := regexp.Compile(pattern); err != nil {
				t.Fatalf("regexp.Compile(%q) = %v", pattern, err)
			}
		})
	}
}

func TestPatternCatalogSemantics(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		valid   []string
		invalid []string
	}{
		{name: "non blank", pattern: PatternNonBlank, valid: []string{"A", "  A  "}, invalid: []string{"", " \t"}},
		{name: "no whitespace", pattern: PatternNoWhitespace, valid: []string{"abc", "Á1"}, invalid: []string{"a b", ""}},
		{name: "edge whitespace", pattern: PatternNoLeadingOrTrailingWhitespace, valid: []string{"a b", "a"}, invalid: []string{" a", "a "}},
		{name: "letters", pattern: PatternLettersOnly, valid: []string{"Abc", "Ação"}, invalid: []string{"A1", ""}},
		{name: "uppercase letters", pattern: PatternUppercaseLetters, valid: []string{"ABC", "ÁÇ"}, invalid: []string{"Ab", "123"}},
		{name: "lowercase letters", pattern: PatternLowercaseLetters, valid: []string{"abc", "ação"}, invalid: []string{"aB", "123"}},
		{name: "first uppercase", pattern: PatternFirstLetterUpper, valid: []string{"Ação", "A"}, invalid: []string{"ação", ""}},
		{name: "first lowercase", pattern: PatternFirstLetterLower, valid: []string{"ação", "a"}, invalid: []string{"Ação", ""}},
		{name: "words", pattern: PatternWords, valid: []string{"São Paulo", "Ação"}, invalid: []string{"São-Paulo", "São  Paulo"}},
		{name: "letters and numbers", pattern: PatternLettersAndNumbers, valid: []string{"Ação2"}, invalid: []string{"Ação 2", ""}},
		{name: "letters numbers and spaces", pattern: PatternLettersNumbersAndSpaces, valid: []string{"Sala 2", "Ação 2"}, invalid: []string{"Sala-2"}},
		{name: "digits", pattern: PatternDigitsOnly, valid: []string{"0", "123"}, invalid: []string{"-1", "1.2"}},
		{name: "signed integer", pattern: PatternSignedInteger, valid: []string{"0", "-12"}, invalid: []string{"1.2", "+1"}},
		{name: "decimal", pattern: PatternDecimal, valid: []string{"1", "-1,25", "2.5"}, invalid: []string{"1,2.5", "1."}},
		{name: "uppercase code", pattern: PatternUppercaseCode, valid: []string{"BR-01", "A_2"}, invalid: []string{"br-01", "_A"}},
		{name: "lowercase code", pattern: PatternLowercaseCode, valid: []string{"br-01", "a_2"}, invalid: []string{"BR-01", "-a"}},
		{name: "upper camel case", pattern: PatternUpperCamelCase, valid: []string{"ClearCrud2"}, invalid: []string{"clearCrud", "Clear CRUD"}},
		{name: "lower camel case", pattern: PatternLowerCamelCase, valid: []string{"clearCrud2"}, invalid: []string{"ClearCrud", "clear CRUD"}},
		{name: "snake case", pattern: PatternSnakeCase, valid: []string{"clear_crud2"}, invalid: []string{"Clear_crud", "clear__crud"}},
		{name: "kebab case", pattern: PatternKebabCase, valid: []string{"clear-crud2"}, invalid: []string{"Clear-crud", "clear--crud"}},
		{name: "dot separated key", pattern: PatternDotSeparatedKey, valid: []string{"crud.field.name"}, invalid: []string{"Crud.field", "crud..name"}},
		{name: "two letter code", pattern: PatternTwoLetterCode, valid: []string{"BR"}, invalid: []string{"B", "Br"}},
		{name: "three letter code", pattern: PatternThreeLetterCode, valid: []string{"ISO"}, invalid: []string{"BR", "Iso"}},
		{name: "hexadecimal", pattern: PatternHexadecimal, valid: []string{"0xFF", "abc123"}, invalid: []string{"0x", "xyz"}},
		{name: "hex color", pattern: PatternHexColor, valid: []string{"#fff", "#12ABef"}, invalid: []string{"fff", "#12"}},
		{name: "uuid", pattern: PatternUUID, valid: []string{"550e8400-e29b-41d4-a716-446655440000"}, invalid: []string{"550e8400-e29b-41d4-a716", "not-a-uuid"}},
		{name: "version", pattern: PatternVersion, valid: []string{"1", "1.2", "1.2.3"}, invalid: []string{"1.2.3.4", "v1.2"}},
		{name: "time", pattern: PatternTime24Hour, valid: []string{"00:00", "23:59"}, invalid: []string{"24:00", "9:00"}},
		{name: "E164 phone", pattern: PatternE164Phone, valid: []string{"+5511999999999"}, invalid: []string{"5511999999999", "+0123"}},
		{name: "HTTP URL", pattern: PatternHTTPURL, valid: []string{"https://clear.example/path"}, invalid: []string{"ftp://clear.example", "https://clear.example/a b"}},
		{name: "MAC address", pattern: PatternMACAddress, valid: []string{"AA:bb:01:02:03:FF"}, invalid: []string{"AA-bb-01-02-03-FF", "AA:bb:01:02:03"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compiled := regexp.MustCompile(test.pattern)
			for _, value := range test.valid {
				if !compiled.MatchString(value) {
					t.Errorf("pattern %q rejected valid value %q", test.pattern, value)
				}
			}
			for _, value := range test.invalid {
				if compiled.MatchString(value) {
					t.Errorf("pattern %q accepted invalid value %q", test.pattern, value)
				}
			}
		})
	}
}

func patternCatalog() map[string]string {
	return map[string]string{
		"non_blank":                         PatternNonBlank,
		"no_whitespace":                     PatternNoWhitespace,
		"no_leading_or_trailing_whitespace": PatternNoLeadingOrTrailingWhitespace,
		"letters_only":                      PatternLettersOnly,
		"uppercase_letters":                 PatternUppercaseLetters,
		"all_upper":                         PatternAllUpper,
		"lowercase_letters":                 PatternLowercaseLetters,
		"all_lower":                         PatternAllLower,
		"first_letter_upper":                PatternFirstLetterUpper,
		"first_letter_lower":                PatternFirstLetterLower,
		"words":                             PatternWords,
		"letters_and_numbers":               PatternLettersAndNumbers,
		"alphanumeric":                      PatternAlphaNumeric,
		"letters_numbers_and_spaces":        PatternLettersNumbersAndSpaces,
		"alphanumeric_with_spaces":          PatternAlphaNumericWithSpaces,
		"digits_only":                       PatternDigitsOnly,
		"unsigned_integer":                  PatternUnsignedInteger,
		"signed_integer":                    PatternSignedInteger,
		"decimal":                           PatternDecimal,
		"uppercase_code":                    PatternUppercaseCode,
		"lowercase_code":                    PatternLowercaseCode,
		"upper_camel_case":                  PatternUpperCamelCase,
		"lower_camel_case":                  PatternLowerCamelCase,
		"snake_case":                        PatternSnakeCase,
		"kebab_case":                        PatternKebabCase,
		"slug":                              PatternSlug,
		"dot_separated_key":                 PatternDotSeparatedKey,
		"two_letter_code":                   PatternTwoLetterCode,
		"three_letter_code":                 PatternThreeLetterCode,
		"hexadecimal":                       PatternHexadecimal,
		"hex_color":                         PatternHexColor,
		"uuid":                              PatternUUID,
		"version":                           PatternVersion,
		"time_24_hour":                      PatternTime24Hour,
		"e164_phone":                        PatternE164Phone,
		"http_url":                          PatternHTTPURL,
		"mac_address":                       PatternMACAddress,
	}
}
