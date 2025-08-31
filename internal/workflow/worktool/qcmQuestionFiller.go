package worktool

import (
	"log"
)

func QcmQuestionsFiller(featureName, baseURL, urlTable, urlCrud, qcmID, fieldName, fieldValue string) {
	log.Println("---------")
	log.Printf("Testing : %s\n", featureName)

	log.Println("Testing post on qcm questions")
	fields := map[string]string{
		fieldName: fieldValue,
	}
	urlParam := urlCrud + "?qcm_id=" + qcmID
	PostTesterWF(baseURL, urlParam, fields)
}
