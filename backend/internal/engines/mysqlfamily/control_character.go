package mysqlfamily

import (
	"strings"
	"unicode"
)

// hasControlCharacter reports a line break or another control character. In a value that reaches
// the client tools, a line break starts a new line of the .my.cnf file, and the tools read that
// line as one more option.
func hasControlCharacter(value string) bool {
	return strings.ContainsFunc(value, unicode.IsControl)
}
