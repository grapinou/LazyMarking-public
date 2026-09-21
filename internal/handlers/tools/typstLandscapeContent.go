package tools

import (
	"fmt"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TypstLandscapeContent(qcm config.QCM) (string, error) {
	var builder strings.Builder
	fmt.Fprintf(&builder, "\n#text(weight: \"bold\", %s)\n", typstStringLiteral(qcm.Name))
	builder.WriteString("#list(spacing: 8pt, [Prénom + Nom : ], [Classe : ")
	builder.WriteString("#" + typstStringLiteral(qcm.Student.ClassCodes.Name))
	builder.WriteString("],)\n")
	builder.WriteString("#text(8pt)[Répondez au stylo bleu ou noir. Coloriez complètement le/les cercle(s) correspondant(s) à votre/vos réponse(s).]\n")
	for _, question := range qcm.Questions {
		content, err := typstQuestionContent(question)
		if err != nil {
			return "", err
		}
		builder.WriteString(content)
	}
	// A weak break separates copies without creating an empty trailing column/page.
	builder.WriteString("#colbreak(weak: true)\n")
	return builder.String(), nil
}
