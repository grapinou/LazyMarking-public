package tools

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
)

func LeftPages(tempPath string, name db.GetExamAndMarkNameRow) (string, error) {
	files, err := GetAllFiles(tempPath, "*.png")
	if err != nil {
		log.Printf("From LeftPages -> error getting all png files")
		return "", err
	}

	if len(files) == 0 {
		return "", nil
	}

	var pdfNames []string
	for _, file := range files {
		dir, imgName := filepath.Split(file)
		pdf, err := ConvertPngTopdf(tempPath, imgName)
		if err != nil {
			return "", err
		}
		pdfNames = append(pdfNames, filepath.Join(dir, pdf))
	}

	pdfName := name.ExamName.String
	pdfName = strings.TrimSuffix(filepath.Base(pdfName), filepath.Ext(pdfName))
	pdfName = pdfName + "_NOT.pdf"
	pdfMerge := filepath.Join(tempPath, pdfName)

	if err := MergePdf(pdfNames, pdfMerge); err != nil {
		log.Println("From LeftPages -> can't merge pdf")
		return "", err
	}

	if err := RemoveFiles(pdfNames); err != nil {
		log.Printf("From LeftPages -> RemoveFiles return error : %v", err)
		return "", err
	}

	if err := RemoveFiles(files); err != nil {
		log.Printf("From LeftPages -> RemoveFiles return error : %v", err)
		return "", err
	}

	return pdfName, nil
}
