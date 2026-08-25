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

	readAndGroupScannedExams(t, pdfPath, "")
}

func readAndGroupScannedExams(t *testing.T, pdfPath, tempDir string) ([]config.Exam, string) {
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
	if got, want := len(pages), 69; got != want {
		t.Fatalf("split page count = %d, want %d", got, want)
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

	if got, want := len(qrCodes), 69; got != want {
		t.Fatalf("successfully read QR code count = %d, want %d", got, want)
	}

	exams := GroupQrCodes(qrCodes)
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

	return exams, tempDir
}
