package tools

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"gocv.io/x/gocv"
)

var DebugQR = false // Mets à false pour désactiver

func DecodeWithGoCV(imgPath string) (string, error) {
	img := gocv.IMRead(imgPath, gocv.IMReadColor)
	if img.Empty() {
		log.Println("from DecodeWithGoCV can't read image")
		return "", errors.New("from DecodeWithGoCV can't read image")
	}
	defer img.Close()

	// Découpe la zone fixe en haut à gauche
	roi := CropTopLeft(img, 0.25, 0.15) // ajuster largeur/hauteur selon placement réel
	defer roi.Close()
	DebugSave(roi, "roi")

	detector := gocv.NewQRCodeDetector()
	defer detector.Close()

	points := gocv.NewMat()
	defer points.Close()
	straight := gocv.NewMat()
	defer straight.Close()

	if result, err := DecodeROIWithGozxing(roi); err == nil && result != "" {
		return result, nil
	}

	// 1. Essai brut
	if result := detector.DetectAndDecode(roi, &points, &straight); result != "" {
		return result, nil
	}

	// 2. Prétraitement : gris + equalize hist
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(roi, &gray, gocv.ColorBGRToGray)
	DebugSave(gray, "gray")

	if result, err := DecodeROIWithGozxing(gray); err == nil && result != "" {
		return result, nil
	}

	if result := detector.DetectAndDecode(gray, &points, &straight); result != "" {
		return result, nil
	}

	// 3. Prétraitement : seuillage adaptatif
	blur := gocv.NewMat()
	gocv.GaussianBlur(gray, &blur, image.Pt(3, 3), 0, 0, gocv.BorderDefault)
	adapt := gocv.NewMat()
	defer adapt.Close()
	gocv.AdaptiveThreshold(
		blur, &adapt,
		255,
		gocv.AdaptiveThresholdGaussian, // ou AdaptiveThresholdMean
		gocv.ThresholdBinary,
		25, // taille fenêtre (impair, ex: 11, 15, 25)
		10, // constante soustraite
	)
	DebugSave(adapt, "thresh_adapt")

	// Test sur adapt (Gozxing + GoCV)
	if result, err := DecodeROIWithGozxing(adapt); err == nil && result != "" {
		return result, nil
	}
	if result := detector.DetectAndDecode(adapt, &points, &straight); result != "" {
		return result, nil
	}

	// 4. Fallbacks plus coûteux : plusieurs cadrages puis agrandissement.
	// Certaines numérisations historiques déforment légèrement les modules du QR ;
	// varier la marge et rééchantillonner aide les décodeurs sans dépendre de la
	// position physique de la page dans le PDF.
	fallbackRatios := []struct {
		width  float64
		height float64
	}{
		{width: 0.25, height: 0.15},
		{width: 0.30, height: 0.20},
		{width: 0.35, height: 0.25},
	}
	for _, ratios := range fallbackRatios {
		fallbackROI := CropTopLeft(img, ratios.width, ratios.height)
		result := decodeUpscaledQRROI(fallbackROI, detector)
		fallbackROI.Close()
		if result != "" {
			return result, nil
		}
	}

	log.Println("from DecodeWithGoCV no qr code detected in top-left ROIs")
	return "", errors.New("from DecodeWithGoCV no qr code detected")
}

func decodeUpscaledQRROI(roi gocv.Mat, detector gocv.QRCodeDetector) string {
	for _, scale := range []int{2, 3} {
		upscaled := gocv.NewMat()
		gocv.Resize(roi, &upscaled, image.Pt(roi.Cols()*scale, roi.Rows()*scale), 0, 0, gocv.InterpolationCubic)

		if result, err := DecodeROIWithGozxing(upscaled); err == nil && result != "" {
			upscaled.Close()
			return result
		}
		if result := decodeQRWithDetector(detector, upscaled); result != "" {
			upscaled.Close()
			return result
		}

		gray := gocv.NewMat()
		gocv.CvtColor(upscaled, &gray, gocv.ColorBGRToGray)
		upscaled.Close()
		if result, err := DecodeROIWithGozxing(gray); err == nil && result != "" {
			gray.Close()
			return result
		}
		if result := decodeQRWithDetector(detector, gray); result != "" {
			gray.Close()
			return result
		}

		binary := gocv.NewMat()
		gocv.Threshold(gray, &binary, 0, 255, gocv.ThresholdBinary+gocv.ThresholdOtsu)
		gray.Close()
		if result, err := DecodeROIWithGozxing(binary); err == nil && result != "" {
			binary.Close()
			return result
		}
		if result := decodeQRWithDetector(detector, binary); result != "" {
			binary.Close()
			return result
		}
		binary.Close()
	}
	return ""
}

func decodeQRWithDetector(detector gocv.QRCodeDetector, mat gocv.Mat) string {
	points := gocv.NewMat()
	defer points.Close()
	straight := gocv.NewMat()
	defer straight.Close()
	return detector.DetectAndDecode(mat, &points, &straight)
}

// DecodeROIWithGozxing : applique gozxing sur un Mat (ROI déjà extrait)
func DecodeROIWithGozxing(mat gocv.Mat) (string, error) {
	// Convertir Mat → image.Image (via buffer PNG)
	buf, err := gocv.IMEncode(gocv.PNGFileExt, mat)
	if err != nil {
		return "", errors.New("gozxing: cannot encode ROI to PNG")
	}
	defer buf.Close()

	img, err := png.Decode(bytes.NewReader(buf.GetBytes()))
	if err != nil {
		log.Println("gozxing: decode PNG failed:", err)
		return "", errors.New("gozxing: cannot decode PNG")
	}

	// Convertir en BinaryBitmap
	bmp, _ := gozxing.NewBinaryBitmapFromImage(img)

	// Décodage QR
	reader := qrcode.NewQRCodeReader()
	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}
	result, err := reader.Decode(bmp, hints)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

// cropTopLeft extrait une ROI dans le coin supérieur gauche.
// widthRatio et heightRatio = pourcentage de la taille totale.
func CropTopLeft(img gocv.Mat, widthRatio, heightRatio float64) gocv.Mat {
	rows := img.Rows()
	cols := img.Cols()

	w := int(float64(cols) * widthRatio)
	h := int(float64(rows) * heightRatio)

	x := 0
	y := 0

	rect := image.Rect(x, y, w, h)
	return img.Region(rect)
}

func DebugSave(mat gocv.Mat, stage string) {
	if !DebugQR {
		return
	}

	// Crée dossier si inexistant
	dir := "./debug_qr"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Println("cannot create debug dir:", err)
		return
	}

	// Nom fichier avec timestamp
	filename := fmt.Sprintf("%s_%d.png", stage, time.Now().UnixNano())
	path := filepath.Join(dir, filename)

	if ok := gocv.IMWrite(path, mat); !ok {
		log.Println("failed to save debug image:", path)
	} else {
		log.Println("saved debug image:", path)
	}
}
