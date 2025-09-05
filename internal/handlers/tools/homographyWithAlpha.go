package tools

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"gocv.io/x/gocv"
)

func HomographyWithAlpha(tempDir, pngFromPdf, pngBase string) string {
	// Charger deux images
	fromPdf := filepath.Join(tempDir, pngFromPdf)
	baseImg := filepath.Join(tempDir, pngBase)
	img1 := gocv.IMRead(fromPdf, gocv.IMReadColor)
	img2 := gocv.IMRead(baseImg, gocv.IMReadColor)
	if img1.Empty() || img2.Empty() {
		fmt.Println("Erreur : impossible de charger les images")
		return ""
	}
	defer img1.Close()
	defer img2.Close()

	// Créer ORB
	orb := gocv.NewORB()
	defer orb.Close()

	// Détecter et calculer les descripteurs
	kps1, desc1 := orb.DetectAndCompute(img1, gocv.NewMat())
	kps2, desc2 := orb.DetectAndCompute(img2, gocv.NewMat())

	// Créer un matcher BF (Brute Force) avec norme Hamming
	bf := gocv.NewBFMatcher()

	// KNN match (2 voisins)
	matches := bf.KnnMatch(desc1, desc2, 2)

	// Ratio test de Lowe
	goodMatches := make([]gocv.DMatch, 0)
	ratio := float64(0.75)
	for _, m := range matches {
		if len(m) == 2 && m[0].Distance < ratio*m[1].Distance {
			goodMatches = append(goodMatches, m[0])
		}
	}

	// Vérifier qu'on a assez de points
	if len(goodMatches) < 4 {
		fmt.Println("Pas assez de correspondances pour calculer une homographie")
		return ""
	}

	// Construire les points correspondants
	srcPts := gocv.NewMatWithSize(len(goodMatches), 1, gocv.MatTypeCV32FC2)
	dstPts := gocv.NewMatWithSize(len(goodMatches), 1, gocv.MatTypeCV32FC2)

	for i, m := range goodMatches {
		p1 := image.Pt(int(kps1[m.QueryIdx].X), int(kps1[m.QueryIdx].Y))
		p2 := image.Pt(int(kps2[m.TrainIdx].X), int(kps2[m.TrainIdx].Y))

		srcPts.SetFloatAt(i, 0, float32(p1.X))
		srcPts.SetFloatAt(i, 1, float32(p1.Y))

		dstPts.SetFloatAt(i, 0, float32(p2.X))
		dstPts.SetFloatAt(i, 1, float32(p2.Y))
	}

	// Calculer l’homographie avec RANSAC
	mask := gocv.NewMat()
	H := gocv.FindHomography(
		srcPts, dstPts,
		gocv.HomographyMethodRANSAC,
		3,     // seuil RANSAC plus strict
		&mask, // masque de correspondance
		5000,  // itérations max
		0.999, // confiance
	)
	defer H.Close()

	// Appliquer la transformation
	warped := gocv.NewMat()
	gocv.WarpPerspective(img1, &warped, H, image.Pt(img2.Cols(), img2.Rows()))
	defer warped.Close()

	// Créer une image de sortie et y coller img2
	result := warped.Clone()
	defer result.Close()

	// Conversion en BGRA pour transparence
	img2BGRA := gocv.NewMat()
	gocv.CvtColor(img2, &img2BGRA, gocv.ColorBGRToBGRA)
	defer img2BGRA.Close()

	// Fixer alpha (transparence du pngBase)
	alpha := uint8(100)
	for y := 0; y < img2BGRA.Rows(); y++ {
		for x := 0; x < img2BGRA.Cols(); x++ {
			img2BGRA.SetUCharAt(y, x*4+3, alpha)
		}
	}

	// Convertir warped → BGRA
	warpedBGRA := gocv.NewMat()
	gocv.CvtColor(warped, &warpedBGRA, gocv.ColorBGRToBGRA)
	defer warpedBGRA.Close()

	// Copier img2 transparent dans le résultat
	result = warpedBGRA.Clone()
	roi := result.Region(image.Rect(0, 0, img2BGRA.Cols(), img2BGRA.Rows()))
	img2BGRA.CopyTo(&roi)
	roi.Close()

	// Sauvegarder
	name := strings.TrimSuffix(pngBase, filepath.Ext(pngBase))
	fullName := name + "_homography.png"
	saveResult := filepath.Join(tempDir, fullName)
	gocv.IMWrite(saveResult, result)

	return fullName
}
