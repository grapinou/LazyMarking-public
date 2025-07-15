package config

import (
	"log"
	"os"
)

func GetBaseURL() string {
	url := os.Getenv("APP_BASE_URL")
	log.Println(url)
	if url == "" {
		log.Println("empty url, using default")
		return "http://localhost:8080"
	}
	return url
}
