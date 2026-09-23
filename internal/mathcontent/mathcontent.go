// Package mathcontent separates literal question text from explicit Typst math.
// It has no PDF or HTTP dependency so the same interpretation can be reused by
// other question consumers.
package mathcontent

import (
	"fmt"
	"strings"
	"unicode"
)

type Segment struct {
	Text string
	Math bool
}

// Parse recognizes $...$ as Typst math and \$ as a literal dollar sign.
// Everything else remains literal text. Code interpolation and markup escapes
// are deliberately unavailable inside math: questions must not execute Typst.
func Parse(input string) ([]Segment, error) {
	var segments []Segment
	var text strings.Builder
	for i := 0; i < len(input); {
		if input[i] == '\\' && i+1 < len(input) && input[i+1] == '$' {
			text.WriteByte('$')
			i += 2
			continue
		}
		if input[i] != '$' {
			text.WriteByte(input[i])
			i++
			continue
		}
		if text.Len() > 0 {
			segments = append(segments, Segment{Text: text.String()})
			text.Reset()
		}
		start := i + 1
		i = start
		for i < len(input) && input[i] != '$' {
			i++
		}
		if i == len(input) {
			return nil, fmt.Errorf("formule Typst non fermée : ajoutez le second signe $")
		}
		formula := input[start:i]
		if strings.TrimSpace(formula) == "" {
			return nil, fmt.Errorf("formule Typst vide entre les signes $")
		}
		runes := []rune(formula)
		for index, r := range runes {
			if unicode.IsControl(r) && r != '\n' && r != '\t' || strings.ContainsRune("#\\[]{};`@", r) {
				return nil, fmt.Errorf("caractère %q non autorisé dans une formule Typst ; seuls les calculs entre $ sont acceptés", r)
			}
			// Typst's math mode also exposes the standard library as std.*.
			// Accept decimal points, never qualified names such as std.read.
			if r == '.' && (index == 0 || index+1 == len(runes) || !unicode.IsDigit(runes[index-1]) || !unicode.IsDigit(runes[index+1])) {
				return nil, fmt.Errorf("point non autorisé dans une formule Typst hors d’un nombre décimal")
			}
		}
		segments = append(segments, Segment{Text: formula, Math: true})
		i++
	}
	if text.Len() > 0 || len(segments) == 0 {
		segments = append(segments, Segment{Text: text.String()})
	}
	return segments, nil
}
