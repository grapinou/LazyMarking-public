package tools

import (
	"fmt"
	"image"
	"path/filepath"
	"sort"
	"strings"

	"gocv.io/x/gocv"
)

func Homography(tempDir, pngFromPdf, pngBase string) string {
	fromPdf := filepath.Join(tempDir, pngFromPdf)
	baseImg := filepath.Join(tempDir, pngBase)

	// Charger les images
	img1 := gocv.IMRead(fromPdf, gocv.IMReadColor)
	img2 := gocv.IMRead(baseImg, gocv.IMReadColor)
	if img1.Empty() || img2.Empty() {
		fmt.Println("Erreur : impossible de charger les images")
		return ""
	}
	defer img1.Close()
	defer img2.Close()

	// --- Prétraitement : grayscale + threshold adaptatif ---
	gray1 := gocv.NewMat()
	gray2 := gocv.NewMat()
	defer gray1.Close()
	defer gray2.Close()

	gocv.CvtColor(img1, &gray1, gocv.ColorBGRToGray)
	gocv.CvtColor(img2, &gray2, gocv.ColorBGRToGray)

	// améliore la netteté des contours du texte
	gocv.AdaptiveThreshold(gray1, &gray1, 255, gocv.AdaptiveThresholdMean, gocv.ThresholdBinary, 15, 5)
	gocv.AdaptiveThreshold(gray2, &gray2, 255, gocv.AdaptiveThresholdMean, gocv.ThresholdBinary, 15, 5)

	// --- Détecteur AKAZE ---
	akaze := gocv.NewAKAZE()
	defer akaze.Close()

	kps1, desc1 := akaze.DetectAndCompute(gray1, gocv.NewMat())
	defer desc1.Close()
	kps2, desc2 := akaze.DetectAndCompute(gray2, gocv.NewMat())
	defer desc2.Close()

	if desc1.Empty() || desc2.Empty() {
		fmt.Println("Descripteurs vides")
		return ""
	}

	// --- Matcher BF + KNN ---
	bf := gocv.NewBFMatcher()
	defer bf.Close()
	matches := bf.KnnMatch(desc1, desc2, 2)

	// --- Ratio test + distance max ---
	var good []gocv.DMatch
	for _, m := range matches {
		if len(m) == 2 && m[0].Distance < 0.65*m[1].Distance {
			good = append(good, m[0])
		}
	}

	if len(good) < 4 {
		fmt.Printf("Pas assez de correspondances : %d\n", len(good))
		return ""
	}

	// --- Garder uniquement les N meilleurs matches ---
	N := 500
	if len(good) > N {
		sort.Slice(good, func(i, j int) bool {
			return good[i].Distance < good[j].Distance
		})
		good = good[:N]
	}

	// --- Construire les matrices de points ---
	srcPts := gocv.NewMatWithSize(len(good), 1, gocv.MatTypeCV32FC2)
	dstPts := gocv.NewMatWithSize(len(good), 1, gocv.MatTypeCV32FC2)
	defer srcPts.Close()
	defer dstPts.Close()

	for i, m := range good {
		srcPts.SetFloatAt(i, 0, float32(kps1[m.QueryIdx].X))
		srcPts.SetFloatAt(i, 1, float32(kps1[m.QueryIdx].Y))
		dstPts.SetFloatAt(i, 0, float32(kps2[m.TrainIdx].X))
		dstPts.SetFloatAt(i, 1, float32(kps2[m.TrainIdx].Y))
	}

	// --- Estimer l’homographie avec RANSAC plus strict ---
	mask := gocv.NewMat()
	defer mask.Close()
	H := gocv.FindHomography(srcPts, dstPts, gocv.HomographyMethodRANSAC, 2.0, &mask, 10000, 0.9999)
	defer H.Close()

	if H.Empty() {
		fmt.Println("FindHomography a échoué")
		return ""
	}

	// --- Appliquer la transformation ---
	warped := gocv.NewMat()
	defer warped.Close()
	gocv.WarpPerspective(img1, &warped, H, image.Pt(img2.Cols(), img2.Rows()))

	// --- Fusion simple : coller warped sur le fond ---
	result := img2.Clone()
	defer result.Close()

	roi := result.Region(image.Rect(0, 0, warped.Cols(), warped.Rows()))
	warped.CopyTo(&roi)
	roi.Close()

	// --- Sauvegarde ---
	name := strings.TrimSuffix(pngBase, filepath.Ext(pngBase))
	fullName := name + "_homography_enhanced.png"
	saveResult := filepath.Join(tempDir, fullName)
	gocv.IMWrite(saveResult, result)

	// fmt.Printf("Homography align done, good matches: %d\n", len(good))
	return fullName
}

/*


	fromPdf := filepath.Join(tempDir, pngFromPdf)
	baseImg := filepath.Join(tempDir, pngBase)

	// Charger les images
	img1 := gocv.IMRead(fromPdf, gocv.IMReadColor)
	img2 := gocv.IMRead(baseImg, gocv.IMReadColor)
	if img1.Empty() || img2.Empty() {
		fmt.Println("Erreur : impossible de charger les images")
		return ""
	}
	defer img1.Close()
	defer img2.Close()

	// Détecteur + descripteurs (AKAZE pour les scans)
	akaze := gocv.NewAKAZE()
	defer akaze.Close()

	kps1, desc1 := akaze.DetectAndCompute(img1, gocv.NewMat())
	defer desc1.Close()
	kps2, desc2 := akaze.DetectAndCompute(img2, gocv.NewMat())
	defer desc2.Close()

	if desc1.Empty() || desc2.Empty() {
		fmt.Println("Descripteurs vides")
		return ""
	}

	// Brute Force matcher
	bf := gocv.NewBFMatcher()
	defer bf.Close()
	matches := bf.KnnMatch(desc1, desc2, 2)

	// Ratio test de Lowe
	var good []gocv.DMatch
	for _, m := range matches {
		if len(m) == 2 && m[0].Distance < 0.7*m[1].Distance { // ratio plus strict
			good = append(good, m[0])
		}
	}

	if len(good) < 4 {
		fmt.Printf("Pas assez de correspondances : %d\n", len(good))
		return ""
	}

	// Construire les matrices de points
	srcPts := gocv.NewMatWithSize(len(good), 1, gocv.MatTypeCV32FC2)
	dstPts := gocv.NewMatWithSize(len(good), 1, gocv.MatTypeCV32FC2)
	defer srcPts.Close()
	defer dstPts.Close()

	for i, m := range good {
		srcPts.SetFloatAt(i, 0, float32(kps1[m.QueryIdx].X))
		srcPts.SetFloatAt(i, 1, float32(kps1[m.QueryIdx].Y))
		dstPts.SetFloatAt(i, 0, float32(kps2[m.TrainIdx].X))
		dstPts.SetFloatAt(i, 1, float32(kps2[m.TrainIdx].Y))
	}

	// Estimer l’homographie avec RANSAC
	mask := gocv.NewMat()
	defer mask.Close()
	H := gocv.FindHomography(srcPts, dstPts, gocv.HomographyMethodRANSAC, 3, &mask, 10000, 0.999)
	defer H.Close()

	if H.Empty() {
		fmt.Println("FindHomography a échoué")
		return ""
	}

	// Appliquer la transformation
	warped := gocv.NewMat()
	defer warped.Close()
	gocv.WarpPerspective(img1, &warped, H, image.Pt(img2.Cols(), img2.Rows()))

	// Fusion simple : coller warped sur le fond
	result := img2.Clone()
	defer result.Close()

	roi := result.Region(image.Rect(0, 0, warped.Cols(), warped.Rows()))
	warped.CopyTo(&roi)
	roi.Close()

	// Sauvegarder
	name := strings.TrimSuffix(pngBase, filepath.Ext(pngBase))
	fullName := name + "_homography_align.png"
	saveResult := filepath.Join(tempDir, fullName)
	gocv.IMWrite(saveResult, result)

	return fullName

*/
