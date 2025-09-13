package tools

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"

	"gocv.io/x/gocv"
)

func Homography(tempDir, pngFromPdf, pngBase string) string {
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

	orb := gocv.NewORB()
	defer orb.Close()

	mask1 := gocv.NewMat()
	defer mask1.Close()
	mask2 := gocv.NewMat()
	defer mask2.Close()

	kps1, desc1 := orb.DetectAndCompute(img1, mask1)
	defer desc1.Close()
	kps2, desc2 := orb.DetectAndCompute(img2, mask2)
	defer desc2.Close()

	if desc1.Empty() || desc2.Empty() {
		fmt.Println("Descripteurs vides")
		return ""
	}

	bf := gocv.NewBFMatcher()
	defer bf.Close()

	matches := bf.KnnMatch(desc1, desc2, 2)

	// Lowe ratio test
	var good []gocv.DMatch
	for _, m := range matches {
		if len(m) == 2 && m[0].Distance < float64(0.75)*m[1].Distance {
			good = append(good, m[0])
		}
	}

	if len(good) < 3 {
		fmt.Println("Pas assez de correspondances pour estimer une affinité")
		return ""
	}

	// Construire slices de gocv.Point2f (attention au cast float32)
	srcPoints := make([]gocv.Point2f, 0, len(good))
	dstPoints := make([]gocv.Point2f, 0, len(good))

	for _, m := range good {
		srcPoints = append(srcPoints, gocv.Point2f{
			X: float32(kps1[m.QueryIdx].X),
			Y: float32(kps1[m.QueryIdx].Y),
		})
		dstPoints = append(dstPoints, gocv.Point2f{
			X: float32(kps2[m.TrainIdx].X),
			Y: float32(kps2[m.TrainIdx].Y),
		})
	}

	srcPts := gocv.NewPoint2fVectorFromPoints(srcPoints)
	defer srcPts.Close()
	dstPts := gocv.NewPoint2fVectorFromPoints(dstPoints)
	defer dstPts.Close()

	M := gocv.EstimateAffine2D(srcPts, dstPts)
	defer M.Close()
	if M.Empty() {
		fmt.Println("EstimateAffine2D a échoué")
		return ""
	}

	warped := gocv.NewMat()
	defer warped.Close()
	gocv.WarpAffine(img1, &warped, M, image.Pt(img2.Cols(), img2.Rows()))

	// Coller le warped SUR le fond (img2)
	result := img2.Clone()
	defer result.Close()

	roi := result.Region(image.Rect(0, 0, warped.Cols(), warped.Rows()))
	warped.CopyTo(&roi)
	roi.Close()

	// Sauvegarde
	name := strings.TrimSuffix(pngBase, filepath.Ext(pngBase))
	fullName := name + "_homo_affine_fixed.png"
	saveResult := filepath.Join(tempDir, fullName)
	gocv.IMWrite(saveResult, result)

	return fullName
}
