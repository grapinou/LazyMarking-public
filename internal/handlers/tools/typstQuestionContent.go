package tools

import (
	"fmt"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
)

// All current previews and printed copies use the same question layout.
// The legacy renderer stays frozen for snapshots predating layout_version.
func typstQuestionContent(question config.Question) (string, error) {
	var out strings.Builder
	fmt.Fprintf(&out, "\n#let question=%s\n", typstStringLiteral(question.Content))
	fmt.Fprintf(&out, "#let instruction=%s\n", typstStringLiteral(question.Instruction))
	out.WriteString("#let monimage=\"\"\n")
	if question.Image.Name != "" {
		path, err := typstImagePath(question.Image.Name)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&out, "#let monimage=[#image(%s, width: %s%%, height: 5cm, fit: \"contain\")]\n", typstStringLiteral(path), question.Image.Width)
	}
	out.WriteString(`#block(sticky: true)[
  #table(columns: (20pt, 1fr), stroke: none,
    circle(radius: 8pt, fill: black),
    [#set par(justify: false)
     #if instruction != "" [#block(sticky: true, above: 0pt, below: 4pt)[#text(weight: "bold")[#instruction]]]
     #question
     #if monimage != "" [#parbreak() #monimage]])
]
#let answer(symbo, ans)=[#table(columns: (25pt, 1fr), stroke: none, text(2.5em, baseline: -6pt)[#symbo], [#ans])]
#table(columns: (1fr, 1fr), stroke: none,
`)
	for _, answer := range question.Answers {
		fmt.Fprintf(&out, "answer(\"%s\", %s),\n", answer.Symbol, typstStringLiteral(answer.Content))
	}
	out.WriteString(")\n\n")
	return out.String(), nil
}
