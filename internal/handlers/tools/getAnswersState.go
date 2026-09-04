package tools

import (
	"fmt"
	"image"
	"math"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"gocv.io/x/gocv"
)

const MarkingDetectionThreshold = 150.0

const (
	V2ROIRadiusRatio       = 0.40
	V2DarkPixelThreshold   = 220.0
	V2DarkRatioThreshold   = 0.10
	V2ChromaPixelThreshold = 12.0
	V2ChromaRatioThreshold = 0.05
	HybridReviewReason     = "detector_disagreement"
)

func GetAnswerDetections(tempDir, homoName string, answers []config.CircleValidated) ([]config.AnswerDetection, error) {
	imgPath := filepath.Join(tempDir, homoName)

	gray := gocv.IMRead(imgPath, gocv.IMReadGrayScale)
	if gray.Empty() {
		return nil, fmt.Errorf("unable to load image %q", homoName)
	}
	defer gray.Close()
	color := gocv.IMRead(imgPath, gocv.IMReadColor)
	if color.Empty() {
		return nil, fmt.Errorf("unable to load color image %q", homoName)
	}
	defer color.Close()

	detections := make([]config.AnswerDetection, len(answers))
	bounds := image.Rect(0, 0, gray.Cols(), gray.Rows())

	for i, answer := range answers {
		rect := MarkingAnswerMeasurementRect(image.Pt(answer.Position.X, answer.Position.Y), answer.Radius, bounds)
		if rect.Empty() {
			return nil, fmt.Errorf("answer %d is outside image bounds", i)
		}

		roi := gray.Region(rect)
		mean := roi.Mean()
		roi.Close()
		avg := mean.Val1

		historical, err := answerDetectionFromMean(avg)
		if err != nil {
			return nil, fmt.Errorf("answer %d: %w", i, err)
		}
		v2, err := measureV2Detector(color, answer, bounds)
		if err != nil {
			return nil, fmt.Errorf("answer %d: %w", i, err)
		}
		detections[i] = applyHybridPolicy(historical, v2)
	}

	return detections, nil
}

type v2Detection struct {
	DarkRatio, ChromaRatio       float64
	GrayscaleSignal, ColorSignal bool
	State                        int
}

func measureV2Detector(img gocv.Mat, answer config.CircleValidated, bounds image.Rectangle) (v2Detection, error) {
	if answer.Radius <= 0 {
		return v2Detection{}, fmt.Errorf("invalid answer radius %d", answer.Radius)
	}
	radius := V2ROIRadiusRatio * float64(answer.Radius)
	extent := int(math.Ceil(radius))
	center := image.Pt(answer.Position.X, answer.Position.Y)
	if center.X-extent < bounds.Min.X || center.Y-extent < bounds.Min.Y || center.X+extent >= bounds.Max.X || center.Y+extent >= bounds.Max.Y {
		return v2Detection{}, fmt.Errorf("V2 ROI is outside image bounds")
	}
	var dark, chromatic, count int
	radiusSquared := radius * radius
	for dy := -extent; dy <= extent; dy++ {
		for dx := -extent; dx <= extent; dx++ {
			if float64(dx*dx+dy*dy) > radiusSquared {
				continue
			}
			pixel := img.GetVecbAt(center.Y+dy, center.X+dx)
			blue, green, red := float64(pixel[0]), float64(pixel[1]), float64(pixel[2])
			gray := 0.299*red + 0.587*green + 0.114*blue
			if gray < V2DarkPixelThreshold {
				dark++
			}
			if math.Max(red, math.Max(green, blue))-math.Min(red, math.Min(green, blue)) > V2ChromaPixelThreshold {
				chromatic++
			}
			count++
		}
	}
	if count == 0 {
		return v2Detection{}, fmt.Errorf("V2 ROI contains no pixels")
	}
	darkRatio, chromaRatio := float64(dark)/float64(count), float64(chromatic)/float64(count)
	graySignal, colorSignal := darkRatio >= V2DarkRatioThreshold, chromaRatio >= V2ChromaRatioThreshold
	state := 0
	if graySignal || colorSignal {
		state = 1
	}
	return v2Detection{darkRatio, chromaRatio, graySignal, colorSignal, state}, nil
}

func applyHybridPolicy(historical config.AnswerDetection, v2 v2Detection) config.AnswerDetection {
	review := historical.State != v2.State
	reason := ""
	if review {
		reason = HybridReviewReason
	}
	automaticState := historical.State
	return config.AnswerDetection{
		Hybrid: true, State: historical.State, MeanGray: historical.MeanGray, HistoricalState: historical.State,
		V2State: v2.State, DarkRatio: v2.DarkRatio, ChromaRatio: v2.ChromaRatio,
		GrayscaleSignal: v2.GrayscaleSignal, ColorSignal: v2.ColorSignal,
		AutomaticState: automaticState, HasAutomaticState: !review,
		RequiresReview: review, ReviewReason: reason,
	}
}

// GetAnswersState remains as a compatibility adapter for callers that only
// need the historical 0/1 vector. Measurement and classification still have a
// single source of truth in GetAnswerDetections.
func GetAnswersState(tempDir, homoName string, answers []config.CircleValidated) ([]int, error) {
	detections, err := GetAnswerDetections(tempDir, homoName, answers)
	if err != nil {
		return nil, err
	}
	return answerDetectionStates(detections), nil
}

func answerDetectionFromMean(meanGray float64) (config.AnswerDetection, error) {
	if math.IsNaN(meanGray) || math.IsInf(meanGray, 0) || meanGray < 0 || meanGray > 255 {
		return config.AnswerDetection{}, fmt.Errorf("invalid mean gray value %v", meanGray)
	}
	state := 0
	if meanGray < MarkingDetectionThreshold {
		state = 1
	}
	return config.AnswerDetection{State: state, MeanGray: meanGray, HistoricalState: state}, nil
}

func answerDetectionStates(detections []config.AnswerDetection) []int {
	states := make([]int, len(detections))
	for index, detection := range detections {
		states[index] = detection.State
	}
	return states
}
