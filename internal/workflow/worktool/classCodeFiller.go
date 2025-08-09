package worktool

import (
	"log"
)

func ClassCodeFiller(featureName, baseURL, urlTable, urlCrud, fieldName, fieldValue string) {

	log.Println("---------")
	log.Printf("Testing : %s\n", featureName)

	log.Println("Testing post on form")
	fields := map[string]string{
		fieldName: fieldValue,
	}
	PostTesterWF(baseURL, urlCrud, fields)

	log.Println("Checking after post")
	GetTester(baseURL, urlTable, fieldValue)
}
