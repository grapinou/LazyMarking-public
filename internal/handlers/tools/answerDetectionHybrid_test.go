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
		historical, v2 int
		review         bool
	}{
		{0, 0, false}, {1, 1, false}, {0, 1, true}, {1, 0, true},
	} {
		got := applyHybridPolicy(config.AnswerDetection{State: tc.historical, MeanGray: 100}, v2Detection{State: tc.v2})
		if got.State != tc.historical || got.RequiresReview != tc.review {
			t.Fatalf("historical=%d v2=%d: %+v", tc.historical, tc.v2, got)
		}
		if tc.review && got.ReviewReason != HybridReviewReason {
			t.Fatalf("missing review reason: %+v", got)
		}
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
