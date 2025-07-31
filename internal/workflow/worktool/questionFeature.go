package worktool

import (
	"log"
)

func QuestionFeature(featureName, baseURL, urlTable, urlCrud, fieldName, fieldValue, contentTableExpected, contentFormExpected string) {

	log.Println("---------")
	log.Printf("Testing : %s\n", featureName)

	log.Println("Checking table")
	GetTester(baseURL, urlTable, contentTableExpected)
	log.Println("Checking form")
	GetTester(baseURL, urlCrud, contentFormExpected)

	log.Println("Testing post on form")
	fields := map[string]string{
		fieldName: fieldValue,
	}
	PostTesterWF(baseURL, urlCrud, fields)

	log.Println("Checking after post")
	GetTester(baseURL, urlTable, fieldValue)
}
