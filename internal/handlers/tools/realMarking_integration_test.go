package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestRealStudentExamMarkingSmoke(t *testing.T) {
	pdfOnePage := os.Getenv("LAZYMARKING_TEST_PDF_1_PAGE")
	pdfTwoPages := os.Getenv("LAZYMARKING_TEST_PDF_2_PAGES")
	pdfThreePages := os.Getenv("LAZYMARKING_TEST_PDF")
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	if pdfOnePage == "" || pdfTwoPages == "" || pdfThreePages == "" || dbPath == "" {
		t.Skip("private one-, two-, and three-page PDFs and historical DB must all be configured; skipping real-marking smoke test")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	t.Chdir(filepath.Clean(filepath.Join(workingDir, "../../..")))

	conn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("open historical database: %v", err)
	}
	defer conn.Close()
	queries := db.New(conn)

	testCases := []struct {
		name          string
		pdfPath       string
		studentExamID int64
		pageCount     int
	}{
		{name: "one_page", pdfPath: pdfOnePage, studentExamID: 781, pageCount: 1},
		{name: "two_pages", pdfPath: pdfTwoPages, studentExamID: 15, pageCount: 2},
		{name: "three_pages", pdfPath: pdfThreePages, studentExamID: 387, pageCount: 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			corpus := readAndGroupScannedExams(t, tc.pdfPath, newMarkingIntegrationTempDir(t))
			targetExam := findExamByID(t, corpus.Exams, tc.studentExamID)
			if got := len(targetExam.Pages); got != tc.pageCount {
				t.Fatalf("student_exam_id %d page count = %d, want %d", tc.studentExamID, got, tc.pageCount)
			}

			markExam, err := MarkingStudentExam(
				1,
				"integration-test",
				corpus.TempDir,
				targetExam,
				context.Background(),
				queries,
			)
			if err != nil {
				t.Fatalf("mark student_exam_id %d: %v", tc.studentExamID, err)
			}
			if !markExam.Status {
				t.Error("MarkExam.Status = false, want true")
			}
			if got := markExam.Pages; got != tc.pageCount {
				t.Errorf("MarkExam.Pages = %d, want %d", got, tc.pageCount)
			}
			if markExam.Total <= 0 {
				t.Errorf("MarkExam.Total = %d, want > 0", markExam.Total)
			}
			if markExam.Score < 0 {
				t.Errorf("MarkExam.Score = %v, want >= 0", markExam.Score)
			}
			if markExam.Score > float64(markExam.Total) {
				t.Errorf("MarkExam.Score = %v, want <= Total (%d)", markExam.Score, markExam.Total)
			}
		})
	}
}

func newMarkingIntegrationTempDir(t *testing.T) string {
	t.Helper()
	tempDir, err := os.MkdirTemp(".", ".real-marking-integration-*")
	if err != nil {
		t.Fatalf("create integration-test directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("remove integration-test directory: %v", err)
		}
	})

	tempDir, err = filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("resolve integration-test directory: %v", err)
	}
	return tempDir
}

func findExamByID(t *testing.T, exams []config.Exam, studentExamID int64) config.Exam {
	t.Helper()
	for _, exam := range exams {
		if exam.StudentExamID == studentExamID {
			return exam
		}
	}
	t.Fatalf("student_exam_id %d was not found in scanned exams", studentExamID)
	return config.Exam{}
}
