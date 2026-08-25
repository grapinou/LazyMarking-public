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
	pdfPath := os.Getenv("LAZYMARKING_TEST_PDF")
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	if pdfPath == "" || dbPath == "" {
		t.Skip("LAZYMARKING_TEST_PDF and LAZYMARKING_TEST_DB must both be set; skipping private real-marking smoke test")
	}
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	t.Chdir(filepath.Clean(filepath.Join(workingDir, "../../..")))

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

	exams, tempDir := readAndGroupScannedExams(t, pdfPath, tempDir)
	var targetExam config.Exam
	for _, exam := range exams {
		if exam.StudentExamID == 387 {
			targetExam = exam
			break
		}
	}
	if targetExam.StudentExamID == 0 {
		t.Fatal("student_exam_id 387 was not found in scanned exams")
	}
	if got, want := len(targetExam.Pages), 3; got != want {
		t.Fatalf("student_exam_id 387 page count = %d, want %d", got, want)
	}

	conn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("open historical database: %v", err)
	}
	defer conn.Close()

	markExam, err := MarkingStudentExam(
		1,
		"integration-test",
		tempDir,
		targetExam,
		context.Background(),
		db.New(conn),
	)
	if err != nil {
		t.Fatalf("mark student_exam_id 387: %v", err)
	}
	if !markExam.Status {
		t.Error("MarkExam.Status = false, want true")
	}
	if got, want := markExam.Pages, 3; got != want {
		t.Errorf("MarkExam.Pages = %d, want %d", got, want)
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
}
