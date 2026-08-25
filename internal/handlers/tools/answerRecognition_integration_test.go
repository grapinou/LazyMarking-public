package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestRealOnePageAnswerRecognition(t *testing.T) {
	pdfPath := os.Getenv("LAZYMARKING_TEST_PDF_1_PAGE")
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	if pdfPath == "" || dbPath == "" {
		t.Skip("LAZYMARKING_TEST_PDF_1_PAGE and LAZYMARKING_TEST_DB must both be set; skipping private answer-recognition integration test")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	t.Chdir(filepath.Clean(filepath.Join(workingDir, "../../..")))

	corpus := readAndGroupScannedExams(t, pdfPath, newMarkingIntegrationTempDir(t))
	exam := findExamByID(t, corpus.Exams, 781)
	if got, want := len(exam.Pages), 1; got != want {
		t.Fatalf("page count = %d, want %d", got, want)
	}

	conn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("open historical database: %v", err)
	}
	defer conn.Close()
	queries := db.New(conn)
	ctx := context.Background()

	examContent, err := queries.GetStudentContentExam(ctx, db.GetStudentContentExamParams{
		StudentExamID: 781,
		UserID:        1,
	})
	if err != nil {
		t.Fatalf("get student exam content: %v", err)
	}
	var qcm config.QCM
	if err := json.Unmarshal([]byte(examContent.Content), &qcm); err != nil {
		t.Fatalf("decode QCM: %v", err)
	}

	pageData, err := queries.GetPageContent(ctx, db.GetPageContentParams{
		StudentExamID: 781,
		Page:          int64(exam.Pages[0].Number),
		UserID:        1,
	})
	if err != nil {
		t.Fatalf("get page content: %v", err)
	}
	var pageContent config.PageContent
	if err := json.Unmarshal([]byte(pageData), &pageContent); err != nil {
		t.Fatalf("decode page content: %v", err)
	}
	if got, want := len(pageContent.Answers), 16; got != want {
		t.Fatalf("answer count = %d, want %d", got, want)
	}

	typstPath, ok := TypstWriter(corpus.TempDir, "integration-test", qcm, config.ExamQCM)
	if !ok {
		t.Fatal("reconstruct reference page with Typst")
	}
	referencePages, ok := ExportTypstToPNGs(typstPath)
	if !ok {
		t.Fatal("export Typst reference page to PNG")
	}
	if got, want := len(referencePages), 1; got != want {
		t.Fatalf("reference page count = %d, want %d", got, want)
	}

	homographyName := Homography(corpus.TempDir, exam.Pages[0].Name, filepath.Base(referencePages[0]))
	if homographyName == "" {
		t.Fatal("align scanned page with reference page")
	}
	got, err := GetAnswersState(corpus.TempDir, homographyName, pageContent.Answers)
	if err != nil {
		t.Fatalf("recognize answers: %v", err)
	}
	if len(got) != 16 {
		t.Fatalf("recognized answer count = %d, want 16", len(got))
	}

	want := []int{
		0, 0, 1, 0,
		0, 1, 0, 0,
		1, 0, 0, 0,
		1, 0, 0, 0,
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("case %d: got %d, want %d", i+1, got[i], want[i])
		}
	}
}
