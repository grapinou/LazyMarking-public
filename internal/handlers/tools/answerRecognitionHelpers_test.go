package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func testRealMultiPageAnswerRecognition(
	t *testing.T,
	pdfEnv string,
	studentExamID int64,
	wantPageLengths []int,
	want []int,
) (config.QCM, []int) {
	t.Helper()

	pdfPath := os.Getenv(pdfEnv)
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	if pdfPath == "" || dbPath == "" {
		t.Skipf("%s and LAZYMARKING_TEST_DB must both be set; skipping private multi-page answer-recognition integration test", pdfEnv)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	t.Chdir(filepath.Clean(filepath.Join(workingDir, "../../..")))

	corpus := readAndGroupScannedExams(t, pdfPath, newMarkingIntegrationTempDir(t))
	exam := findExamByID(t, corpus.Exams, studentExamID)
	if got, want := len(exam.Pages), len(wantPageLengths); got != want {
		t.Fatalf("scan page count = %d, want %d", got, want)
	}
	sort.Slice(exam.Pages, func(i, j int) bool {
		return exam.Pages[i].Number < exam.Pages[j].Number
	})
	for i, page := range exam.Pages {
		if got, want := page.Number, i+1; got != want {
			t.Fatalf("page at index %d has number %d, want %d", i, got, want)
		}
	}

	conn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("open historical database: %v", err)
	}
	defer conn.Close()
	queries := db.New(conn)
	ctx := context.Background()

	examContent, err := queries.GetStudentContentExam(ctx, db.GetStudentContentExamParams{
		StudentExamID: studentExamID,
		UserID:        1,
	})
	if err != nil {
		t.Fatalf("get student exam content: %v", err)
	}
	if got, want := examContent.PageTot, int64(len(wantPageLengths)); got != want {
		t.Fatalf("database page count = %d, want %d", got, want)
	}
	var qcm config.QCM
	if err := json.Unmarshal([]byte(examContent.Content), &qcm); err != nil {
		t.Fatalf("decode QCM: %v", err)
	}

	typstPath, ok := TypstWriter(corpus.TempDir, "integration-test", qcm, config.ExamQCM)
	if !ok {
		t.Fatal("reconstruct reference pages with Typst")
	}
	referencePages, ok := ExportTypstToPNGs(typstPath)
	if !ok {
		t.Fatal("export Typst reference pages to PNG")
	}
	if got, want := len(referencePages), len(wantPageLengths); got != want {
		t.Fatalf("reference page count = %d, want %d", got, want)
	}

	got := make([]int, 0, len(want))
	for i, page := range exam.Pages {
		pageData, err := queries.GetPageContent(ctx, db.GetPageContentParams{
			StudentExamID: studentExamID,
			Page:          int64(page.Number),
			UserID:        1,
		})
		if err != nil {
			t.Fatalf("get page %d content: %v", page.Number, err)
		}
		var pageContent config.PageContent
		if err := json.Unmarshal([]byte(pageData), &pageContent); err != nil {
			t.Fatalf("decode page %d content: %v", page.Number, err)
		}

		homographyName, err := Homography(corpus.TempDir, page.Name, filepath.Base(referencePages[i]))
		if err != nil {
			t.Fatalf("align scanned page %d with reference page: %v", page.Number, err)
		}
		pageStates, err := GetAnswersState(corpus.TempDir, homographyName, pageContent.Answers)
		if err != nil {
			t.Fatalf("recognize page %d answers: %v", page.Number, err)
		}
		if got, want := len(pageStates), wantPageLengths[i]; got != want {
			t.Fatalf("page %d recognized answer count = %d, want %d", page.Number, got, want)
		}
		got = append(got, pageStates...)
	}
	if len(got) != len(want) {
		t.Fatalf("recognized answer count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("case %d: got %d, want %d", i+1, got[i], want[i])
		}
	}

	return qcm, got
}

func testRealOnePageAnswerRecognition(t *testing.T, studentExamID int64, want []int) (config.QCM, []int) {
	t.Helper()

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
	exam := findExamByID(t, corpus.Exams, studentExamID)
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
		StudentExamID: studentExamID,
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
		StudentExamID: studentExamID,
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
	if got := len(pageContent.Answers); got != len(want) {
		t.Fatalf("answer count = %d, want %d", got, len(want))
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

	homographyName, err := Homography(corpus.TempDir, exam.Pages[0].Name, filepath.Base(referencePages[0]))
	if err != nil {
		t.Fatalf("align scanned page with reference page: %v", err)
	}
	got, err := GetAnswersState(corpus.TempDir, homographyName, pageContent.Answers)
	if err != nil {
		t.Fatalf("recognize answers: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("recognized answer count = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("case %d: got %d, want %d", i+1, got[i], want[i])
		}
	}

	return qcm, got
}
