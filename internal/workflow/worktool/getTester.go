package worktool

import (
	"io"
	"log"
	"net/http"
	"strings"
)

func GetTester(baseURL, urlTested, contentExpected string) {

	resp, err := Client.Get(baseURL + urlTested)
	if err != nil {
		log.Fatalf("❌ GET %s failed: %v", urlTested, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("❌ GET %s returned status %d", urlTested, resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, contentExpected) {
		log.Printf("❌ GET %s does not contain expected content\n", urlTested)
	} else {
		log.Printf("✅ GET %s succeeded and content is correct", urlTested)
	}
}
