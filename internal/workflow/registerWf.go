package workflow

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func GetRegisterWF(baseURL string) {
	resp, err := http.Get(baseURL + data.DefaultHomeRoutes.RegisterURL)
	if err != nil {
		log.Fatalf("GET /register failed: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("GET /register returned status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "Créer un compte:") {
		log.Println("GET /register does not contain expected content")
	} else {
		log.Println("GET /register succeeded and content is correct")
	}
}

func PostRegisterWF(username, email, password, baseURL string) {
	form := url.Values{}
	form.Set("username", username)
	form.Set("email", email)
	form.Set("password", password)

	resp, err := http.Post(
		baseURL+data.DefaultHomeRoutes.RegisterURL,
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Registration failed: status %d", resp.StatusCode)
	}

	log.Println("POST /register : user created successfully (redirected to success)")
}
