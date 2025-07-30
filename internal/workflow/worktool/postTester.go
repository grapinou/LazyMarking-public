package worktool

import (
	"log"
	"net/http"
	"net/url"
	"strings"
)

func PostRegisterWF(baseURL, urlTested string, fields map[string]string) {
	form := url.Values{}
	for key, value := range fields {
		form.Set(key, value)
	}

	resp, err := http.Post(
		baseURL+urlTested,
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		log.Fatalf("Failed to send request to : POST "+urlTested+": %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("POST "+urlTested+" failed: status %d", resp.StatusCode)
	}

	log.Println("POST " + urlTested + " : redirected with success")
}
