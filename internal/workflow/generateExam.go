package workflow

import (
	"log"

	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func GenerateExamWf(baseURL string) {
	log.Println("---------")
	log.Println("Testing get on generate exam")

	worktool.GetTesterGenerateExam(
		baseURL,
		data.DefaultExamRoutes.GenerateExamPdf,
		"1",
	)
}
