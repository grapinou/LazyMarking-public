package tools

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TypstWriterLandscape(tempDir, username string, qcm config.QCM) (string, bool) {
	refQCMTypst := config.RefQCMLandscapeTypst // fichier existant

	// 1. Ouvrir l'ancien fichier pour lecture
	input, err := os.Open(refQCMTypst)
	if err != nil {
		log.Printf("Can't open file from TypstWriterLandscape : %s, error : %v", refQCMTypst, err)
		return "", false
	}
	defer input.Close()

	// 2. Créer le nouveau fichier (écrasement s’il existe)
	typstFilePath := filepath.Join(tempDir, fmt.Sprintf("%s%v", username, config.PreviewLandscapeQCM))
	output, err := os.Create(typstFilePath)
	if err != nil {
		log.Printf("Can't create file from TypstWriterLandscape: %s, error : %v", typstFilePath, err)
		return "", false
	}
	defer output.Close()

	// 4. Copier le contenu de l’ancien fichier
	_, err = io.Copy(output, input)
	if err != nil {
		log.Printf("Can't copy ref file into from TypstWriterLandscape: %s, error : %v", typstFilePath, err)
		return "", false
	}

	name := "#list(spacing: 15pt, [Prénom + Nom : ], [Classe : ],)"

	// 5. Ajouter des lignes à la fin
	for _, question := range qcm.Questions {
		questionTypst := fmt.Sprintf("#let question=%s", typstStringLiteral(question.Content))
		imageTypst := "#let monimage=\"\""
		if question.Image.Name != "" {
			imagePath, err := typstImagePath(question.Image.Name)
			if err != nil {
				log.Printf("Can't resolve Typst image path: %v", err)
				return "", false
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

		content := "\n" + name + "\n" + questionTypst + "\n" + imageTypst + "\n" + tableQuestionTypst + "\n" + tableAnswersTypst + "\n" + answersTypst + "\n\n\n"
		_, err = output.WriteString(content)
		if err != nil {
			log.Printf("Can't write content from TypstWriterLandscape, error : %v", err)
			return "", false
		}
	}
	nextColumn := "#colbreak()" + "\n\n\n"
	_, err = output.WriteString(nextColumn)
	if err != nil {
		log.Printf("Can't write content from TypstWriterLandscape, error : %v", err)
		return "", false
	}

	return typstFilePath, true
}
