package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func MarkingStudentExam(userID int64, username, tempDir string, exam config.Exam, ctx context.Context, queries *db.Queries) {
	// re-construire l'exam de base avec typst et faire le png
	datas, err := queries.GetStudentContentExam(ctx, db.GetStudentContentExamParams{
		StudentExamID: exam.StudentExamID,
		UserID:        userID,
	})
	if err != nil {
		log.Printf("From MarkingStudentExam -> GetStudentContentExam return error : %v", err)
		return
	}

	// vérification du nombre de pages
	if len(exam.Pages) != int(datas.PageTot) {
		log.Println("From MarkingStudentExam -> exam not treated because pages is missing")
		return
	}

	// unmarchal qcm
	var qcm config.QCM
	if err := json.Unmarshal([]byte(datas.Content), &qcm); err != nil {
		log.Printf("From MarkingStudentExam -> Unmarshal return error : %v", err)
		return
	}

	typstFilePath, ok := TypstWriter(username, qcm, config.ExamQCM)
	if !ok {
		log.Println("TypstWriter return not ok")
	}

	pages, ok := ExportTypstToPNGs(typstFilePath)
	if !ok {
		log.Println("ExportTypstToPNGs return not ok")
	}

	// tri des pages par mesure de sécurité
	var pagesNumbers []int
	for _, n := range exam.Pages {
		pagesNumbers = append(pagesNumbers, n.Number)
	}
	sort.Ints(pagesNumbers)

	var answersState []int
	for i, page := range pages {

		imgName := filepath.Base(page)

		pagedatas, err := queries.GetPageContent(ctx, db.GetPageContentParams{
			StudentExamID: exam.StudentExamID,
			Page:          int64(pagesNumbers[i]),
			UserID:        userID,
		})
		if err != nil {
			log.Printf("From MarkingStudentExam -> GetPageContent return error : %v", err)
			return
		}

		var pageContent config.PageContent
		if err := json.Unmarshal([]byte(pagedatas), &pageContent); err != nil {
			log.Printf("From MarkingStudentExam -> Unmarshal return error : %v", err)
			return
		}

		// DrawCircleOnQcm(tempDir, imgName, "sur_png_", pageContent.Questions, pageContent.Answers)

		// on s'assure de prendre la bonne page
		var pageName string

		for _, p := range exam.Pages {
			if pagesNumbers[i] == p.Number {
				pageName = p.Name
			}
		}

		// DrawCircleOnQcm(tempDir, pageName, "sur_png_", pageContent.Questions, pageContent.Answers)

		// homographie de la page sur le png de typst
		homoName := Homography(tempDir, pageName, imgName)

		// DrawCircleOnQcm(tempDir, homoName, "sur_homo_", pageContent.Questions, pageContent.Answers)

		// sur l'homographie, regarder les réponses eleves

		state := GetAnswersState(tempDir, homoName, pageContent.Answers)

		answersState = append(answersState, state...)

	}

	// faire une liste question - reponse et compararer à au réponses (dans la db)
	fmt.Println(answersState)
}
