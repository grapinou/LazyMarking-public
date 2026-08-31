package tools

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
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

func TestHomographyAbsoluteHistoricalReferencePreservesDetectionsAndScores(t *testing.T) {
	tempDir := t.TempDir()
	legacyReference := filepath.Join(tempDir, "legacy-reference.png")
	writeFeatureRichHomographyPNG(t, legacyReference)
	bytes, err := os.ReadFile(legacyReference)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "scan.png"), bytes, 0o600); err != nil {
		t.Fatal(err)
	}
	durableDir := filepath.Join(t.TempDir(), "student-exam-42")
	if err := os.MkdirAll(durableDir, 0o750); err != nil {
		t.Fatal(err)
	}
	durableReference := filepath.Join(durableDir, "page-1.png")
	if err := os.WriteFile(durableReference, bytes, 0o600); err != nil {
		t.Fatal(err)
	}

	legacyHomo, err := Homography(tempDir, "scan.png", filepath.Base(legacyReference))
	if err != nil {
		t.Fatalf("legacy path homography: %v", err)
	}
	durableHomo, err := Homography(tempDir, "scan.png", durableReference)
	if err != nil {
		t.Fatalf("durable path homography: %v", err)
	}
	answers := []config.CircleValidated{{Position: config.Position{X: 105, Y: 125}, Radius: 20}}
	legacyDetections, err := GetAnswerDetections(tempDir, legacyHomo, answers)
	if err != nil {
		t.Fatal(err)
	}
	durableDetections, err := GetAnswerDetections(tempDir, durableHomo, answers)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(legacyDetections, durableDetections) {
		t.Fatalf("detections differ: legacy=%+v durable=%+v", legacyDetections, durableDetections)
	}
	qcm := config.QCM{Questions: []config.Question{{
		Tags:    config.Tags{Point: config.Point{PointValue: 2}},
		Answers: []config.Answer{{State: int64(legacyDetections[0].State)}},
	}}}
	legacyMarks := CountingPoints(qcm, answerDetectionStates(legacyDetections))
	durableMarks := CountingPoints(qcm, answerDetectionStates(durableDetections))
	if !reflect.DeepEqual(legacyMarks, durableMarks) {
		t.Fatalf("question marks differ: legacy=%+v durable=%+v", legacyMarks, durableMarks)
	}
	legacyScore, legacyTotal := CountingTotalPoint(legacyMarks)
	durableScore, durableTotal := CountingTotalPoint(durableMarks)
	if legacyScore != durableScore || legacyTotal != durableTotal {
		t.Fatalf("scores differ: legacy=%v/%d durable=%v/%d", legacyScore, legacyTotal, durableScore, durableTotal)
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

func writeFeatureRichHomographyPNG(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	img := image.NewRGBA(image.Rect(0, 0, 800, 1000))
	for y := 0; y < 1000; y++ {
		for x := 0; x < 800; x++ {
			value := uint8(255)
			if (x/17+y/23)%7 == 0 || (x > 90 && x < 120 && y > 110 && y < 140) {
				value = uint8((x*13 + y*7) % 120)
			}
			img.Set(x, y, color.RGBA{R: value, G: value, B: value, A: 255})
		}
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
}
