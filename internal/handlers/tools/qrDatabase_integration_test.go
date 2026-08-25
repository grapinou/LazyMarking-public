package tools

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestRealScansMatchHistoricalDB(t *testing.T) {
	pdfPath := os.Getenv("LAZYMARKING_TEST_PDF")
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	if pdfPath == "" || dbPath == "" {
		t.Skip("LAZYMARKING_TEST_PDF and LAZYMARKING_TEST_DB must both be set; skipping private historical-data integration test")
	}

	corpus := readAndGroupScannedExams(t, pdfPath, "")
	validateHistoricalCorpus(t, corpus, 69, 23, 3)
	exams := corpus.Exams

	conn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("open historical database: %v", err)
	}
	defer conn.Close()
	queries := db.New(conn)
	ctx := context.Background()

	matchedPages := 0
	for _, exam := range exams {
		studentExamID := exam.StudentExamID
		examContent, err := queries.GetStudentContentExam(ctx, db.GetStudentContentExamParams{
			StudentExamID: studentExamID,
			UserID:        1,
		})
		if err != nil {
			t.Errorf("get content for student_exam_id %d: %v", studentExamID, err)
		} else {
			if got, want := examContent.PageTot, int64(3); got != want {
				t.Errorf("student_exam_id %d page_tot = %d, want %d", studentExamID, got, want)
			}

			var qcm config.QCM
			if err := json.Unmarshal([]byte(examContent.Content), &qcm); err != nil {
				t.Errorf("student_exam_id %d has invalid QCM JSON: %v", studentExamID, err)
			}
		}

		for pageNumber := int64(1); pageNumber <= 3; pageNumber++ {
			pageContent, err := queries.GetPageContent(ctx, db.GetPageContentParams{
				StudentExamID: studentExamID,
				Page:          pageNumber,
				UserID:        1,
			})
			if err != nil {
				t.Errorf("get page %d for student_exam_id %d: %v", pageNumber, studentExamID, err)
				continue
			}

			var content config.PageContent
			if err := json.Unmarshal([]byte(pageContent), &content); err != nil {
				t.Errorf("page %d for student_exam_id %d has invalid PageContent JSON: %v", pageNumber, studentExamID, err)
				continue
			}
			matchedPages++
		}
	}

	if got, want := matchedPages, 69; got != want {
		t.Errorf("database page entries with valid JSON = %d, want %d", got, want)
	}
}
