package tools

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
)

func LeftPages(username string, name db.GetExamAndMarkNameRow) string {
	tempPath := filepath.Join("assets", "tmp", username)
	files, err := GetAllFiles(tempPath, "*.png")
	if err != nil {
		log.Printf("From LeftPages -> error getting all png files")
		return ""
	}

	if len(files) == 0 {
		return ""
	}

	var pdfNames []string
	for _, file := range files {
		dir, imgName := filepath.Split(file)
		pdf := ConvertPngTopdf(tempPath, imgName)
		pdfNames = append(pdfNames, filepath.Join(dir, pdf))
	}

	pdfName := name.ExamName.String
	pdfName = strings.TrimSuffix(filepath.Base(pdfName), filepath.Ext(pdfName))
	pdfName = pdfName + "_NOT.pdf"
	pdfMerge := filepath.Join(tempPath, pdfName)

	if err := MergePdf(pdfNames, pdfMerge); err != nil {
		log.Println("From LeftPages -> can't merge pdf")
		return ""
	}

	if err := RemoveFiles(pdfNames); err != nil {
		log.Printf("From LeftPages -> RemoveFiles return error : %v", err)
		return ""
	}

	if err := RemoveFiles(files); err != nil {
		log.Printf("From LeftPages -> RemoveFiles return error : %v", err)
		return ""
	}

	return pdfName
}
