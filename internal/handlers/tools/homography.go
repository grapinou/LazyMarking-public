package tools

import (
	"fmt"
	"image"
	"path/filepath"
	"sort"
	"strings"

	"gocv.io/x/gocv"
)

func Homography(tempDir, pngFromPdf, pngBase string) (string, error) {
	fromPdf := filepath.Join(tempDir, pngFromPdf)
	baseImg := filepath.Join(tempDir, pngBase)
	if filepath.IsAbs(pngFromPdf) {
		fromPdf = pngFromPdf
	}
	if filepath.IsAbs(pngBase) {
		baseImg = pngBase
	}

	// Charger les images
	img1 := gocv.IMRead(fromPdf, gocv.IMReadColor)
	defer img1.Close()
	if img1.Empty() {
		return "", fmt.Errorf("load homography source image %q", pngFromPdf)
	}

	img2 := gocv.IMRead(baseImg, gocv.IMReadColor)
	defer img2.Close()
	if img2.Empty() {
		return "", fmt.Errorf("load homography reference image %q", pngBase)
	}

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

	sift := gocv.NewSIFT()
	defer sift.Close()
	kps1, desc1 := sift.DetectAndCompute(gray1, gocv.NewMat())
	kps2, desc2 := sift.DetectAndCompute(gray2, gocv.NewMat())

	defer desc1.Close()
	defer desc2.Close()

	if desc1.Empty() || desc2.Empty() {
		return "", fmt.Errorf("compute SIFT descriptors: source or reference descriptors are empty")
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
		return "", fmt.Errorf("match SIFT descriptors: got %d good matches, need at least 4", len(good))
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
		return "", fmt.Errorf("find homography: resulting matrix is empty")
	}

	// --- Appliquer la transformation ---
	warped := gocv.NewMat()
	defer warped.Close()
	gocv.WarpPerspective(img1, &warped, H, image.Pt(img2.Cols(), img2.Rows()))
	if warped.Empty() {
		return "", fmt.Errorf("warp homography perspective: resulting image is empty")
	}

	// --- Fusion simple : coller warped sur le fond ---
	result := img2.Clone()
	defer result.Close()

	roi := result.Region(image.Rect(0, 0, warped.Cols(), warped.Rows()))
	warped.CopyTo(&roi)
	roi.Close()

	// --- Sauvegarde ---
	name := strings.TrimSuffix(filepath.Base(pngBase), filepath.Ext(pngBase))
	if filepath.IsAbs(pngBase) {
		name = filepath.Base(filepath.Dir(pngBase)) + "-" + name
	}
	fullName := name + "_homography_enhanced.png"
	saveResult := filepath.Join(tempDir, fullName)
	if ok := gocv.IMWrite(saveResult, result); !ok {
		return "", fmt.Errorf("write homography result %q", fullName)
	}

	// fmt.Printf("Homography align done, good matches: %d\n", len(good))
	return fullName, nil
}
