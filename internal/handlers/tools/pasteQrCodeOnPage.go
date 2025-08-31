package tools

import (
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
)

func PasteQrCodeOnPage(tempDir, qrName, imgName string) (string, bool) {
	qrPath := filepath.Join(tempDir, qrName)
	imgPath := filepath.Join(tempDir, imgName)
	newImgName := "qr_" + imgName
	newImgPath := filepath.Join(tempDir, newImgName)

	// --- 1. Charger l'image de fond ---
	baseF, err := os.Open(imgPath)
	if err != nil {
		log.Printf("From PasteQrCodeOnPage error : %v", err)
		return "", false
	}
	defer baseF.Close()
	baseImg, err := png.Decode(baseF)
	if err != nil {
		log.Printf("From PasteQrCodeOnPage error : %v", err)
		return "", false
	}

	// --- 2. Charger l'image à coller ---
	overF, err := os.Open(qrPath)
	if err != nil {
		log.Printf("From PasteQrCodeOnPage error : %v", err)
		return "", false
	}
	defer overF.Close()
	overImg, err := png.Decode(overF)
	if err != nil {
		log.Printf("From PasteQrCodeOnPage error : %v", err)
		return "", false
	}

	// --- 3. Créer un canevas modifiable (RGBA) ---
	canevas := image.NewRGBA(baseImg.Bounds())

	// --- 4. Copier le fond dans le canevas ---
	draw.Draw(canevas, baseImg.Bounds(), baseImg, image.Point{}, draw.Src)

	// --- 5. Définir la source et la destination ---
	// sr = tout le rectangle source (le QR complet)
	// ici on rogne de crop px en haut/gauche et crop px en bas/droite
	crop := 25
	sr := image.Rect(crop, crop, overImg.Bounds().Dx()-crop, overImg.Bounds().Dy()-crop)

	// dp = le point de destination (où placer sr.Min)
	dp := image.Pt(127, 30)

	// r = rectangle de destination (dp → dp+sr.Size)
	r := image.Rectangle{Min: dp, Max: dp.Add(sr.Size())}

	// --- 6. Coller l'image overlay sur le canevas ---
	draw.Draw(canevas, r, overImg, sr.Min, draw.Over)

	// --- 7. Sauvegarder en PNG ---
	outF, err := os.Create(newImgPath)
	if err != nil {
		log.Printf("From PasteQrCodeOnPage error : %v", err)
		return "", false
	}
	defer outF.Close()
	if err := png.Encode(outF, canevas); err != nil {
		log.Printf("From PasteQrCodeOnPage error : %v", err)
		return "", false
	}

	return newImgName, true
}
