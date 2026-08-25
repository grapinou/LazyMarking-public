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

func TestRealTwoPageAnswerRecognition14(t *testing.T) {
	pdfPath := os.Getenv("LAZYMARKING_TEST_PDF_2_PAGES")
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	if pdfPath == "" || dbPath == "" {
		t.Skip("LAZYMARKING_TEST_PDF_2_PAGES and LAZYMARKING_TEST_DB must both be set; skipping private multi-page answer-recognition integration test")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	t.Chdir(filepath.Clean(filepath.Join(workingDir, "../../..")))
	tempDir := newMarkingIntegrationTempDir(t)

	conn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("open historical database: %v", err)
	}
	defer conn.Close()
	queries := db.New(conn)
	ctx := context.Background()

	examContent, err := queries.GetStudentContentExam(ctx, db.GetStudentContentExamParams{
		StudentExamID: 14,
		UserID:        1,
	})
	if err != nil {
		t.Fatalf("get student exam content: %v", err)
	}
	if got, want := examContent.PageTot, int64(2); got != want {
		t.Fatalf("database page count = %d, want %d", got, want)
	}
	var qcm config.QCM
	if err := json.Unmarshal([]byte(examContent.Content), &qcm); err != nil {
		t.Fatalf("decode QCM: %v", err)
	}

	exam := findHistoricalTwoPageExam(t, pdfPath, tempDir, 14)
	if got, want := len(exam.Pages), 2; got != want {
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

	typstPath, ok := TypstWriter(tempDir, "integration-test", qcm, config.ExamQCM)
	if !ok {
		t.Fatal("reconstruct reference pages with Typst")
	}
	referencePages, ok := ExportTypstToPNGs(typstPath)
	if !ok {
		t.Fatal("export Typst reference pages to PNG")
	}
	if got, want := len(referencePages), 2; got != want {
		t.Fatalf("reference page count = %d, want %d", got, want)
	}

	wantPageLengths := []int{20, 4}
	got := make([]int, 0, 24)
	for i, page := range exam.Pages {
		pageData, err := queries.GetPageContent(ctx, db.GetPageContentParams{
			StudentExamID: 14,
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
	if len(got) != 24 {
		t.Fatalf("recognized answer count = %d, want 24", len(got))
	}

	want := []int{
		1, 1, 0, 1,
		1, 0, 0, 1,
		0, 1, 0, 1,
		1, 0, 0, 0,
		1, 0, 0, 1,
		0, 0, 1, 1,
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("case %d: got %d, want %d", i+1, got[i], want[i])
		}
	}

	questionMarks := CountingPoints(qcm, got)
	if got, want := len(questionMarks), 6; got != want {
		t.Fatalf("question mark count = %d, want %d", got, want)
	}
	wantQuestionMarks := []config.QuestionMark{
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 2},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Partial, Score: 0.5, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Partial, Score: 1, Total: 2},
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
	if score != 1.5 {
		t.Errorf("score = %v, want 1.5", score)
	}
	if total != 8 {
		t.Errorf("total = %d, want 8", total)
	}
}

func findHistoricalTwoPageExam(t *testing.T, pdfPath, tempDir string, studentExamID int64) config.Exam {
	t.Helper()

	pdf, err := os.Open(pdfPath)
	if err != nil {
		t.Fatalf("open reference PDF: %v", err)
	}
	defer pdf.Close()
	if err := SplitPdf(pdf, tempDir, "page-%d.pdf"); err != nil {
		t.Fatalf("split reference PDF: %v", err)
	}
	pagePaths, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		t.Fatalf("list split PDF pages: %v", err)
	}
	sort.Strings(pagePaths)

	targetPages := make(map[int]config.Page)
	for _, pagePath := range pagePaths {
		pngName, err := ConvertPdfToPng(tempDir, filepath.Base(pagePath), "")
		if err != nil {
			t.Fatalf("convert PDF page to PNG: %v", err)
		}
		qrData, err := QrReader(filepath.Join(tempDir, pngName))
		if err != nil {
			continue
		}
		var info config.QrCodeInfo
		if err := json.Unmarshal([]byte(qrData), &info); err != nil {
			continue
		}
		if info.StudentExamID != studentExamID {
			continue
		}
		targetPages[info.PageExam] = config.Page{Number: info.PageExam, Name: pngName}
	}
	exam := config.Exam{StudentExamID: studentExamID}
	for pageNumber := 1; pageNumber <= 2; pageNumber++ {
		page, ok := targetPages[pageNumber]
		if !ok {
			t.Fatalf("page %d was not found in scanned exam", pageNumber)
		}
		exam.Pages = append(exam.Pages, page)
	}
	return exam
}
