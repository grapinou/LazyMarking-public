package tools

import (
	"encoding/json"
	"log"
	"path/filepath"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/skip2/go-qrcode"
)

func QrCodeMaker(tempDir, pageName string, info config.QrCodeInfo) (string, bool) {
	name := strings.TrimSuffix(pageName, filepath.Ext(pageName))

	qrName := "qr_code_" + name + ".png"
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
