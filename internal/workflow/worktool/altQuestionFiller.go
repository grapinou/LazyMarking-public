package worktool

import (
	"fmt"
	"log"
)

func AltQuestionFiller(baseURL, urlTable, urlCrud, contentTableExpected, contentFormExpected string, maxQuestion int, altQuestions []AltQuestionStructWf) {
	log.Println("---------")
	log.Println("Testting : Alt question")

	for i := 1; i <= maxQuestion; i++ {
		log.Printf("Checking table for question_id : %d\n", i)
		urlTableParam := fmt.Sprintf(urlTable+"?question_id=%d", i)
		GetTester(baseURL, urlTableParam, contentTableExpected)
		log.Printf("Checking form for question_id : %d\n", i)
		urlCrudParam := fmt.Sprintf(urlCrud+"?question_id=%d", i)
		GetTester(baseURL, urlCrudParam, contentFormExpected)

	}

	log.Println("Testing post on form")

	for _, altQuestion := range altQuestions {
		urlCrudForm := urlCrud + "?question_id=" + altQuestion.QuestionID
		fields := map[string]string{
			"question_id": altQuestion.QuestionID,
			"content":     altQuestion.Content,
		}

		PostTesterWF(baseURL, urlCrudForm, fields)
		log.Println("Checking after post")
		urlTableParam := fmt.Sprintf(urlTable+"?question_id=%v", altQuestion.QuestionID)
		GetTester(baseURL, urlTableParam, altQuestion.Content)
	}
}
