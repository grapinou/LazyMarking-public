package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestQrPipelineWithScannedExams(t *testing.T) {
	testCases := []struct {
		name         string
		env          string
		pageCount    int
		examCount    int
		pagesPerExam int
	}{
		{name: "one_page", env: "LAZYMARKING_TEST_PDF_1_PAGE", pageCount: 27, examCount: 27, pagesPerExam: 1},
		{name: "two_pages", env: "LAZYMARKING_TEST_PDF_2_PAGES", pageCount: 52, examCount: 26, pagesPerExam: 2},
		{name: "three_pages", env: "LAZYMARKING_TEST_PDF", pageCount: 69, examCount: 23, pagesPerExam: 3},
	}
	for _, tc := range testCases {
		if os.Getenv(tc.env) == "" {
			t.Skip("private one-, two-, and three-page PDF corpora must all be configured; skipping historical QR pipeline integration test")
		}
	}

	var totalPages, totalQRCodes, totalExams int
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			corpus := readAndGroupScannedExams(t, os.Getenv(tc.env), "")
			validateHistoricalCorpus(t, corpus, tc.pageCount, tc.examCount, tc.pagesPerExam)
			totalPages += corpus.PageCount
			totalQRCodes += corpus.QRCount
			totalExams += len(corpus.Exams)
		})
	}

	if got, want := totalPages, 148; got != want {
		t.Errorf("total split page count = %d, want %d", got, want)
	}
	if got, want := totalQRCodes, 148; got != want {
		t.Errorf("total successfully read QR code count = %d, want %d", got, want)
	}
	if got, want := totalExams, 76; got != want {
		t.Errorf("total grouped student_exam count = %d, want %d", got, want)
	}
}

type scannedExamCorpus struct {
	Exams     []config.Exam
	TempDir   string
	PageCount int
	QRCount   int
}

func readAndGroupScannedExams(t *testing.T, pdfPath, tempDir string) scannedExamCorpus {
	t.Helper()

	pdf, err := os.Open(pdfPath)
	if err != nil {
		t.Fatalf("open reference PDF %q: %v", pdfPath, err)
	}
	defer pdf.Close()

	if tempDir == "" {
		tempDir = t.TempDir()
	}
	if err := SplitPdf(pdf, tempDir, "page-%d.pdf"); err != nil {
		t.Fatalf("split reference PDF: %v", err)
	}

	pages, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		t.Fatalf("list split PDF pages: %v", err)
	}
	qrCodes := make([]config.QrCodeInfo, 0, len(pages))
	for _, pagePath := range pages {
		pageName := filepath.Base(pagePath)
		pngName, err := ConvertPdfToPng(tempDir, pageName, "")
		if err != nil {
			t.Errorf("convert %s to PNG: %v", pageName, err)
			continue
		}

		qrData, err := QrReader(filepath.Join(tempDir, pngName))
		if err != nil {
			t.Errorf("read QR code from %s: %v", pngName, err)
			continue
		}

		var info config.QrCodeInfo
		if err := json.Unmarshal([]byte(qrData), &info); err != nil {
			t.Errorf("decode QR data from %s: %v", pngName, err)
			continue
		}
		info.PageName = pngName
		qrCodes = append(qrCodes, info)
	}

	exams := GroupQrCodes(qrCodes)
	return scannedExamCorpus{
		Exams:     exams,
		TempDir:   tempDir,
		PageCount: len(pages),
		QRCount:   len(qrCodes),
	}
}

func validateHistoricalCorpus(t *testing.T, corpus scannedExamCorpus, pageCount, examCount, pagesPerExam int) {
	t.Helper()
	if got, want := corpus.PageCount, pageCount; got != want {
		t.Fatalf("split page count = %d, want %d", got, want)
	}
	if got, want := corpus.QRCount, pageCount; got != want {
		t.Fatalf("successfully read QR code count = %d, want %d", got, want)
	}

	exams := corpus.Exams
	if got, want := len(exams), examCount; got != want {
		t.Fatalf("grouped student_exam count = %d, want %d", got, want)
	}

	for _, exam := range exams {
		if got, want := len(exam.Pages), pagesPerExam; got != want {
			t.Errorf("student_exam %d page count = %d, want %d", exam.StudentExamID, got, want)
			continue
		}
		for i, page := range exam.Pages {
			wantPageNumber := i + 1
			if page.Number != wantPageNumber {
				t.Errorf("student_exam %d page at index %d has number %d, want %d", exam.StudentExamID, i, page.Number, wantPageNumber)
			}
		}
	}

}
