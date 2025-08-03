package worktool

import (
	"fmt"
	"log"
)

func AltanswerFiller(baseURL, urlTable, urlCrud string, altAnswers []AltAnswerStructWf) {
	log.Println("---------")
	log.Println("Testting : Alt answer")

	log.Println("Testing post on form")

	for _, altAnswer := range altAnswers {
		urlCrudForm := urlCrud + "?question_id=" + altAnswer.QuestionID + "&alt_question_id=" + altAnswer.AltQuestionID
		fields := map[string]string{
			"question_id":     altAnswer.QuestionID,
			"alt_question_id": altAnswer.AltQuestionID,
			"state":           altAnswer.State,
			"content":         altAnswer.Content,
		}

		PostTesterWF(baseURL, urlCrudForm, fields)
		log.Println("Checking after post")
		urlTableParam := fmt.Sprintf(urlTable+"?question_id=%v&alt_question_id=%v", altAnswer.QuestionID, altAnswer.AltQuestionID)
		GetTester(baseURL, urlTableParam, altAnswer.Content)
	}
}
