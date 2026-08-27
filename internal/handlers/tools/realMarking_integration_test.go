package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestRealStudentExamMarkingSmoke(t *testing.T) {
	testCases := []struct {
		name         string
		pdfEnv       string
		studentIDEnv string
		pageCount    int
	}{
		{name: "one_page", pdfEnv: "LAZYMARKING_TEST_PDF_1_PAGE", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_1_PAGE", pageCount: 1},
		{name: "two_pages", pdfEnv: "LAZYMARKING_TEST_PDF_2_PAGES", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_2_PAGES", pageCount: 2},
		{name: "three_pages", pdfEnv: "LAZYMARKING_TEST_PDF", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_3_PAGES", pageCount: 3},
	}
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	userIDValue := os.Getenv("LAZYMARKING_TEST_USER_ID")
	for _, tc := range testCases {
		if dbPath == "" || userIDValue == "" || os.Getenv(tc.pdfEnv) == "" || os.Getenv(tc.studentIDEnv) == "" {
			t.Skip("private one-, two-, and three-page PDFs, IDs, and historical DB must all be configured; skipping real-marking smoke test")
		}
	}
	userID := parsePositiveFixtureID(t, "LAZYMARKING_TEST_USER_ID")

	queries, closeDB := openCopiedHistoricalDB(t, dbPath)
	t.Cleanup(closeDB)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			studentExamID := parsePositiveFixtureID(t, tc.studentIDEnv)
			corpus := readAndGroupScannedExams(t, os.Getenv(tc.pdfEnv), newMarkingIntegrationTempDir(t))
			targetExam := findExamByID(t, corpus.Exams, studentExamID)
			if got := len(targetExam.Pages); got != tc.pageCount {
				t.Fatalf("student_exam_id %d page count = %d, want %d", studentExamID, got, tc.pageCount)
			}

			markExam, err := markStudentExamWithoutPanic(userID, corpus.TempDir, targetExam, queries)
			if err != nil {
				t.Fatalf("marking stage for student_exam_id %d: %v", studentExamID, err)
			}
			validateSuccessfulMark(t, markExam)
			if got := markExam.Pages; got != tc.pageCount {
				t.Errorf("MarkExam.Pages = %d, want %d", got, tc.pageCount)
			}
		})
	}
}

func TestRealCompleteMarkingBatches(t *testing.T) {
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	batchValue := os.Getenv("LAZYMARKING_TEST_BATCH_PDFS")
	if dbPath == "" || batchValue == "" || os.Getenv("LAZYMARKING_TEST_USER_ID") == "" {
		t.Skip("LAZYMARKING_TEST_DB, LAZYMARKING_TEST_USER_ID, and LAZYMARKING_TEST_BATCH_PDFS are not configured; skipping complete real-marking batches")
	}
	userID := parsePositiveFixtureID(t, "LAZYMARKING_TEST_USER_ID")

	batchPaths := splitConfiguredBatchPaths(batchValue)
	if len(batchPaths) == 0 {
		t.Fatal("LAZYMARKING_TEST_BATCH_PDFS contains no paths")
	}
	queries, closeDB := openCopiedHistoricalDB(t, dbPath)
	t.Cleanup(closeDB)

	for i, batchPath := range batchPaths {
		batchName := fmt.Sprintf("batch_%03d", i+1)
		t.Run(batchName, func(t *testing.T) {
			corpus, err := readAndGroupCompleteBatch(batchPath, newMarkingIntegrationTempDir(t))
			if err != nil {
				t.Fatalf("%s: %v", batchName, err)
			}
			if len(corpus.Exams) == 0 {
				t.Fatalf("%s QR grouping stage: no exams found", batchName)
			}

			seen := make(map[int64]struct{}, len(corpus.Exams))
			for _, exam := range corpus.Exams {
				if exam.StudentExamID <= 0 {
					t.Errorf("%s QR grouping stage: invalid student_exam_id %d", batchName, exam.StudentExamID)
					continue
				}
				if _, exists := seen[exam.StudentExamID]; exists {
					t.Errorf("%s QR grouping stage: duplicate student_exam_id %d", batchName, exam.StudentExamID)
					continue
				}
				seen[exam.StudentExamID] = struct{}{}
				if len(exam.Pages) == 0 {
					t.Errorf("%s QR grouping stage: student_exam_id %d has no pages", batchName, exam.StudentExamID)
					continue
				}

				markExam, err := markStudentExamWithoutPanic(userID, corpus.TempDir, exam, queries)
				if err != nil {
					t.Errorf("%s marking stage: student_exam_id %d: %v", batchName, exam.StudentExamID, err)
					continue
				}
				validateSuccessfulMark(t, markExam)
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

func parsePositiveFixtureID(t *testing.T, envName string) int64 {
	t.Helper()
	id, err := strconv.ParseInt(os.Getenv(envName), 10, 64)
	if err != nil || id <= 0 {
		t.Fatalf("%s must contain a positive student_exam_id", envName)
	}
	return id
}

func splitConfiguredBatchPaths(value string) []string {
	parts := filepath.SplitList(value)
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		if path := strings.TrimSpace(part); path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func openCopiedHistoricalDB(t *testing.T, sourcePath string) (*db.Queries, func()) {
	t.Helper()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	t.Chdir(filepath.Clean(filepath.Join(workingDir, "../../..")))

	source, err := os.Open(sourcePath)
	if err != nil {
		t.Fatal("historical DB copy stage: source fixture is not readable")
	}
	defer source.Close()

	copyDir := newMarkingIntegrationTempDir(t)
	copyPath := filepath.Join(copyDir, "historical-copy.db")
	destination, err := os.OpenFile(copyPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("historical DB copy stage: create temporary copy: %v", err)
	}
	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		t.Fatalf("historical DB copy stage: copy fixture: %v", err)
	}
	if err := destination.Close(); err != nil {
		t.Fatalf("historical DB copy stage: close temporary copy: %v", err)
	}

	conn, err := db.InitDB(copyPath)
	if err != nil {
		t.Fatalf("historical DB copy stage: initialize temporary copy: %v", err)
	}
	return db.New(conn), func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close temporary historical DB copy: %v", err)
		}
	}
}

func readAndGroupCompleteBatch(pdfPath, tempDir string) (scannedExamCorpus, error) {
	pdf, err := os.Open(pdfPath)
	if err != nil {
		return scannedExamCorpus{}, fmt.Errorf("PDF read stage: configured batch is not readable")
	}
	defer pdf.Close()

	if err := SplitPdf(pdf, tempDir, "page-%d.pdf"); err != nil {
		return scannedExamCorpus{}, fmt.Errorf("PDF read stage: split configured batch: %w", err)
	}
	pages, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		return scannedExamCorpus{}, fmt.Errorf("PDF read stage: list split pages: %w", err)
	}
	if len(pages) == 0 {
		return scannedExamCorpus{}, fmt.Errorf("PDF read stage: split produced no pages")
	}

	qrCodes := make([]config.QrCodeInfo, 0, len(pages))
	for pageIndex, pagePath := range pages {
		pageName := filepath.Base(pagePath)
		pngName, err := ConvertPdfToPng(tempDir, pageName, "")
		if err != nil {
			return scannedExamCorpus{}, fmt.Errorf("QR grouping stage: convert page %d: %w", pageIndex+1, err)
		}
		qrData, err := QrReader(filepath.Join(tempDir, pngName))
		if err != nil {
			return scannedExamCorpus{}, fmt.Errorf("QR grouping stage: read page %d QR code: %w", pageIndex+1, err)
		}
		var info config.QrCodeInfo
		if err := json.Unmarshal([]byte(qrData), &info); err != nil {
			return scannedExamCorpus{}, fmt.Errorf("QR grouping stage: decode page %d QR data: %w", pageIndex+1, err)
		}
		info.PageName = pngName
		qrCodes = append(qrCodes, info)
	}

	return scannedExamCorpus{
		Exams:     GroupQrCodes(qrCodes),
		TempDir:   tempDir,
		PageCount: len(pages),
		QRCount:   len(qrCodes),
	}, nil
}

func markStudentExamWithoutPanic(userID int64, tempDir string, exam config.Exam, queries *db.Queries) (markExam config.MarkExam, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return MarkingStudentExam(userID, "integration-test", tempDir, exam, context.Background(), queries)
}

func validateSuccessfulMark(t *testing.T, markExam config.MarkExam) {
	t.Helper()
	if !markExam.Status {
		t.Errorf("marking stage: student_exam_id %d: Status = false, want true", markExam.StudentExamID)
	}
	if markExam.Pages <= 0 {
		t.Errorf("marking stage: student_exam_id %d: Pages = %d, want > 0", markExam.StudentExamID, markExam.Pages)
	}
	if markExam.Total <= 0 {
		t.Errorf("marking stage: student_exam_id %d: Total = %d, want > 0", markExam.StudentExamID, markExam.Total)
	}
	if markExam.Score < 0 {
		t.Errorf("marking stage: student_exam_id %d: Score = %v, want >= 0", markExam.StudentExamID, markExam.Score)
	}
	if markExam.Score > float64(markExam.Total) {
		t.Errorf("marking stage: student_exam_id %d: Score = %v, want <= Total (%d)", markExam.StudentExamID, markExam.Score, markExam.Total)
	}
}
