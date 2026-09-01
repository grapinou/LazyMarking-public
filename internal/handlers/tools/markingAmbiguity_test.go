package tools

import (
	"math"
	"testing"
)

func TestIsMarkingDetectionAmbiguousInclusiveBoundaries(t *testing.T) {
	for _, tc := range []struct {
		mean float64
		want bool
	}{
		{mean: 144.99, want: false},
		{mean: 145.00, want: true},
		{mean: 149.99, want: true},
		{mean: 150.00, want: true},
		{mean: 154.01, want: true},
		{mean: 155.00, want: true},
		{mean: 155.01, want: false},
	} {
		got, err := IsMarkingDetectionAmbiguous(tc.mean, MarkingDetectionThreshold, MarkingAmbiguityDelta)
		if err != nil || got != tc.want {
			t.Fatalf("mean=%v ambiguous=%v err=%v, want %v", tc.mean, got, err, tc.want)
		}
	}
	detection, err := answerDetectionFromMean(154.01)
	if err != nil || detection.State != 0 {
		t.Fatalf("historical detection=%+v err=%v, want unchecked", detection, err)
	}
	ambiguous, err := IsMarkingDetectionAmbiguous(detection.MeanGray, MarkingDetectionThreshold, MarkingAmbiguityDelta)
	if err != nil || !ambiguous || detection.State != 0 {
		t.Fatalf("ambiguous=%v detection=%+v err=%v", ambiguous, detection, err)
	}
}

func TestIsMarkingDetectionAmbiguousRejectsInvalidValues(t *testing.T) {
	for name, values := range map[string][3]float64{
		"mean NaN":       {math.NaN(), 150, 5},
		"mean low":       {-0.01, 150, 5},
		"mean high":      {255.01, 150, 5},
		"threshold NaN":  {150, math.NaN(), 5},
		"threshold low":  {150, -0.01, 5},
		"threshold high": {150, 255.01, 5},
		"delta NaN":      {150, 150, math.NaN()},
		"delta negative": {150, 150, -0.01},
		"delta infinite": {150, 150, math.Inf(1)},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := IsMarkingDetectionAmbiguous(values[0], values[1], values[2]); err == nil {
				t.Fatal("invalid ambiguity input accepted")
			}
		})
	}
}
