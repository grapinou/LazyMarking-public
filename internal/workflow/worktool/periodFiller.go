package worktool

import (
	"log"
)

func PeriodFiller(featureName, baseURL, urlTable, urlCrud, fieldName, fieldValue string) {
	log.Println("---------")
	log.Printf("Testing : %s\n", featureName)

	log.Println("Testing post on period exam")
	fields := map[string]string{
		fieldName: fieldValue,
	}
	PostTesterWF(baseURL, urlCrud, fields)
}
