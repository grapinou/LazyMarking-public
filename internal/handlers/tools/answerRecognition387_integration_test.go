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

func TestRealThreePageAnswerRecognition387(t *testing.T) {
	pdfPath := os.Getenv("LAZYMARKING_TEST_PDF")
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	if pdfPath == "" || dbPath == "" {
		t.Skip("LAZYMARKING_TEST_PDF and LAZYMARKING_TEST_DB must both be set; skipping private three-page answer-recognition integration test")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	t.Chdir(filepath.Clean(filepath.Join(workingDir, "../../..")))
	tempDir := newMarkingIntegrationTempDir(t)

	corpus := readAndGroupScannedExams(t, pdfPath, tempDir)
	exam := findExamByID(t, corpus.Exams, 387)
	if got, want := len(exam.Pages), 3; got != want {
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
		StudentExamID: 387,
		UserID:        1,
	})
	if err != nil {
		t.Fatalf("get student exam content: %v", err)
	}
	if got, want := examContent.PageTot, int64(3); got != want {
		t.Fatalf("database page count = %d, want %d", got, want)
	}
	var qcm config.QCM
	if err := json.Unmarshal([]byte(examContent.Content), &qcm); err != nil {
		t.Fatalf("decode QCM: %v", err)
	}

	typstPath, ok := TypstWriter(tempDir, "integration-test", qcm, config.ExamQCM)
	if !ok {
		t.Fatal("reconstruct reference pages with Typst")
	}
	referencePages, ok := ExportTypstToPNGs(typstPath)
	if !ok {
		t.Fatal("export Typst reference pages to PNG")
	}
	if got, want := len(referencePages), 3; got != want {
		t.Fatalf("reference page count = %d, want %d", got, want)
	}

	wantPageLengths := []int{18, 16, 12}
	got := make([]int, 0, 46)
	for i, page := range exam.Pages {
		pageData, err := queries.GetPageContent(ctx, db.GetPageContentParams{
			StudentExamID: 387,
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

		homographyName := Homography(tempDir, page.Name, filepath.Base(referencePages[i]))
		if homographyName == "" {
			t.Fatalf("align scanned page %d with reference page", page.Number)
		}
		pageStates, err := GetAnswersState(tempDir, homographyName, pageContent.Answers)
		if err != nil {
			t.Fatalf("recognize page %d answers: %v", page.Number, err)
		}
		if len(pageStates) != wantPageLengths[i] {
			t.Fatalf("page %d recognized answer count = %d, want %d", page.Number, len(pageStates), wantPageLengths[i])
		}
		got = append(got, pageStates...)
	}
	if len(got) != 46 {
		t.Fatalf("recognized answer count = %d, want 46", len(got))
	}

	want := []int{
		0, 0, 0, 1,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
		0, 1,

		0, 0, 0, 0,
		0, 1, 1, 0,
		0, 0, 1, 0,
		1, 0, 1, 0,

		0, 1, 0, 0,
		0, 0, 0, 1,
		0, 1, 0, 0,
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("case %d: got %d, want %d", i+1, got[i], want[i])
		}
	}

	questionMarks := CountingPoints(qcm, got)
	if got, want := len(questionMarks), 13; got != want {
		t.Fatalf("question mark count = %d, want %d", got, want)
	}
	wantQuestionMarks := []config.QuestionMark{
		{State: config.Partial, Score: 0.5, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Correct, Score: 1, Total: 1},
		{State: config.Partial, Score: 1, Total: 2},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Correct, Score: 1, Total: 1},
		{State: config.Partial, Score: 1, Total: 2},
		{State: config.Correct, Score: 1, Total: 1},
		{State: config.Correct, Score: 1, Total: 1},
	}
	for i := range wantQuestionMarks {
		if questionMarks[i].State != wantQuestionMarks[i].State ||
			questionMarks[i].Score != wantQuestionMarks[i].Score ||
			questionMarks[i].Total != wantQuestionMarks[i].Total {
			t.Errorf(
				"Q%d: got state=%d score=%v total=%d, want state=%d score=%v total=%d",
				i+1,
				questionMarks[i].State,
				questionMarks[i].Score,
				questionMarks[i].Total,
				wantQuestionMarks[i].State,
				wantQuestionMarks[i].Score,
				wantQuestionMarks[i].Total,
			)
		}
	}

	score, total := CountingTotalPoint(questionMarks)
	if score != 6.5 {
		t.Errorf("score = %v, want 6.5", score)
	}
	if total != 15 {
		t.Errorf("total = %d, want 15", total)
	}
}
