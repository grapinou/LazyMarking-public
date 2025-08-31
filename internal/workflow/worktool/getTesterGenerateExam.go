package worktool

import (
	"log"
	"net/http"
)

func GetTesterGenerateExam(baseURL, urlTested, examID string) {
	fullURL := baseURL + urlTested + "?exam_id=" + examID
	resp, err := Client.Get(fullURL)
	if err != nil {
		log.Fatalf("❌ GET %s failed: %v", urlTested, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("❌ GET %s returned status %d", urlTested, resp.StatusCode)
	} else {
		log.Printf("✅ GET %s succeeded to generate exam", urlTested)
	}
}
