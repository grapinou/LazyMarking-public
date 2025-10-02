package tools

import (
	"errors"
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
		log.Println("from DecodeWithGozxing can't read image")
		return "", errors.New("from DecodeWithGozxing can't read image")
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("From DecodeWithGozxin : can't decode file, error : %v", err)
		return "", errors.New("from DecodeWithGozxing can't decode image")

	}

	// Convertit en source binaire
	bitmap, _ := gozxing.NewBinaryBitmapFromImage(img)

	// Décodeur QR code

	reader := qrcode.NewQRCodeReader()
	result, err := reader.Decode(bitmap, nil)
	if err != nil {
		log.Println("From DecodeWithGozxin: QR code not found :", err)
		return "", err
	}

	return result.String(), nil
}
