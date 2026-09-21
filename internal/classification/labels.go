// Package classification defines typographic equivalence, never semantic aliases.
package classification

import (
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// Key is for comparisons only. Stored/displayed labels must remain unchanged.
// Canonical Unicode decomposition, case folding and removal of combining marks
// make accented and decomposed forms equivalent. Fields trims and collapses
// Unicode whitespace. Punctuation, including hyphens, remains significant.
func Key(label string) string {
	value := norm.NFD.String(cases.Fold().String(label))
	value = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, value)
	return norm.NFC.String(strings.Join(strings.Fields(value), " "))
}
