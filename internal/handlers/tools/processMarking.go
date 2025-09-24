package tools

import (
	"context"
	"database/sql"
	"io"
	"log"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/db"
)

func ProcessMarking(userID int64, username string, jobDBID int64, file io.Reader, queries *db.Queries) {
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

	ctx := context.Background()

	if err := queries.UpdateMarkingJobTotalPages(ctx, db.UpdateMarkingJobTotalPagesParams{
		TotalPages: sql.NullInt64{
			Int64: int64(len(pages)),
			Valid: true,
		},
		ID:     jobDBID,
		UserID: userID,
	}); err != nil {
		log.Printf("From ProcessingMarkingHandler -> queries.UpdateMarkingJobTotalExam DB error : %v", err)
		return
	}

	// Ici on lance les sous-goroutines pour traiter les pages
	qrDatas, qrNotDetected, err := ProcessPagesConcurrently(pages, tempDir, queries, ctx, jobDBID, userID)
	if err != nil {
		log.Printf("ProcessPagesConcurrently error: %v", err)
		if err := queries.UpdateMarkingJobStatus(ctx, db.UpdateMarkingJobStatusParams{
			Status: "failed",
			ID:     jobDBID,
			UserID: userID,
		}); err != nil {
			log.Printf("From UpdateMarkingJobStatus Db error : %v", err)
		}
		return
	}

	if err := RemoveFiles(pages); err != nil {
		log.Printf("From ProcessingMarkingHandler -> RemoveFiles return error : %v", err)
		return
	}

	exams := GroupQrCodes(qrDatas)

	if err := queries.UpdateMarkingJobTotalExam(ctx, db.UpdateMarkingJobTotalExamParams{
		TotalExams: sql.NullInt64{
			Int64: int64(len(exams)),
			Valid: true,
		},
		ID:     jobDBID,
		UserID: userID,
	}); err != nil {
		log.Printf("From ProcessingMarkingHandler -> queries.UpdateMarkingJobTotalExam DB error : %v", err)
		return
	}

	// 3. Traitement des examens en parallèle
	markExams, notMarkedExams, err := ProcessExamsConcurrently(exams, userID, username, tempDir, ctx, queries, jobDBID)
	if err != nil {
		log.Printf("ProcessExamsConcurrently error: %v", err)
		if err := queries.UpdateMarkingJobStatus(ctx, db.UpdateMarkingJobStatusParams{
			Status: "failed",
			ID:     jobDBID,
			UserID: userID,
		}); err != nil {
			log.Printf("From UpdateMarkingJobStatus Db error : %v", err)
		}
		return
	}

	pdfFiles, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> GetAllFiles return error : %v", err)
		return
	}

	// faire un sommaire
	typstPath, ok := TypstBuildContent(tempDir, markExams, pdfFiles)
	if !ok {
		log.Println("can't build content")
		return
	}

	pdfContent, ok := CompileTypst(typstPath)
	if !ok {
		log.Println("can't make pdf from typstPath")
		return
	}

	pdfFiles = append([]string{pdfContent}, pdfFiles...)

	examName := markExams[0].ExamName
	className := markExams[0].ClassName
	name := filepath.Join(tempDir, examName+"_"+className+"_corrected.pdf")
	if err := MergePdf(pdfFiles, name); err != nil {
		log.Println("can't merge pdf")
	}

	pdfFiles = append(pdfFiles, typstPath)

	if err := queries.UpdateMarkingJobStatus(ctx, db.UpdateMarkingJobStatusParams{
		Status: "success",
		ID:     jobDBID,
		UserID: userID,
	}); err != nil {
		log.Printf("From UpdateMarkingJobStatus Db error : %v", err)
	}

	globalSkills, globalThemeSkills := AgregateThemeSkill(markExams)

	mean, stdDev, median := ComputeStatMarking(markExams)

	typstMarkTablePath, ok := TypstBuildMarkTable(tempDir, markExams, mean, stdDev, median, globalSkills, globalThemeSkills, qrNotDetected, notMarkedExams)
	if !ok {
		log.Println("can't build mark table")
		return
	}

	markName, ok := CompileTypst(typstMarkTablePath)
	if !ok {
		log.Println("can't make pdf from typstMarkTablePath")
		return
	}

	pdfFiles = append(pdfFiles, typstMarkTablePath)
	if err := RemoveFiles(pdfFiles); err != nil {
		log.Printf("From ProcessingMarkingHandler -> RemoveFiles return error : %v", err)
		return
	}

	if err := queries.UpdateMarkingJobStatusPDF(ctx, db.UpdateMarkingJobStatusPDFParams{
		StatusPdf: "success",
		ExamName: sql.NullString{
			String: name,
			Valid:  true,
		},
		MarkTableName: sql.NullString{
			String: markName,
			Valid:  true,
		},

		ID:     jobDBID,
		UserID: userID,
	}); err != nil {
		log.Printf("From UpdateMarkingJobStatusPDF Db error : %v", err)
	}
}
