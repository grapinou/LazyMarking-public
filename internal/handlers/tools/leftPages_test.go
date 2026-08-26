package tools

import (
	"database/sql"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestLeftPagesWithoutPNGsOrExistingResult(t *testing.T) {
	tempDir := t.TempDir()

	got, err := LeftPages(tempDir, leftPagesTestName())
	if err != nil {
		t.Fatalf("LeftPages: %v", err)
	}
	if got != "" {
		t.Fatalf("LeftPages returned %q, want empty name", got)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "corrected_NOT.pdf")); !os.IsNotExist(err) {
		t.Fatalf("unexpected left-pages PDF: %v", err)
	}
}

func TestLeftPagesReturnsExistingResultWithoutPNGs(t *testing.T) {
	tempDir := t.TempDir()
	want := "corrected_NOT.pdf"
	if err := os.WriteFile(filepath.Join(tempDir, want), []byte("existing"), 0o600); err != nil {
		t.Fatalf("create existing result: %v", err)
	}

	got, err := LeftPages(tempDir, leftPagesTestName())
	if err != nil {
		t.Fatalf("LeftPages: %v", err)
	}
	if got != want {
		t.Fatalf("LeftPages returned %q, want %q", got, want)
	}
}

func TestLeftPagesCreatesResultAndReturnsItAgain(t *testing.T) {
	tempDir := t.TempDir()
	pngPath := filepath.Join(tempDir, "page-1.png")
	writeLeftPagesTestPNG(t, pngPath)

	want := "corrected_NOT.pdf"
	got, err := LeftPages(tempDir, leftPagesTestName())
	if err != nil {
		t.Fatalf("first LeftPages call: %v", err)
	}
	if got != want {
		t.Fatalf("first LeftPages call returned %q, want %q", got, want)
	}
	info, err := os.Stat(filepath.Join(tempDir, want))
	if err != nil {
		t.Fatalf("stat merged result: %v", err)
	}
	if !info.Mode().IsRegular() {
		t.Fatal("merged result is not a regular file")
	}
	if _, err := os.Stat(pngPath); !os.IsNotExist(err) {
		t.Fatalf("source PNG still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "page-1.pdf")); !os.IsNotExist(err) {
		t.Fatalf("intermediate PDF still exists: %v", err)
	}

	got, err = LeftPages(tempDir, leftPagesTestName())
	if err != nil {
		t.Fatalf("second LeftPages call: %v", err)
	}
	if got != want {
		t.Fatalf("second LeftPages call returned %q, want %q", got, want)
	}
}

func leftPagesTestName() db.GetExamAndMarkNameRow {
	return db.GetExamAndMarkNameRow{
		ExamName: sql.NullString{String: filepath.Join("ignored", "corrected.pdf"), Valid: true},
	}
}

func writeLeftPagesTestPNG(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create PNG: %v", err)
	}
	defer file.Close()

	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.White)
		}
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
}
