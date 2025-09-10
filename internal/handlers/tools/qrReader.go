package tools

import "errors"

var ErrQrCodeNoReading = errors.New("qr code not detected or be read")

func QrReader(imgPath string) (string, error) {
	// 1. Essayer avec gozxing
	if result, err := DecodeWithGozxing(imgPath); err == nil && result != "" {
		return result, nil
	}

	// 2. Fallback avec GoCV
	if result, err := DecodeWithGoCV(imgPath); err == nil && result != "" {
		return result, nil
	}

	return "", ErrQrCodeNoReading
}
