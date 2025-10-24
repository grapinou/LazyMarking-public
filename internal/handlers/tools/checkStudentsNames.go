package tools

import (
	"fmt"
	"strings"
	"sync"

	"github.com/grapinou/LazyMarking/internal/db"
)

func CheckStudentsNames(students []db.Student) []string {
	var invalidNames []string
	var mu sync.Mutex     // pour protéger invalidNames
	var wg sync.WaitGroup // pour attendre la fin des goroutines

	sem := make(chan struct{}, 5) // sémaphore de concurrence

	for _, student := range students {
		wg.Add(1)
		sem <- struct{}{} // bloque si trop de goroutines actives

		go func(s db.Student) {
			defer wg.Done()
			defer func() { <-sem }() // libère une place dans le sémaphore

			// Vérifie les noms
			if strings.ContainsRune(s.FirstName, '"') || strings.ContainsRune(s.LastName, '"') {
				name := fmt.Sprintf("%s %s", s.FirstName, s.LastName)
				mu.Lock()
				invalidNames = append(invalidNames, name)
				mu.Unlock()
			}
		}(student)
	}

	wg.Wait()
	return invalidNames
}
