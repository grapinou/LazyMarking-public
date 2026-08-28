package tools

import (
	"errors"
	"fmt"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TypstLandscapeContent(qcm config.QCM) (string, error) {
	var builder strings.Builder

	name := "#list(spacing: 15pt, [Prénom + Nom : ], [Classe : ],)"
	builder.WriteString("\n")
	builder.WriteString(name)

	for _, question := range qcm.Questions {
		questionTypst := fmt.Sprintf("#let question=%s", typstStringLiteral(question.Content))
		imageTypst := "#let monimage=\"\""
		if question.Image.Name != "" {
			imagePath, err := typstImagePath(question.Image.Name)
			if err != nil {
				return "", errors.New("resolve landscape Typst image path: " + err.Error())
			}
			imageTypst = fmt.Sprintf("#let monimage=[#figure(image(%s, width: %s%%))]", typstStringLiteral(imagePath), question.Image.Width)
		}
		tableQuestionTypst := "#table(columns: (auto, auto, auto),stroke: none, circle(radius: 8pt, fill: black),text(baseline: 3pt)[#question], text(baseline: 3pt)[#monimage])"
		tableAnswersTypst := "#let answer(symbo, ans)=[#table(columns: (auto, auto),stroke: none,  text(2.5em, baseline: -6pt)[#symbo], [#ans])]"
		answersTypst := "#table(columns: (auto, auto),stroke: none,"
		for _, answer := range question.Answers {
			answersTypst += fmt.Sprintf("answer(\"%s\", %s),", answer.Symbol, typstStringLiteral(answer.Content))
		}
		answersTypst += ")"

		builder.WriteString("\n")
		builder.WriteString(questionTypst)
		builder.WriteString("\n")
		builder.WriteString(imageTypst)
		builder.WriteString("\n")
		builder.WriteString(tableQuestionTypst)
		builder.WriteString("\n")
		builder.WriteString(tableAnswersTypst)
		builder.WriteString("\n")
		builder.WriteString(answersTypst)
		builder.WriteString("\n\n\n")

	}

	builder.WriteString("#colbreak()\n\n\n")

	return builder.String(), nil
}
