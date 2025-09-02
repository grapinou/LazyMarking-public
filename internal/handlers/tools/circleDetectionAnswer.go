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
func CircleDetectionAnswer(tempDir, imgName string, topLimit, bottomLimit int) ([]config.CircleValidated, bool) {
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
	roi := image.Rect(0, topLimit, gray.Cols(), bottomLimit)

	// Créer le masque
	mask := MakeMask(gray, roi)
	defer mask.Close()

	// Appliquer le masque
	masked := gocv.NewMat()
	defer masked.Close()
	gocv.BitwiseAnd(gray, mask, &masked)

	// Pré-traitement : flou + seuillage
	gocv.MedianBlur(masked, &masked, 5)

	maskedImg := filepath.Join(tempDir, "maskedImg_"+imgName)
	gocv.IMWrite(maskedImg, masked)

	circles := gocv.NewMat()
	defer circles.Close()

	gocv.HoughCirclesWithParams(
		masked, &circles, gocv.HoughGradient,
		1,   // dp
		20,  // minDist
		100, // param1 (Canny high threshold)
		30,  // param2 (accumulator threshold, ajuste entre 20 et 50)
		18,  // minRadius (tes ronds noirs sont petits)
		23,  // maxRadius
	)

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
		if mean.Val1 > 200 { // suffisamment blanc
			validateCircle = append(validateCircle, config.CircleValidated{
				Position: config.Position{
					X: int(x),
					Y: int(y),
				},
				Radius: int(r),
			})
		}

	}

	if len(validateCircle) == 0 {
		return validateCircle, true
	}

	// Tri par Y (croissant)
	sort.Slice(validateCircle, func(i, j int) bool {
		return validateCircle[i].Position.Y < validateCircle[j].Position.Y
	})

	var sortedCircle []config.CircleValidated

	elements := len(validateCircle)
	for elements > 1 {

		leftSide := validateCircle[:2]
		sort.Slice(leftSide, func(i, j int) bool {
			return leftSide[i].Position.X < leftSide[j].Position.X
		})
		sortedCircle = append(sortedCircle, leftSide...)
		validateCircle = validateCircle[2:]
		elements = len(validateCircle)
	}
	if len(validateCircle) == 1 {
		sortedCircle = append(sortedCircle, validateCircle[0])
	}

	/*
		// Charger image couleur pour tracer les cercles
		colorImg := gocv.IMRead(imgPath, gocv.IMReadColor)
		defer colorImg.Close()
		// pour tracer des cercles autours des cercles trouvés
		fmt.Println("entourage des cercles détectés dans l'ordre")
		for i, circle := range sortedCircle {
			greenColor := uint8(50 + 50*i)
			gocv.Circle(&colorImg, image.Pt(circle.Position.X, circle.Position.Y), circle.Radius, color.RGBA{0, greenColor, 0, 0}, 8)
		}
		result := filepath.Join(tempDir, "answer_result_"+imgName)
		gocv.IMWrite(result, colorImg)
	*/
	return sortedCircle, true
}

/*
Cercle trouvé: centre=(505.50, 814.50), rayon=20.50
Cercle trouvé: centre=(193.50, 977.50), rayon=20.20
Cercle trouvé: centre=(194.50, 814.50), rayon=19.90
cercle validés : [{{505 814} 20} {{193 977} 20} {{194 814} 19}]
[{{194 1537} 19} {{742 1537} 20} {{194 1699} 19} {{742 1699} 20}]
*/
