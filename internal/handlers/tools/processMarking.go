package tools

import (
	"context"
	"database/sql"
	"io"
	"log"
	"path/filepath"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
)

func ProcessMarking(ctx context.Context, userID int64, username string, jobDBID int64, file io.Reader, queries *db.Queries) {
	operation := "marking-" + strconv.FormatInt(jobDBID, 10)
	tempDir, ok := CreateOperationTempDir(username, operation)
	if !ok {
		log.Println("From ProcessingMarkingHandler -> CreateUserTempDir return not ok")
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	if err := SplitPdf(file, tempDir, "page-%d.pdf"); err != nil {
		log.Printf("From ProcessingMarkingHandler -> SplitPdf return error : %v", err)
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	pages, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> GetAllFiles return error : %v", err)
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}
	if err := queries.UpdateMarkingJobTotalPages(ctx, db.UpdateMarkingJobTotalPagesParams{
		TotalPages: sql.NullInt64{
			Int64: int64(len(pages)),
			Valid: true,
		},
		ID:     jobDBID,
		UserID: userID,
	}); err != nil {
		log.Printf("From ProcessingMarkingHandler -> queries.UpdateMarkingJobTotalExam DB error : %v", err)
		MarkingFailed(userID, jobDBID, ctx, queries)
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

	if len(qrDatas) == 0 {
		log.Println("No qrDatas found, can't make marking process")
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	if err := RemoveFiles(pages); err != nil {
		log.Printf("From ProcessingMarkingHandler -> RemoveFiles return error : %v", err)
		MarkingFailed(userID, jobDBID, ctx, queries)
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
		MarkingFailed(userID, jobDBID, ctx, queries)
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
	if len(markExams) == 0 {
		log.Println("No exams could be marked")
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	pdfFiles, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> GetAllFiles return error : %v", err)
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	// faire un sommaire
	typstPath, ok := TypstBuildContent(tempDir, markExams, pdfFiles)
	if !ok {
		log.Println("can't build content")
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	pdfContent, ok := CompileTypst(typstPath)
	if !ok {
		log.Println("can't make pdf from typstPath")
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	pdfFiles = append([]string{pdfContent}, pdfFiles...)

	examName := markExams[0].ExamName
	className := markExams[0].ClassName
	name := filepath.Join(tempDir, examName+"_"+className+"_corrected.pdf")
	if err := MergePdf(pdfFiles, name); err != nil {
		log.Println("can't merge pdf")
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	pdfFiles = append(pdfFiles, typstPath)

	if err := queries.UpdateMarkingJobStatus(ctx, db.UpdateMarkingJobStatusParams{
		Status: "success",
		ID:     jobDBID,
		UserID: userID,
	}); err != nil {
		log.Printf("From UpdateMarkingJobStatus Db error : %v", err)
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	globalSkills, globalThemeSkills := AgregateThemeSkill(markExams)

	mean, stdDev, median := ComputeStatMarking(markExams)

	typstMarkTablePath, ok := TypstBuildMarkTable(tempDir, markExams, mean, stdDev, median, globalSkills, globalThemeSkills, qrNotDetected, notMarkedExams)
	if !ok {
		log.Println("can't build mark table")
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	markName, ok := CompileTypst(typstMarkTablePath)
	if !ok {
		log.Println("can't make pdf from typstMarkTablePath")
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}

	pdfFiles = append(pdfFiles, typstMarkTablePath)
	if err := RemoveFiles(pdfFiles); err != nil {
		log.Printf("From ProcessingMarkingHandler -> RemoveFiles return error : %v", err)
		MarkingFailed(userID, jobDBID, ctx, queries)
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
		MarkingFailed(userID, jobDBID, ctx, queries)
		return
	}
}
