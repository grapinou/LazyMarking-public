package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/phpdave11/gofpdf"
)

func ConvertPngTopdf(tempDir, imgName string) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")

	pdf.AddPage()

	// Options pour l'image
	opts := gofpdf.ImageOptions{
		ImageType: "PNG", // Type de l'image ("PNG", "JPG", "GIF")
		ReadDpi:   true,  // Lit la résolution (DPI) du fichier si disponible
	}

	imgPath := filepath.Join(tempDir, imgName)

	// Enregistrer l'image et récupérer ses infos
	info := pdf.RegisterImageOptions(imgPath, opts)
	if pdf.Error() != nil || info == nil {
		return "", fmt.Errorf("load PNG for PDF: %w", pdf.Error())
	}

	info.SetDpi(300)
	wMM, hMM := info.Extent() // recalcul
	// Insère l'image avec les options
	// filename, x, y, width, height, flow, options, link, linkStr
	pdf.ImageOptions(imgPath, 0, 0, wMM, hMM, false, opts, 0, "") // 210 ok

	// Sauvegarde
	pdfName := strings.TrimSuffix(imgName, filepath.Ext(imgName))
	pdfName += ".pdf"
	pdfPath := filepath.Join(tempDir, pdfName)
	err := pdf.OutputFileAndClose(pdfPath)
	if err != nil {
		return "", fmt.Errorf("write PDF: %w", err)
	}

	return pdfName, nil
}
