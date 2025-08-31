package tools

import (
	"encoding/json"
	"log"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/skip2/go-qrcode"
)

func QrCodeMaker(tempDir string, info config.QrCodeInfo) (string, bool) {
	qrName := "qr_code.png"
	qrFilePath := filepath.Join(tempDir, qrName)
	// Convertir en JSON
	data, err := json.Marshal(info)
	if err != nil {
		log.Printf("From QrCodeMaker -> can't convert into json : error : %v", err)
		return "", false
	}
	err = qrcode.WriteFile(string(data), qrcode.Highest, 425, qrFilePath)
	if err != nil {
		log.Printf("From QrCodeMaker -> can't create qr code : error : %v", err)
		return "", false
	}
	return qrName, true
}
