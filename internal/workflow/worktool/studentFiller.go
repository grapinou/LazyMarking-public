package worktool

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
)

// Création d’un générateur local, basé sur une source aléatoire
var rndsrc = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandomClassCode(min, max int) string {
	n := rndsrc.Intn(max-min+1) + min
	return strconv.Itoa(n)
}

func StudentFiller(baseURL, urlTable, urlCrud string, maxClassCode int) {
	log.Println("---------")
	log.Println("Testing : Student")

	students := map[string]string{
		"Alice":  "Dupont",
		"Jean":   "Martin",
		"Sophie": "Leroy",
		"Lucas":  "Bernard",

		"Emma":    "Moreau",
		"Hugo":    "Petit",
		"Chloé":   "Robert",
		"Louis":   "Garcia",
		"Léa":     "Richard",
		"Maxime":  "Michel",
		"Camille": "Thomas",
		"Nathan":  "Roux",
		"Manon":   "Fontaine",
		"Tom":     "Giraud",
		"Sarah":   "Carpentier",
	}

	log.Println("Testing post on form")

	for firstName, lastName := range students {

		fields := map[string]string{
			"class_code_id": RandomClassCode(1, maxClassCode),
			"first_name":    firstName,
			"last_name":     lastName,
		}
		PostTesterWF(baseURL, urlCrud, fields)

		log.Println("Checking after post")
		GetTester(baseURL, urlTable, fmt.Sprintf("%s %s", firstName, lastName))
	}
}
