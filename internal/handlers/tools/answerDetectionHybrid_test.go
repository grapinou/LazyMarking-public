package tools

import (
	"image"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"gocv.io/x/gocv"
)

func TestFrozenV2DetectorSignalsAndCircularROI(t *testing.T) {
	answer := config.CircleValidated{Position: config.Position{X: 12, Y: 12}, Radius: 10}
	bounds := image.Rect(0, 0, 25, 25)
	circle := circleOffsets(4)

	tests := []struct {
		name                           string
		pixels                         map[image.Point][3]uint8
		wantDark, wantColor            bool
		wantState                      int
		wantDarkCount, wantChromaCount int
	}{
		{name: "white", pixels: map[image.Point][3]uint8{}, wantState: 0},
		{name: "dark exact ratio", pixels: paintOffsets(circle[:5], [3]uint8{0, 0, 0}), wantDark: true, wantState: 1, wantDarkCount: 5},
		{name: "dark below ratio", pixels: paintOffsets(circle[:4], [3]uint8{0, 0, 0}), wantState: 0, wantDarkCount: 4},
		{name: "colored exact ratio", pixels: paintOffsets(circle[:3], [3]uint8{255, 240, 255}), wantColor: true, wantState: 1, wantChromaCount: 3},
		{name: "blue mark", pixels: paintOffsets(circle[:3], [3]uint8{255, 240, 220}), wantColor: true, wantState: 1, wantChromaCount: 3},
		{name: "green mark", pixels: paintOffsets(circle[:3], [3]uint8{240, 255, 240}), wantColor: true, wantState: 1, wantChromaCount: 3},
		{name: "pixel thresholds are strict", pixels: map[image.Point][3]uint8{circle[0]: {220, 220, 220}, circle[1]: {255, 243, 255}}, wantState: 0},
		{name: "outside circular ROI ignored", pixels: map[image.Point][3]uint8{{X: 4, Y: 4}: {0, 0, 0}}, wantState: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mat := colorMat(t, 25, 25, tc.pixels)
			defer mat.Close()
			got, err := measureV2Detector(mat, answer, bounds)
			if err != nil {
				t.Fatal(err)
			}
			if got.GrayscaleSignal != tc.wantDark || got.ColorSignal != tc.wantColor || got.State != tc.wantState {
				t.Fatalf("detection=%+v", got)
			}
			if got.DarkRatio != float64(tc.wantDarkCount)/49 || got.ChromaRatio != float64(tc.wantChromaCount)/49 {
				t.Fatalf("ratios=(%v,%v)", got.DarkRatio, got.ChromaRatio)
			}
		})
	}
}

func TestHybridPolicyFourCombinations(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		policy                       string
		historical, v2               int
		color                        bool
		review, automatic            bool
		wantAutomatic, wantEffective int
	}{
		{name: "C negative agreement", policy: MarkingReviewPolicyColorConfidence, historical: 0, v2: 0, automatic: true},
		{name: "C positive agreement without color", policy: MarkingReviewPolicyColorConfidence, historical: 1, v2: 1, automatic: true, wantAutomatic: 1, wantEffective: 1},
		{name: "C positive agreement with color", policy: MarkingReviewPolicyColorConfidence, historical: 1, v2: 1, color: true, automatic: true, wantAutomatic: 1, wantEffective: 1},
		{name: "C color confidence", policy: MarkingReviewPolicyColorConfidence, historical: 0, v2: 1, color: true, automatic: true, wantAutomatic: 1, wantEffective: 1},
		{name: "C grayscale disagreement", policy: MarkingReviewPolicyColorConfidence, historical: 0, v2: 1, review: true},
		{name: "C reverse disagreement", policy: MarkingReviewPolicyColorConfidence, historical: 1, v2: 0, review: true, wantEffective: 1},
		{name: "agreement v1 keeps colored disagreement", policy: MarkingReviewPolicyAgreement, historical: 0, v2: 1, color: true, review: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v2 := v2Detection{State: tc.v2, ColorSignal: tc.color, GrayscaleSignal: tc.v2 == 1 && !tc.color}
			got, err := applyHybridPolicyVersion(tc.policy, config.AnswerDetection{State: tc.historical, MeanGray: 100}, v2)
			if err != nil {
				t.Fatal(err)
			}
			if got.State != tc.historical || got.RequiresReview != tc.review || got.HasAutomaticState != tc.automatic || (tc.automatic && got.AutomaticState != tc.wantAutomatic) {
				t.Fatalf("detection=%+v", got)
			}
			if tc.review && got.ReviewReason != HybridReviewReason {
				t.Fatalf("missing review reason: %+v", got)
			}
			if effective := answerDetectionScoringStates([]config.AnswerDetection{got})[0]; effective != tc.wantEffective {
				t.Fatalf("scoring state=%d, want %d", effective, tc.wantEffective)
			}
			if err := validateHybridDetectionForPolicy(tc.policy, got); err != nil {
				t.Fatalf("validate policy result: %v", err)
			}
		})
	}

	if _, err := applyHybridPolicyVersion("unknown", config.AnswerDetection{}, v2Detection{}); err == nil {
		t.Fatal("unknown policy succeeded")
	}
}

func colorMat(t *testing.T, rows, cols int, pixels map[image.Point][3]uint8) gocv.Mat {
	t.Helper()
	data := make([]byte, rows*cols*3)
	for i := range data {
		data[i] = 255
	}
	for offset, value := range pixels {
		row, col := 12+offset.Y, 12+offset.X
		for channel := range 3 {
			data[(row*cols+col)*3+channel] = value[channel]
		}
	}
	mat, err := gocv.NewMatFromBytes(rows, cols, gocv.MatTypeCV8UC3, data)
	if err != nil {
		t.Fatal(err)
	}
	return mat
}

func circleOffsets(radius int) []image.Point {
	var result []image.Point
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			if x*x+y*y <= radius*radius {
				result = append(result, image.Pt(x, y))
			}
		}
	}
	return result
}

func paintOffsets(offsets []image.Point, value [3]uint8) map[image.Point][3]uint8 {
	result := make(map[image.Point][3]uint8, len(offsets))
	for _, offset := range offsets {
		result[offset] = value
	}
	return result
}
