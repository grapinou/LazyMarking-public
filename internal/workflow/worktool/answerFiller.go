package worktool

import (
	"fmt"
	"log"
)

func AnswerFiller(baseURL, urlTable, urlCrud, contentTableExpected, contentFormExpected string, maxQuestion int, answers []AnswerStructWf) {

	log.Println("---------")
	log.Println("Testting : Answer")

	for i := 1; i <= maxQuestion; i++ {
		log.Printf("Checking table for question_id : %d\n", i)
		urlTableParam := fmt.Sprintf(urlTable+"?question_id=%d", i)
		GetTester(baseURL, urlTableParam, contentTableExpected)
		log.Printf("Checking form for question_id : %d\n", i)
		urlCrudParam := fmt.Sprintf(urlCrud+"?question_id=%d", i)
		GetTester(baseURL, urlCrudParam, contentFormExpected)

	}

	log.Println("Testing post on form")

	for _, answer := range answers {
		urlCrudForm := urlCrud + "?question_id=" + answer.QuestionID
		fields := map[string]string{
			"question_id": answer.QuestionID,
			"state":       answer.State,
			"content":     answer.Content,
		}

		PostTesterWF(baseURL, urlCrudForm, fields)
		log.Println("Checking after post")
		urlTableParam := fmt.Sprintf(urlTable+"?question_id=%v", answer.QuestionID)
		GetTester(baseURL, urlTableParam, answer.Content)
	}
}
