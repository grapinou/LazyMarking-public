package tools

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestHomographyReturnsErrorForMissingSource(t *testing.T) {
	tempDir := t.TempDir()
	writeUniformHomographyPNG(t, filepath.Join(tempDir, "reference.png"))

	name, err := Homography(tempDir, "missing.png", "reference.png")
	if err == nil {
		t.Fatal("Homography returned nil error for missing source")
	}
	if name != "" {
		t.Fatalf("Homography name = %q, want empty", name)
	}
}

func TestHomographyReturnsErrorForMissingReference(t *testing.T) {
	tempDir := t.TempDir()
	writeUniformHomographyPNG(t, filepath.Join(tempDir, "source.png"))

	name, err := Homography(tempDir, "source.png", "missing.png")
	if err == nil {
		t.Fatal("Homography returned nil error for missing reference")
	}
	if name != "" {
		t.Fatalf("Homography name = %q, want empty", name)
	}
}

func TestHomographyReturnsErrorForImagesWithoutFeatures(t *testing.T) {
	tempDir := t.TempDir()
	writeUniformHomographyPNG(t, filepath.Join(tempDir, "source.png"))
	writeUniformHomographyPNG(t, filepath.Join(tempDir, "reference.png"))

	name, err := Homography(tempDir, "source.png", "reference.png")
	if err == nil {
		t.Fatal("Homography returned nil error for featureless images")
	}
	if name != "" {
		t.Fatalf("Homography name = %q, want empty", name)
	}
}

func writeUniformHomographyPNG(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create PNG: %v", err)
	}
	defer file.Close()

	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.White)
		}
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
}
