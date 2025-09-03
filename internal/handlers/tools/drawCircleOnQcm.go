package tools

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"gocv.io/x/gocv"
)

func DrawCircleOnQcm(tempDir, imgName, outputName string, questions []config.CircleValidated, answers [][]config.CircleValidated) {
	imgPath := filepath.Join(tempDir, imgName)
	// Charger image couleur pour tracer les cercles
	colorImg := gocv.IMRead(imgPath, gocv.IMReadColor)
	defer colorImg.Close()

	// pour tracer des cercles autours des cercles questions
	fmt.Println("entourage des cercles des questions")
	for i, question := range questions {
		redColor := uint8(100 + 50*i)
		gocv.Circle(&colorImg, image.Pt(question.Position.X, question.Position.Y), question.Radius+10, color.RGBA{redColor, 0, 0, 0}, 8)
	}

	// pour tracer des cercles autours des cercles reponses
	fmt.Println("entourage des cercles des réponses")
	for _, answerLine := range answers {
		for i, answer := range answerLine {
			greenColor := uint8(50 + 50*i)
			gocv.Circle(&colorImg, image.Pt(answer.Position.X, answer.Position.Y), answer.Radius+5, color.RGBA{0, greenColor, 0, 0}, 8)
		}
	}

	result := filepath.Join(tempDir, outputName+imgName)
	gocv.IMWrite(result, colorImg)
}
