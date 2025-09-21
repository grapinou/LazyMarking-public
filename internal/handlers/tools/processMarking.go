package tools

import (
	"context"
	"io"
	"log"

	"github.com/grapinou/LazyMarking/internal/db"
)

func ProcessMarking(jobID string, userID int64, username string, file io.Reader, queries *db.Queries) {
	tempDir, ok := CreateUserTempDir(username)
	if !ok {
		log.Println("From ProcessingMarkingHandler -> CreateUserTempDir return not ok")
		return
	}

	if err := ClearDir(tempDir); err != nil {
		log.Printf("From ProcessingMarkingHandler -> ClearDir return error : %v", err)
		return
	}

	if err := SplitPdf(file, tempDir, "page-%d.pdf"); err != nil {
		log.Printf("From ProcessingMarkingHandler -> SplitPdf return error : %v", err)
		return
	}

	pages, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> GetAllFiles return error : %v", err)
		return
	}

	// commencer ici la go routine ?

	ctx := context.Background()
	// Ici on lance les sous-goroutines pour traiter les pages
	qrDatas, qrNotDetected := ProcessPagesConcurrently(pages, tempDir)

	if err := RemoveFiles(pages); err != nil {
		log.Printf("From ProcessingMarkingHandler -> RemoveFiles return error : %v", err)
		return
	}

	exams := GroupQrCodes(qrDatas)

	// 3. Traitement des examens en parallèle
	markExams, notMarkedExams := ProcessExamsConcurrently(exams, userID, username, tempDir, ctx, queries)

	log.Printf("Job %s: Done! Success=%d Failed=%d QRNotDetected=%d",
		jobID, len(markExams), len(notMarkedExams), len(qrNotDetected))
}
