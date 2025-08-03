package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func AltAnswerWf(baseURL string) {
	altAnswers := []worktool.AltAnswerStructWf{
		{QuestionID: "1", AltQuestionID: "1", State: "0", Content: "bleu"},
		{QuestionID: "1", AltQuestionID: "1", State: "0", Content: "vert"},
		{QuestionID: "1", AltQuestionID: "1", State: "0", Content: "turquoise"},
		{QuestionID: "1", AltQuestionID: "1", State: "1", Content: "incolore"},
		{QuestionID: "1", AltQuestionID: "2", State: "0", Content: "rouge"},
		{QuestionID: "1", AltQuestionID: "2", State: "0", Content: "violet"},
		{QuestionID: "1", AltQuestionID: "2", State: "1", Content: "blanc"},
		{QuestionID: "1", AltQuestionID: "3", State: "0", Content: "magenta"},
		{QuestionID: "1", AltQuestionID: "3", State: "0", Content: "cyan"},
		{QuestionID: "1", AltQuestionID: "3", State: "0", Content: "jaune"},
		{QuestionID: "1", AltQuestionID: "3", State: "1", Content: "verte"},
		{QuestionID: "2", AltQuestionID: "4", State: "0", Content: "3 kg"},
		{QuestionID: "2", AltQuestionID: "4", State: "0", Content: "1000 kg"},
		{QuestionID: "2", AltQuestionID: "4", State: "0", Content: "1 000 000 000 kg"},
		{QuestionID: "2", AltQuestionID: "4", State: "1", Content: "10 100 000  kg"},
		{QuestionID: "2", AltQuestionID: "5", State: "0", Content: "5900 kg"},
		{QuestionID: "2", AltQuestionID: "5", State: "0", Content: "5 900 000 kg"},
		{QuestionID: "2", AltQuestionID: "5", State: "0", Content: "590 000 000 000 kg"},
		{QuestionID: "2", AltQuestionID: "5", State: "1", Content: "5,972 × 10^24 kg"},
		{QuestionID: "2", AltQuestionID: "6", State: "0", Content: "500 kg"},
		{QuestionID: "2", AltQuestionID: "6", State: "0", Content: "500 000 kg"},
		{QuestionID: "2", AltQuestionID: "6", State: "0", Content: "5 000 000 kg"},
		{QuestionID: "2", AltQuestionID: "6", State: "1", Content: "500 000 000 kg"},
		{QuestionID: "3", AltQuestionID: "7", State: "0", Content: "rond x rond"},
		{QuestionID: "3", AltQuestionID: "7", State: "0", Content: "rond x hauteur"},
		{QuestionID: "3", AltQuestionID: "7", State: "0", Content: "rond x rond x rond"},
		{QuestionID: "3", AltQuestionID: "7", State: "1", Content: "pi x r²"},
		{QuestionID: "3", AltQuestionID: "8", State: "0", Content: "triangle x triangle"},
		{QuestionID: "3", AltQuestionID: "8", State: "0", Content: "triangle x côté"},
		{QuestionID: "3", AltQuestionID: "8", State: "0", Content: "triangle x triangle x triangle"},
		{QuestionID: "3", AltQuestionID: "8", State: "1", Content: "un demi x base x hauteur"},
		{QuestionID: "3", AltQuestionID: "9", State: "0", Content: "rectangle x rectangle x rectangle"},
		{QuestionID: "3", AltQuestionID: "9", State: "0", Content: "rectangle x rectangle"},
		{QuestionID: "3", AltQuestionID: "9", State: "1", Content: "largeur x longueur"},
	}

	worktool.AltanswerFiller(baseURL,
		data.DefaultAltQuestionRoutes.AltAnswersURL,
		data.DefaultAltAnswerRoutes.AddURL,
		altAnswers,
	)
}
