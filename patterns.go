package crud

// Reusable server-owned RE2 patterns for common string-backed field shapes.
//
// These constants are ergonomic aliases for declarative Field.Pattern values;
// they do not change validation semantics and custom expressions remain
// supported for reviewed cases that are not covered here. Prefer the
// normalized FieldType values for email, phone, date, datetime and decimal
// data when they express the actual meaning of a field.
const (
	// PatternNonBlank requires at least one non-whitespace character.
	PatternNonBlank = `(?s)^.*\S.*$`
	// PatternNoWhitespace accepts one or more non-whitespace characters.
	PatternNoWhitespace = `^\S+$`
	// PatternNoLeadingOrTrailingWhitespace rejects whitespace at either edge.
	PatternNoLeadingOrTrailingWhitespace = `^\S(?:.*\S)?$`

	// PatternLettersOnly accepts one or more Unicode letters.
	PatternLettersOnly = `^\p{L}+$`
	// PatternUppercaseLetters accepts only Unicode uppercase letters.
	PatternUppercaseLetters = `^\p{Lu}+$`
	// PatternAllUpper is the concise name for PatternUppercaseLetters.
	PatternAllUpper = PatternUppercaseLetters
	// PatternLowercaseLetters accepts only Unicode lowercase letters.
	PatternLowercaseLetters = `^\p{Ll}+$`
	// PatternAllLower is the concise name for PatternLowercaseLetters.
	PatternAllLower = PatternLowercaseLetters
	// PatternFirstLetterUpper requires the first character to be uppercase.
	PatternFirstLetterUpper = `^\p{Lu}.*$`
	// PatternFirstLetterLower requires the first character to be lowercase.
	PatternFirstLetterLower = `^\p{Ll}.*$`
	// PatternWords accepts Unicode words separated by spaces or tabs.
	PatternWords = `^\p{L}+(?:[ \t]\p{L}+)*$`

	// PatternLettersAndNumbers accepts Unicode letters and numbers only.
	PatternLettersAndNumbers = `^[\p{L}\p{N}]+$`
	// PatternAlphaNumeric is the concise name for PatternLettersAndNumbers.
	PatternAlphaNumeric = PatternLettersAndNumbers
	// PatternLettersNumbersAndSpaces accepts words made of letters/numbers.
	PatternLettersNumbersAndSpaces = `^[\p{L}\p{N}]+(?:[ \t][\p{L}\p{N}]+)*$`
	// PatternAlphaNumericWithSpaces is the concise name for its longer counterpart.
	PatternAlphaNumericWithSpaces = PatternLettersNumbersAndSpaces
	// PatternDigitsOnly accepts one or more ASCII decimal digits.
	PatternDigitsOnly = `^[0-9]+$`
	// PatternUnsignedInteger is the named integer equivalent of PatternDigitsOnly.
	PatternUnsignedInteger = PatternDigitsOnly
	// PatternSignedInteger accepts an optional leading minus sign and digits.
	PatternSignedInteger = `^-?[0-9]+$`
	// PatternDecimal accepts a signed decimal with one optional dot or comma.
	PatternDecimal = `^-?[0-9]+(?:[.,][0-9]+)?$`

	// PatternUppercaseCode accepts an ASCII code beginning with an uppercase letter.
	PatternUppercaseCode = `^[A-Z][A-Z0-9_-]*$`
	// PatternLowercaseCode accepts an ASCII code beginning with a lowercase letter.
	PatternLowercaseCode = `^[a-z][a-z0-9_-]*$`
	// PatternUpperCamelCase accepts an ASCII UpperCamelCase identifier.
	PatternUpperCamelCase = `^[A-Z][A-Za-z0-9]*$`
	// PatternLowerCamelCase accepts an ASCII lowerCamelCase identifier.
	PatternLowerCamelCase = `^[a-z][A-Za-z0-9]*$`
	// PatternSnakeCase accepts lowercase words separated by underscores.
	PatternSnakeCase = `^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`
	// PatternKebabCase accepts lowercase words separated by hyphens.
	PatternKebabCase = `^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`
	// PatternSlug is the readable name for the conventional lowercase slug shape.
	PatternSlug = PatternKebabCase
	// PatternDotSeparatedKey accepts lowercase dot-separated configuration keys.
	PatternDotSeparatedKey = `^[a-z][a-z0-9]*(?:\.[a-z0-9]+)*$`
	// PatternTwoLetterCode accepts exactly two ASCII uppercase letters.
	PatternTwoLetterCode = `^[A-Z]{2}$`
	// PatternThreeLetterCode accepts exactly three ASCII uppercase letters.
	PatternThreeLetterCode = `^[A-Z]{3}$`

	// PatternHexadecimal accepts hexadecimal digits with an optional 0x prefix.
	PatternHexadecimal = `^(?:0[xX])?[0-9A-Fa-f]+$`
	// PatternHexColor accepts CSS-style three- or six-digit hexadecimal colors.
	PatternHexColor = `^#(?:[0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`
	// PatternUUID accepts the canonical five-group UUID representation.
	PatternUUID = `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	// PatternVersion accepts one, two or three numeric dot-separated components.
	PatternVersion = `^[0-9]+(?:\.[0-9]+){0,2}$`
	// PatternTime24Hour accepts an hour and minute in 24-hour HH:MM format.
	PatternTime24Hour = `^(?:[01][0-9]|2[0-3]):[0-5][0-9]$`
	// PatternE164Phone accepts the international E.164 phone representation.
	PatternE164Phone = `^\+[1-9][0-9]{1,14}$`
	// PatternHTTPURL accepts an HTTP or HTTPS URL without whitespace.
	PatternHTTPURL = `^https?://\S+$`
	// PatternMACAddress accepts six hexadecimal octets separated by colons.
	PatternMACAddress = `^(?:[0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`
)
