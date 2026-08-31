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
	markingFailed := func() {
		if err := MarkingFailed(userID, jobDBID, ctx, queries); err != nil {
			log.Printf("From MarkingFailed: %v", err)
		}
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("From ProcessMarking -> recovered panic: %v", recovered)
			markingFailed()
		}
	}()

	operation := "marking-" + strconv.FormatInt(jobDBID, 10)
	tempDir, ok := CreateOperationTempDir(username, operation)
	if !ok {
		log.Println("From ProcessingMarkingHandler -> CreateOperationTempDir return not ok")
		markingFailed()
		return
	}

	completed := false
	defer func() {
		if !completed {
			if err := RemoveOperationTempDir(username, operation); err != nil {
				log.Printf("From ProcessMarking -> failed to clean workspace %s: %v", tempDir, err)
			}
		}
	}()

	expectedRows, err := queries.ListExpectedStudentExamsForMarkingJob(ctx, db.ListExpectedStudentExamsForMarkingJobParams{
		MarkingJobID: jobDBID,
		UserID:       userID,
	})
	if err != nil || len(expectedRows) == 0 {
		log.Printf("From ProcessMarking -> cannot load expected copies: %v", err)
		markingFailed()
		return
	}
	expectedPages := make(map[int64]int64, len(expectedRows))
	for _, expected := range expectedRows {
		expectedPages[expected.StudentExamID] = expected.ExpectedPages
	}

	if err := SplitPdf(file, tempDir, "page-%d.pdf"); err != nil {
		log.Printf("From ProcessingMarkingHandler -> SplitPdf return error : %v", err)
		markingFailed()
		return
	}

	pages, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> GetAllFiles return error : %v", err)
		markingFailed()
		return
	}
	rows, err := queries.UpdateMarkingJobTotalPages(ctx, db.UpdateMarkingJobTotalPagesParams{
		TotalPages: sql.NullInt64{
			Int64: int64(len(pages)),
			Valid: true,
		},
		ID:     jobDBID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> queries.UpdateMarkingJobTotalExam DB error : %v", err)
		markingFailed()
		return
	}
	if rows != 1 {
		log.Printf("From ProcessingMarkingHandler -> UpdateMarkingJobTotalPages affected %d rows for job %d", rows, jobDBID)
		markingFailed()
		return
	}

	// Ici on lance les sous-goroutines pour traiter les pages
	qrDatas, qrNotDetected, err := ProcessPagesConcurrently(pages, tempDir, queries, ctx, jobDBID, userID)
	if err != nil {
		log.Printf("ProcessPagesConcurrently error: %v", err)
		markingFailed()
		return
	}

	if len(qrDatas) == 0 {
		if err := persistNotSeenCopies(ctx, queries, userID, jobDBID, expectedPages, nil); err != nil {
			log.Printf("From ProcessMarking -> persist not-seen copies: %v", err)
		}
		log.Println("No qrDatas found, can't make marking process")
		markingFailed()
		return
	}

	if err := RemoveFiles(pages); err != nil {
		log.Printf("From ProcessingMarkingHandler -> RemoveFiles return error : %v", err)
		markingFailed()
		return
	}

	exams := GroupQrCodes(qrDatas)

	rows, err = queries.UpdateMarkingJobTotalExam(ctx, db.UpdateMarkingJobTotalExamParams{
		TotalExams: sql.NullInt64{
			Int64: int64(len(exams)),
			Valid: true,
		},
		ID:     jobDBID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> queries.UpdateMarkingJobTotalExam DB error : %v", err)
		markingFailed()
		return
	}
	if rows != 1 {
		log.Printf("From ProcessingMarkingHandler -> UpdateMarkingJobTotalExam affected %d rows for job %d", rows, jobDBID)
		markingFailed()
		return
	}

	// 3. Traitement des examens en parallèle
	markExams, notMarkedExams, err := ProcessExamsConcurrently(exams, userID, username, tempDir, ctx, queries, jobDBID, expectedPages)
	if err != nil {
		log.Printf("ProcessExamsConcurrently error: %v", err)
		markingFailed()
		return
	}
	seenStudentExams := make(map[int64]struct{}, len(exams))
	for _, exam := range exams {
		seenStudentExams[exam.StudentExamID] = struct{}{}
	}
	if err := persistNotSeenCopies(ctx, queries, userID, jobDBID, expectedPages, seenStudentExams); err != nil {
		log.Printf("From ProcessMarking -> persist not-seen copies: %v", err)
		markingFailed()
		return
	}
	if len(markExams) == 0 {
		log.Println("No exams could be marked")
		markingFailed()
		return
	}

	pdfFiles, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> GetAllFiles return error : %v", err)
		markingFailed()
		return
	}

	// faire un sommaire
	typstPath, ok := TypstBuildContent(tempDir, markExams, pdfFiles)
	if !ok {
		log.Println("can't build content")
		markingFailed()
		return
	}

	pdfContent, ok := CompileTypst(typstPath)
	if !ok {
		log.Println("can't make pdf from typstPath")
		markingFailed()
		return
	}

	pdfFiles = append([]string{pdfContent}, pdfFiles...)

	name := filepath.Join(tempDir, "corrected.pdf")
	if err := MergePdf(pdfFiles, name); err != nil {
		log.Println("can't merge pdf")
		markingFailed()
		return
	}

	pdfFiles = append(pdfFiles, typstPath)

	globalSkills, globalThemeSkills := AgregateThemeSkill(markExams)

	mean, stdDev, median := ComputeStatMarking(markExams)

	typstMarkTablePath, ok := TypstBuildMarkTable(tempDir, markExams, mean, stdDev, median, globalSkills, globalThemeSkills, qrNotDetected, notMarkedExams)
	if !ok {
		log.Println("can't build mark table")
		markingFailed()
		return
	}

	markName, ok := CompileTypst(typstMarkTablePath)
	if !ok {
		log.Println("can't make pdf from typstMarkTablePath")
		markingFailed()
		return
	}

	pdfFiles = append(pdfFiles, typstMarkTablePath)
	if err := RemoveFiles(pdfFiles); err != nil {
		log.Printf("From ProcessingMarkingHandler -> RemoveFiles return error : %v", err)
		markingFailed()
		return
	}

	rows, err = queries.CompleteMarkingJobWithResults(ctx, db.CompleteMarkingJobWithResultsParams{
		ExamName: sql.NullString{
			String: name,
			Valid:  true,
		},
		MarkTableName: sql.NullString{
			String: markName,
			Valid:  true,
		},

		ID:                      jobDBID,
		UserID:                  userID,
		ResultSchemaVersion:     sql.NullInt64{Int64: MarkingResultSchemaVersion, Valid: true},
		MarkingAlgorithmVersion: sql.NullString{String: MarkingAlgorithmVersion, Valid: true},
		DetectionThreshold:      sql.NullFloat64{Float64: MarkingDetectionThreshold, Valid: true},
	})
	if err != nil {
		log.Printf("From CompleteMarkingJob DB error : %v", err)
		markingFailed()
		return
	}
	if rows != 1 {
		log.Printf("From CompleteMarkingJob -> affected %d rows for job %d", rows, jobDBID)
		markingFailed()
		return
	}

	completed = true
}

func persistNotSeenCopies(ctx context.Context, queries *db.Queries, userID, jobID int64, expectedPages map[int64]int64, seen map[int64]struct{}) error {
	for studentExamID, pageCount := range expectedPages {
		if _, ok := seen[studentExamID]; ok {
			continue
		}
		if _, err := db.PersistTerminalMarkingCopy(ctx, queries, db.PersistedTerminalMarkingCopyInput{
			UserID: userID, MarkingJobID: jobID, StudentExamID: studentExamID,
			Outcome: "not_seen", ExpectedPages: pageCount, DetectedPages: 0, FailureCode: "no_qr_pages",
		}); err != nil {
			return err
		}
	}
	return nil
}
