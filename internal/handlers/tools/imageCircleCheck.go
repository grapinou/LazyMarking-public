package tools

import (
	"image"
	"log"

	"gocv.io/x/gocv"
)

// Une fonction pour savoir si une image contient des cercles équivalent à ceux des questions
// si aucun cercle n'est détecté, renvoie true
func ImageCircleCheck(tempDir, imgName string, scale float64) bool {
	imgPath, _, err := validateRegularFile(tempDir, imgName)
	if err != nil {
		log.Printf("unsafe image path: %v", err)
		return false
	}

	// Lire l'image en niveaux de gris
	gray := gocv.IMRead(imgPath, gocv.IMReadGrayScale)
	if gray.Empty() {
		log.Println("impossible de lire l'image")
		return false
	}
	defer gray.Close()

	// Calcul des nouvelles dimensions
	newWidth, newHeight, err := ValidateImageResize(gray.Cols(), gray.Rows(), scale)
	if err != nil {
		log.Printf("dimensions de redimensionnement invalides: %v", err)
		return false
	}

	// Redimensionnement
	resized := gocv.NewMat()
	defer resized.Close()

	gocv.Resize(gray, &resized, image.Pt(newWidth, newHeight), 0, 0, gocv.InterpolationLinear)

	// Pré-traitement : flou médian
	gocv.MedianBlur(resized, &resized, 5)

	// Détection de cercles
	circles := gocv.NewMat()
	defer circles.Close()

	gocv.HoughCirclesWithParams(
		resized, &circles, gocv.HoughGradient,
		1,   // dp (inverse ratio de résolution)
		20,  // minDist (distance entre centres)
		100, // param1 (Canny threshold haut)
		30,  // param2 (accumulator threshold)
		18,  // minRadius
		23,  // maxRadius
	)

	// Vérifier si des cercles sont trouvés
	if circles.Cols() == 0 {
		return true
	}
	return false
}
