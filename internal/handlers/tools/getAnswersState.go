package tools

import (
	"image"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"gocv.io/x/gocv"
)

func GetAnswersState(tempDir, homoName string, answers []config.CircleValidated) []int {
	imgPath := filepath.Join(tempDir, homoName)

	// Charger l'image en niveaux de gris
	img := gocv.IMRead(imgPath, gocv.IMReadGrayScale)
	if img.Empty() {
		panic("Impossible de charger l'image")
	}
	defer img.Close()

	// fmt.Println("ROI pour :", homoName)

	states := make([]int, len(answers))
	threshold := 150.0

	for i, answer := range answers {
		// Définir la ROI (x, y, largeur, hauteur)
		x1 := answer.Position.X - int(answer.Radius/2)
		y1 := answer.Position.Y - int(answer.Radius/2)
		x2 := answer.Position.X + int(answer.Radius/2)
		y2 := answer.Position.Y + int(answer.Radius/2)
		rect := image.Rect(x1, y1, x2, y2)

		roi := img.Region(rect)
		mean := roi.Mean()
		roi.Close()      // on ferme explicitement au lieu d'utiliser defer
		avg := mean.Val1 // pour N&B -> valeur moyenne du canal 0

		// fmt.Printf("Moyenne ROI #%d = %.2f\n", i, avg)

		if avg < threshold {
			states[i] = 1
		} else {
			states[i] = 0
		}

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

	return states
}
