package tools

import (
	"fmt"
	"math"
)

// MarkingAmbiguityDelta is the first review-band value calibrated on a real
// marking corpus. It is a versioned product policy, not a universal property
// of MeanGray measurements.
const MarkingAmbiguityDelta = 5.0

// IsMarkingDetectionAmbiguous reports whether a finite MeanGray measurement is
// inside the inclusive review band snapshot for a marking job.
func IsMarkingDetectionAmbiguous(meanGray, threshold, ambiguityDelta float64) (bool, error) {
	for name, value := range map[string]float64{
		"mean gray":       meanGray,
		"threshold":       threshold,
		"ambiguity delta": ambiguityDelta,
	} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false, fmt.Errorf("%s must be finite", name)
		}
	}
	if meanGray < 0 || meanGray > 255 {
		return false, fmt.Errorf("mean gray must be within [0,255]")
	}
	if threshold < 0 || threshold > 255 {
		return false, fmt.Errorf("threshold must be within [0,255]")
	}
	if ambiguityDelta < 0 {
		return false, fmt.Errorf("ambiguity delta must be non-negative")
	}
	return math.Abs(meanGray-threshold) <= ambiguityDelta, nil
}
