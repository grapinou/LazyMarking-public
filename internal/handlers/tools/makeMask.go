package tools

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

// MakeMask crée un masque noir avec une zone d'intérêt blanche
// img : l'image de référence (pour récupérer taille)
// roi : rectangle de la zone d'intérêt en pixels
func MakeMask(img gocv.Mat, roi image.Rectangle) gocv.Mat {
	// masque noir, même taille que l'image
	mask := gocv.NewMatWithSize(img.Rows(), img.Cols(), gocv.MatTypeCV8U)
	mask.SetTo(gocv.NewScalar(0, 0, 0, 0)) // tout noir

	// dessine un rectangle blanc dans la zone d'intérêt
	gocv.Rectangle(&mask, roi, color.RGBA{255, 255, 255, 0}, -1)

	return mask
}
