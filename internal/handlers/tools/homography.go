package tools

import (
	"fmt"
	"image"
	"path/filepath"
	"sort"

	"gocv.io/x/gocv"
)

func Homography(tempDir, pngFromPdf, pngBase string) string {
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
	matches := bf.Match(desc1, desc2)

	// Trier les matches par distance (plus petit = meilleur)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Distance < matches[j].Distance
	})

	// Garder seulement les 50 meilleurs matches
	goodMatches := matches
	if len(matches) > 50 {
		goodMatches = matches[:50]
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
		5,     // seuil RANSAC
		&mask, // masque de correspondance
		2000,  // itérations max
		0.995, // confiance
	)
	defer H.Close()
	fmt.Println("Matrice H =", H)

	// Appliquer la transformation
	warped := gocv.NewMat()
	gocv.WarpPerspective(img1, &warped, H, image.Pt(img2.Cols(), img2.Rows()))
	defer warped.Close()

	// Créer une image de sortie et y coller img2
	result := warped.Clone()
	defer result.Close()
	roi := result.Region(image.Rect(0, 0, img2.Cols(), img2.Rows()))
	img2.CopyTo(&roi)
	roi.Close()

	name := "homography.png"
	saveResult := filepath.Join(tempDir, name)
	gocv.IMWrite(saveResult, result)

	return name

	/*
		// Afficher le panorama
		window := gocv.NewWindow("Panorama")
		defer window.Close()
		for {
			window.IMShow(result)
			if window.WaitKey(1) >= 0 {
				break
			}
		}

		   // Dessiner les matches pour visualiser
		   matchImg := gocv.NewMat()
		   gocv.DrawMatches(img1, kps1, img2, kps2, goodMatches, &matchImg,

		   	color.RGBA{0, 255, 0, 0}, // couleur des matches
		   	color.RGBA{255, 0, 0, 0}, // couleur des keypoints
		   	nil,
		   	gocv.DrawDefault)

		   defer matchImg.Close()

		   // Afficher les résultats
		   window1 := gocv.NewWindow("Matches ORB")
		   window2 := gocv.NewWindow("Image 1 transformée")
		   defer window1.Close()
		   defer window2.Close()

		   	for {
		   		window1.IMShow(matchImg)
		   		window2.IMShow(warped)

		   		if window1.WaitKey(1) >= 0 {
		   			break
		   		}
		   	}
	*/
}
