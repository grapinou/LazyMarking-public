package tools

import (
	"context"
	"fmt"
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
	jobDBID int64,
) ([]config.MarkExam, []config.MarkExam, error) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // par ex. max 5 examens en parallèle
	var mu sync.Mutex

	var markExams []config.MarkExam
	var notMarkedExams []config.MarkExam

	var firstErr error
	errOnce := sync.Once{} // pour ne garder que la première erreur critique

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

			rows, err := queries.UpdateMarkingJobExamDone(ctx, db.UpdateMarkingJobExamDoneParams{
				ID:     jobDBID,
				UserID: userID,
			})
			if err != nil {
				log.Printf("From ProcessPagesConcurrently -> queries.UpdateMarkingJobExamDone DB error : %v", err)
				errOnce.Do(func() { firstErr = err }) // capture première erreur critique
				return
			}
			if rows != 1 {
				err := fmt.Errorf("UpdateMarkingJobExamDone affected %d rows for job %d", rows, jobDBID)
				log.Printf("From ProcessExamsConcurrently -> %v", err)
				errOnce.Do(func() { firstErr = err })
			}
		}()
	}

	wg.Wait()
	return markExams, notMarkedExams, firstErr
}
