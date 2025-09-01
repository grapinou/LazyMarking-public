package tools

import (
	"image"
	"log"
	"path/filepath"
	"sort"

	"github.com/grapinou/LazyMarking/internal/config"
	"gocv.io/x/gocv"
)

// permet de trouver les cercles des questions. Revoi les positions des centres des cercles et leurs rayons
// la liste est triées par ordre croissant.
func CircleDetection(tempDir, imgName string) ([]config.CircleValidated, bool) {
	imgPath := filepath.Join(tempDir, imgName)
	var validateCircle []config.CircleValidated

	// Lire l'image en niveaux de gris
	gray := gocv.IMRead(imgPath, gocv.IMReadGrayScale)
	if gray.Empty() {
		log.Println("impossible de lire l'image")
		return validateCircle, false
	}
	defer gray.Close()

	// Définir la zone d'intérêt (par ex. les 200 premiers pixels en X)
	// x0, y0, x1, y1 pour les coins d'un rectangle
	roi := image.Rect(110, 420, 235, gray.Rows()-120)

	// Créer le masque
	mask := MakeMask(gray, roi)
	defer mask.Close()

	// Appliquer le masque
	masked := gocv.NewMat()
	defer masked.Close()
	gocv.BitwiseAnd(gray, mask, &masked)

	// Pré-traitement : flou + seuillage
	gocv.MedianBlur(masked, &masked, 5)

	circles := gocv.NewMat()
	defer circles.Close()

	gocv.HoughCirclesWithParams(
		masked, &circles, gocv.HoughGradient,
		1,   // dp
		20,  // minDist
		100, // param1 (Canny high threshold)
		30,  // param2 (accumulator threshold, ajuste entre 20 et 50)
		30,  // minRadius (tes ronds noirs sont petits)
		35,  // maxRadius
	)

	// Charger image couleur pour tracer les cercles
	// colorImg := gocv.IMRead(imgPath, gocv.IMReadColor)
	// defer colorImg.Close()

	// Lire les cercles détectés
	for i := 0; i < circles.Cols(); i++ {
		v := circles.GetVecfAt(0, i)
		x, y, r := v[0], v[1], v[2]
		// 3. Vérifier que c’est bien noir
		roi := gray.Region(image.Rect(
			int(x-(r/2)), int(y-(r/2)),
			int(x+(r/2)), int(y+(r/2)),
		))
		mean := roi.Mean()
		roi.Close()
		if mean.Val1 < 50 { // suffisamment noir
			validateCircle = append(validateCircle, config.CircleValidated{
				Position: config.Position{
					X: int(x),
					Y: int(y),
				},
				Radius: int(r),
			})
		}

		// pour tracer des cercles autours des cercles trouvés
		// fmt.Printf("Cercle trouvé: centre=(%.2f, %.2f), rayon=%.2f\n", x, y, r)
		// gocv.Circle(&colorImg, image.Pt(int(x), int(y)), int(r), color.RGBA{255, 0, 0, 0}, 2)
	}

	// result := filepath.Join(tempDir, "result_"+imgName)
	// gocv.IMWrite(result, colorImg)

	if len(validateCircle) == 0 {
		return validateCircle, false
	}

	// Tri par Y (croissant)
	sort.Slice(validateCircle, func(i, j int) bool {
		return validateCircle[i].Position.Y < validateCircle[j].Position.Y
	})
	return validateCircle, true
}
