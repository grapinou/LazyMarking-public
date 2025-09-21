package tools

import (
	"context"
	"log"
	"sync"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func ProcessExamsConcurrently(
	exams []config.Exam,
	userID int64,
	username string,
	tempDir string,
	ctx context.Context,
	queries *db.Queries,
) ([]config.MarkExam, []config.MarkExam) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // par ex. max 5 examens en parallèle
	var mu sync.Mutex

	var markExams []config.MarkExam
	var notMarkedExams []config.MarkExam

	for _, exam := range exams {
		wg.Add(1)
		exam := exam // éviter la closure

		go func() {
			defer wg.Done()

			sem <- struct{}{} // prend un ticket
			defer func() { <-sem }()

			markExam, err := MarkingStudentExam(userID, username, tempDir, exam, ctx, queries)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				log.Printf("Error with MarkingStudentExam: %v", err)
				notMarkedExams = append(notMarkedExams, markExam)
				return
			}

			if markExam.Status {
				markExams = append(markExams, markExam)
			}
		}()
	}

	wg.Wait()
	return markExams, notMarkedExams
}
