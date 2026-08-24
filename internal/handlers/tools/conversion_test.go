package tools

import "testing"

func TestConvertPdfToPngReturnsError(t *testing.T) {
	if _, err := ConvertPdfToPng(t.TempDir(), "missing.pdf", ""); err == nil {
		t.Fatal("expected conversion of a missing PDF to fail")
	}
}

func TestConvertPngToPdfReturnsError(t *testing.T) {
	if _, err := ConvertPngTopdf(t.TempDir(), "missing.png"); err == nil {
		t.Fatal("expected conversion of a missing PNG to fail")
	}
}

func TestGetAnswersStateReturnsErrorForMissingImage(t *testing.T) {
	if _, err := GetAnswersState(t.TempDir(), "missing.png", nil); err == nil {
		t.Fatal("expected a missing image to return an error")
	}
}
