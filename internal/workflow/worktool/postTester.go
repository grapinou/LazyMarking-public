package worktool

import (
	"log"
	"net/http"
	"net/url"
	"strings"
)

func PostTesterWF(baseURL, urlTested string, fields map[string]string) {
	form := url.Values{}
	for key, value := range fields {
		form.Set(key, value)
	}

	req, err := http.NewRequest("POST", baseURL+urlTested,
		strings.NewReader(form.Encode()))
	if err != nil {
		log.Fatalf("❌ Failed to create POST request %s: %v", urlTested, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := Client.Do(req)
	if err != nil {
		log.Fatalf("POST %s failed: %v", urlTested, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && (resp.StatusCode < 300 || resp.StatusCode > 399) {
		log.Fatalf("❌ POST %s failed: status %d", urlTested, resp.StatusCode)
	}

	log.Printf("✅ POST %s : success (status %d, redirected to %s)\n",
		urlTested, resp.StatusCode, resp.Request.URL.Path)
}
