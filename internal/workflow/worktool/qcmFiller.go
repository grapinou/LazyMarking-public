package worktool

import (
	"log"
)

func QcmFiller(featureName, baseURL, urlTable, urlCrud, fieldName, fieldValue string) {
	log.Println("---------")
	log.Printf("Testing : %s\n", featureName)

	log.Println("Testing post on qcm")
	fields := map[string]string{
		fieldName: fieldValue,
	}
	PostTesterWF(baseURL, urlCrud, fields)
}
