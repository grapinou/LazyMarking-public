package tools

import (
	"errors"
	"log"

	"gocv.io/x/gocv"
)

func DecodeWithGoCV(imgPath string) (string, error) {
	img := gocv.IMRead(imgPath, gocv.IMReadColor)
	if img.Empty() {
		log.Println("from DecodeWithGoCV can't read image")
		return "", errors.New("from DecodeWithGoCV can't read image")
	}
	defer img.Close()

	// Crée le détecteur de QR codes
	detector := gocv.NewQRCodeDetector()
	defer detector.Close()

	points := gocv.NewMat()
	defer points.Close()
	straight := gocv.NewMat()
	defer straight.Close()

	// Détecte et décode en une seule étape
	result := detector.DetectAndDecode(img, &points, &straight)
	if result == "" {
		log.Println("from DecodeWithGoCV no qr code detected")
		return "", errors.New("from DecodeWithGoCV no qr code detected")
	}

	return result, nil
}
