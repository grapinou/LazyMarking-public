package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestQrPipelineWithScannedExams(t *testing.T) {
	pdfPath := os.Getenv("LAZYMARKING_TEST_PDF")
	if pdfPath == "" {
		t.Skip("LAZYMARKING_TEST_PDF is not set; skipping private scanned-exams integration test")
	}

	corpus := readAndGroupScannedExams(t, pdfPath, "")
	validateThreePageHistoricalCorpus(t, corpus)
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
			continue
		}

		qrData, err := QrReader(filepath.Join(tempDir, pngName))
		if err != nil {
			continue
		}

		var info config.QrCodeInfo
		if err := json.Unmarshal([]byte(qrData), &info); err != nil {
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

func validateThreePageHistoricalCorpus(t *testing.T, corpus scannedExamCorpus) {
	t.Helper()
	if got, want := corpus.PageCount, 69; got != want {
		t.Fatalf("split page count = %d, want %d", got, want)
	}
	if got, want := corpus.QRCount, 69; got != want {
		t.Fatalf("successfully read QR code count = %d, want %d", got, want)
	}

	exams := corpus.Exams
	if got, want := len(exams), 23; got != want {
		t.Fatalf("grouped student_exam count = %d, want %d", got, want)
	}

	for _, exam := range exams {
		if got, want := len(exam.Pages), 3; got != want {
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
