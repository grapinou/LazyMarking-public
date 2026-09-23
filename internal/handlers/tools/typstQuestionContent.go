package tools

import (
	"fmt"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/mathcontent"
)

// typstQuestionText preserves the old literal representation when there is no
// math. Mixed content uses escaped text nodes and validated Typst math only.
func typstQuestionText(value string) (string, error) {
	segments, err := mathcontent.Parse(value)
	if err != nil {
		return "", err
	}
	if len(segments) == 1 && !segments[0].Math && segments[0].Text == value {
		return typstStringLiteral(value), nil
	}
	var out strings.Builder
	out.WriteByte('[')
	for _, segment := range segments {
		if segment.Math {
			out.WriteByte('$')
			out.WriteString(segment.Text)
			out.WriteByte('$')
		} else if segment.Text != "" {
			out.WriteString("#text(")
			out.WriteString(typstStringLiteral(segment.Text))
			out.WriteByte(')')
		}
	}
	out.WriteByte(']')
	return out.String(), nil
}

// All current previews and printed copies use the same question layout.
// The legacy renderer stays frozen for snapshots predating layout_version.
func typstQuestionContent(question config.Question) (string, error) {
	var out strings.Builder
	questionText, err := typstQuestionText(question.Content)
	if err != nil {
		return "", fmt.Errorf("énoncé : %w", err)
	}
	instructionText, err := typstQuestionText(question.Instruction)
	if err != nil {
		return "", fmt.Errorf("consigne : %w", err)
	}
	fmt.Fprintf(&out, "\n#let question=%s\n", questionText)
	fmt.Fprintf(&out, "#let instruction=%s\n", instructionText)
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
		answerText, err := typstQuestionText(answer.Content)
		if err != nil {
			return "", fmt.Errorf("réponse %s : %w", answer.Symbol, err)
		}
		fmt.Fprintf(&out, "answer(\"%s\", %s),\n", answer.Symbol, answerText)
	}
	out.WriteString(")\n\n")
	return out.String(), nil
}
