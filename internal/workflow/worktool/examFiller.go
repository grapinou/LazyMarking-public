package worktool

import (
	"log"
)

func ExamFiller(featureName, baseURL, urlTable, urlCrud string) {
	log.Println("---------")
	log.Printf("Testing : %s\n", featureName)

	log.Println("Testing post on exam")
	fields := map[string]string{
		"qcm_id":        "1",
		"class_code_id": "1",
		"period_id":     "1",
		"year_id":       "1",
		"exam":          "mvti",
	}
	PostTesterWF(baseURL, urlCrud, fields)
}
