package tools

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"gocv.io/x/gocv"
)

type answerRenderMarks struct {
	expected           bool
	effective          bool
	incorrectSelection bool
}

func answerMarks(expected, effective int) answerRenderMarks {
	return answerRenderMarks{
		expected:           expected == 1,
		effective:          effective == 1,
		incorrectSelection: expected == 0 && effective == 1,
	}
}

func DrawMarking(tempDir, imgName string, questionsMark []config.QuestionMark, questionsPositions []config.CircleValidated, expectedAnswers, effectiveAnswers []int, answers []config.CircleValidated) {
	imgPath := filepath.Join(tempDir, imgName)
	// Charger image couleur pour tracer les cercles
	colorImg := gocv.IMRead(imgPath, gocv.IMReadColor)
	defer colorImg.Close()

	thickness := 5

	for i, question := range questionsPositions {
		switch questionsMark[i].State {
		case config.Correct:
			offsetX := 4 * question.Radius
			baseX := question.Position.X - offsetX
			baseY := question.Position.Y
			green := color.RGBA{0, 255, 0, 0}
			p1 := image.Pt(baseX, baseY)
			p2 := image.Pt(baseX+int(0.5*float64(question.Radius)), baseY+int(0.8*float64(question.Radius)))
			p3 := image.Pt(baseX+int(1.5*float64(question.Radius)), baseY-int(1.0*float64(question.Radius)))
			gocv.Line(&colorImg, p1, p2, green, thickness)
			gocv.Line(&colorImg, p2, p3, green, thickness)

			grey := color.RGBA{125, 125, 125, 0}
			position := image.Pt(baseX, baseY+2*question.Radius) // coin inférieur gauche du texte
			fontScale := 1.0
			thickness := 2
			txt := fmt.Sprintf("%.2f/%d", questionsMark[i].Score, questionsMark[i].Total)
			gocv.PutText(&colorImg, txt, position, gocv.FontHersheySimplex, fontScale, grey, thickness)

		case config.Incorrect:
			offsetX := 3 * question.Radius
			baseX := question.Position.X - offsetX
			baseY := question.Position.Y
			red := color.RGBA{255, 0, 0, 0}
			size := int(0.7 * float64(question.Radius))
			// Coin supérieur gauche → coin inférieur droit
			p1 := image.Pt(baseX-size, baseY-size)
			p2 := image.Pt(baseX+size, baseY+size)
			// Coin inférieur gauche → coin supérieur droit
			p3 := image.Pt(baseX-size, baseY+size)
			p4 := image.Pt(baseX+size, baseY-size)
			gocv.Line(&colorImg, p1, p2, red, thickness)
			gocv.Line(&colorImg, p3, p4, red, thickness)

			grey := color.RGBA{125, 125, 125, 0}
			position := image.Pt(baseX-1*question.Radius, baseY+2*question.Radius) // coin inférieur gauche du texte
			fontScale := 1.0
			thickness := 2
			txt := fmt.Sprintf("%.2f/%d", questionsMark[i].Score, questionsMark[i].Total)
			gocv.PutText(&colorImg, txt, position, gocv.FontHersheySimplex, fontScale, grey, thickness)

		case config.Partial:
			offsetX := 3 * question.Radius
			baseX := question.Position.X - offsetX
			baseY := question.Position.Y
			green := color.RGBA{0, 255, 0, 0}
			red := color.RGBA{255, 0, 0, 0}
			size := int(0.7 * float64(question.Radius))
			// Coin supérieur gauche → coin inférieur droit
			p1 := image.Pt(baseX-size, baseY-size)
			p2 := image.Pt(baseX+size, baseY+size)
			// Coin inférieur gauche → coin supérieur droit
			p3 := image.Pt(baseX-size, baseY+size)
			p4 := image.Pt(baseX+size, baseY-size)
			gocv.Line(&colorImg, p1, p2, green, thickness)
			gocv.Line(&colorImg, p3, p4, red, thickness)

			grey := color.RGBA{125, 125, 125, 0}
			position := image.Pt(baseX-1*question.Radius, baseY+2*question.Radius) // coin inférieur gauche du texte
			fontScale := 1.0
			thickness := 2
			txt := fmt.Sprintf("%.2f/%d", questionsMark[i].Score, questionsMark[i].Total)
			gocv.PutText(&colorImg, txt, position, gocv.FontHersheySimplex, fontScale, grey, thickness)

		}
	}

	for i, answer := range answers {
		marks := answerMarks(expectedAnswers[i], effectiveAnswers[i])
		center := image.Pt(answer.Position.X, answer.Position.Y)
		if marks.expected {
			green := color.RGBA{0, 255, 0, 0}
			gocv.Circle(&colorImg, center, answer.Radius+10, green, 7)
		}
		if marks.effective {
			blue := color.RGBA{0, 100, 255, 0}
			innerRadius := max(3, answer.Radius-5)
			gocv.Circle(&colorImg, center, innerRadius, blue, 5)
		}
		if marks.incorrectSelection {
			baseX := answer.Position.X
			baseY := answer.Position.Y
			red := color.RGBA{255, 0, 0, 0}
			size := answer.Radius + 5

			// Coin supérieur gauche → coin inférieur droit
			p1 := image.Pt(baseX-size, baseY-size)
			p2 := image.Pt(baseX+size, baseY+size)

			// Coin inférieur gauche → coin supérieur droit
			p3 := image.Pt(baseX-size, baseY+size)
			p4 := image.Pt(baseX+size, baseY-size)

			gocv.Line(&colorImg, p1, p2, red, thickness)
			gocv.Line(&colorImg, p3, p4, red, thickness)
		}
	}
	result := filepath.Join(tempDir, imgName)
	gocv.IMWrite(result, colorImg)
}
