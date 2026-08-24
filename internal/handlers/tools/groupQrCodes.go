package tools

import (
	"slices"

	"github.com/grapinou/LazyMarking/internal/config"
)

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
		slices.SortFunc(pages, func(a, b config.Page) int {
			if a.Number != b.Number {
				return a.Number - b.Number
			}
			if a.Name < b.Name {
				return -1
			}
			if a.Name > b.Name {
				return 1
			}
			return 0
		})
		exams = append(exams, config.Exam{
			StudentExamID: studentID,
			Pages:         pages,
		})
	}
	slices.SortFunc(exams, func(a, b config.Exam) int {
		if a.StudentExamID < b.StudentExamID {
			return -1
		}
		if a.StudentExamID > b.StudentExamID {
			return 1
		}
		return 0
	})

	return exams
}
