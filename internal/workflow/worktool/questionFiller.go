package worktool

import (
	"log"
	"math/rand"
	"strconv"
	"time"
)

// Création d’un générateur local, basé sur une source aléatoire
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandomID(min, max int) string {
	n := rnd.Intn(max-min+1) + min
	return strconv.Itoa(n)
}

func QuestionFiller(baseURL, urlTable, urlCrud, contentTableExpected, contentFormExpected, contentQuestion string, maxField int) {

	log.Println("---------")
	log.Println("Testing : Question")

	log.Println("Checking table")
	GetTester(baseURL, urlTable, contentTableExpected)
	log.Println("Checking form")
	GetTester(baseURL, urlCrud, contentFormExpected)

	log.Println("Testing post on form")
	fields := map[string]string{
		"subjectID":    RandomID(1, maxField),
		"themeID":      RandomID(1, maxField),
		"yearLevelID":  RandomID(1, maxField),
		"skillID":      RandomID(1, maxField),
		"difficultyID": RandomID(1, maxField),
		"pointID":      RandomID(1, maxField),
		"content":      contentQuestion,
	}
	PostTesterWF(baseURL, urlCrud, fields)

	log.Println("Checking after post")
	GetTester(baseURL, urlTable, contentQuestion)
}
