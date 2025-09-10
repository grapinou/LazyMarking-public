package tools

import (
	"image"
	"log"
	"os"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

func DecodeWithGozxing(imgPath string) (string, error) {
	// Ouvre l’image
	file, err := os.Open(imgPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("Can't decode file, error : %v", err)
		return "", err

	}

	// Convertit en source binaire
	bitmap, _ := gozxing.NewBinaryBitmapFromImage(img)

	// Décodeur QR code

	reader := qrcode.NewQRCodeReader()
	result, err := reader.Decode(bitmap, nil)
	if err != nil {
		log.Println("QR code not found :", err)
		return "", err
	}

	return result.String(), nil
}
