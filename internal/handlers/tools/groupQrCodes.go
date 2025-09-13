package tools

import "github.com/grapinou/LazyMarking/internal/config"

func GroupQrCodes(qrDatas []config.QrCodeInfo) []config.Exam {
	m := make(map[int64][]config.Page)

	for _, qr := range qrDatas {
		page := config.Page{
			Number: qr.PageExam,
			Name:   qr.PageName, // sera vide à la génération
		}
		m[qr.StudentExamID] = append(m[qr.StudentExamID], page)
	}

	exams := make([]config.Exam, 0, len(m))
	for studentID, pages := range m {
		exams = append(exams, config.Exam{
			StudentExamID: studentID,
			Pages:         pages,
		})
	}

	return exams
}
