package worktool

import (
	"io"
	"log"
	"net/http"
	"strings"
)

func GetTester(baseURL, urlTested, contentExpected string) {

	resp, err := http.Get(baseURL + urlTested)
	if err != nil {
		log.Fatalf("GET "+urlTested+" failed: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("GET "+urlTested+" returned status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, contentExpected) {
		log.Println("GET " + urlTested + " does not contain expected content")
	} else {
		log.Println("GET " + urlTested + " succeeded and content is correct")
	}
}
