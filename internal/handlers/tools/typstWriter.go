package tools

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TypstWriter(username string, qcm config.QCM, filenameQCM config.QCMType) (string, bool) {
	tempDir, ok := CreateUserTempDir(username)
	if !ok {
		log.Println("From TypstWriter -> CreateUserTempDir return not ok")
		return "", false
	}
	refQCMTypst := config.RefQCMTypst // fichier existant

	// 1. Ouvrir l'ancien fichier pour lecture
	input, err := os.Open(refQCMTypst)
	if err != nil {
		log.Printf("Can't open file : %s, error : %v", refQCMTypst, err)
		return "", false
	}
	defer input.Close()

	// 2. Créer le nouveau fichier (écrasement s’il existe)

	typstFilePath := filepath.Join(tempDir, fmt.Sprintf("%s%v", username, filenameQCM))
	output, err := os.Create(typstFilePath)
	if err != nil {
		log.Printf("Can't create file : %s, error : %v", typstFilePath, err)
		return "", false
	}
	defer output.Close()

	// 3. Écrire une ligne au début
	student := fmt.Sprintf("#let student=\"%s\"\n", qcm.Student)
	_, err = output.WriteString(student)
	if err != nil {
		log.Printf("can't write : %s, error : %v", student, err)
		return "", false
	}

	// 4. Copier le contenu de l’ancien fichier
	_, err = io.Copy(output, input)
	if err != nil {
		log.Printf("Can't copy ref file into : %s, error : %v", typstFilePath, err)
		return "", false
	}

	// 5. Ajouter des lignes à la fin
	for _, question := range qcm.Questions {
		questionTypst := fmt.Sprintf("#let question=\"%s\"", question.Content)
		imageTypst := "#let monimage=\"\""
		if question.Image.Name != "" {
			imagePath := filepath.Join(config.ImagePathTypst, question.Image.Name)
			imageTypst = fmt.Sprintf("#let monimage=[#figure(image(\"%s\", width: %s%%))]", imagePath, question.Image.Width)
			// typst image : [#figure(image("Sighto_Calcul_alt_IMG_7882.JPG", width: 50%))]
		}
		tableQuestionTypst := "#table(columns: (auto, auto, auto),stroke: none, circle(radius: 8pt, fill: black),text(baseline: 3pt)[#question], text(baseline: 3pt)[#monimage])"
		tableAnswersTypst := "#let answer(symbo, ans)=[#table(columns: (auto, auto),stroke: none,  text(2.5em, baseline: -6pt)[#symbo], [#ans])]"
		answersTypst := "#table(columns: (auto, auto),stroke: none,"
		for _, answer := range question.Answers {
			answersTypst += fmt.Sprintf("answer(\"%s\", \"%s\"),", answer.Symbol, answer.Content)
		}
		answersTypst += ")"

		content := "\n" + questionTypst + "\n" + imageTypst + "\n" + tableQuestionTypst + "\n" + tableAnswersTypst + "\n" + answersTypst + "\n\n\n"
		_, err = output.WriteString(content)
		if err != nil {
			log.Printf("Can't write content, error : %v", err)
			return "", false
		}
	}
	return typstFilePath, true
}
