package tools

import (
	"fmt"
	"image"
	"math"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"gocv.io/x/gocv"
)

const markingDetectionThreshold = 150.0

func GetAnswerDetections(tempDir, homoName string, answers []config.CircleValidated) ([]config.AnswerDetection, error) {
	imgPath := filepath.Join(tempDir, homoName)

	// Charger l'image en niveaux de gris
	img := gocv.IMRead(imgPath, gocv.IMReadGrayScale)
	if img.Empty() {
		return nil, fmt.Errorf("unable to load image %q", homoName)
	}
	defer img.Close()

	// fmt.Println("ROI pour :", homoName)

	detections := make([]config.AnswerDetection, len(answers))

	for i, answer := range answers {
		// Définir la ROI (x, y, largeur, hauteur)
		x1 := answer.Position.X - int(answer.Radius/2)
		y1 := answer.Position.Y - int(answer.Radius/2)
		x2 := answer.Position.X + int(answer.Radius/2)
		y2 := answer.Position.Y + int(answer.Radius/2)
		rect := image.Rect(x1, y1, x2, y2)
		rect = rect.Intersect(image.Rect(0, 0, img.Cols(), img.Rows()))
		if rect.Empty() {
			return nil, fmt.Errorf("answer %d is outside image bounds", i)
		}

		roi := img.Region(rect)
		mean := roi.Mean()
		roi.Close()      // on ferme explicitement au lieu d'utiliser defer
		avg := mean.Val1 // pour N&B -> valeur moyenne du canal 0

		// fmt.Printf("Moyenne ROI #%d = %.2f\n", i, avg)

		detection, err := answerDetectionFromMean(avg)
		if err != nil {
			return nil, fmt.Errorf("answer %d: %w", i, err)
		}
		detections[i] = detection

		/*
			// Visualisation
			greenColor := uint8(50 + 50*i)
			gocv.Circle(&img, image.Pt(answer.Position.X, answer.Position.Y),
				answer.Radius+5, color.RGBA{0, greenColor, 0, 0}, 2)

			rectColor := color.RGBA{G: 255, A: 255}
			gocv.Rectangle(&img, rect, rectColor, 2)
		*/
	}

	/*
		// Sauvegarde l’image avec rectangle
		name := "roi_" + homoName
		namePath := filepath.Join(tempDir, name)
		gocv.IMWrite(namePath, img)
	*/

	return detections, nil
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
	if meanGray < markingDetectionThreshold {
		state = 1
	}
	return config.AnswerDetection{State: state, MeanGray: meanGray}, nil
}

func answerDetectionStates(detections []config.AnswerDetection) []int {
	states := make([]int, len(detections))
	for index, detection := range detections {
		states[index] = detection.State
	}
	return states
}
