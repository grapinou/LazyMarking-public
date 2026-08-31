package tools

import (
	"context"
	"errors"
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
	expectedPages map[int64]int64,
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
			defer func() {
				if recovered := recover(); recovered != nil {
					err := fmt.Errorf("exam worker panic: %v", recovered)
					log.Printf("From ProcessExamsConcurrently -> %v", err)
					errOnce.Do(func() { firstErr = err })
				}
			}()

			sem <- struct{}{} // prend un ticket
			defer func() { <-sem }()

			markExam, markErr := MarkingStudentExam(userID, username, tempDir, exam, ctx, queries)

			if markErr != nil {
				log.Printf("Error with MarkingStudentExam: %v", markErr)
				if errors.Is(markErr, ErrMarkingHistoricalReference) {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("copy %d historical reference integrity: %w", exam.StudentExamID, markErr)
					})
					return
				}
				outcome, detectedPages := terminalOutcomeForMarkingError(exam, expectedPages[exam.StudentExamID])
				_, persistErr := db.PersistTerminalMarkingCopy(ctx, queries, db.PersistedTerminalMarkingCopyInput{
					UserID: userID, MarkingJobID: jobDBID, StudentExamID: exam.StudentExamID,
					Outcome: outcome, ExpectedPages: expectedPages[exam.StudentExamID], DetectedPages: detectedPages,
					FailureCode: map[string]string{"incomplete": "page_set_incomplete", "error": "copy_processing_error"}[outcome],
				})
				mu.Lock()
				notMarkedExams = append(notMarkedExams, markExam)
				mu.Unlock()
				if persistErr != nil {
					errOnce.Do(func() { firstErr = fmt.Errorf("persist terminal copy %d: %w", exam.StudentExamID, persistErr) })
				}
				return
			}

			if !markExam.Status || markExam.DetailedResult == nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("copy %d has no corrected detailed result", exam.StudentExamID) })
				return
			}
			input, err := MarkingCopyResultToPersistedInput(userID, jobDBID, *markExam.DetailedResult)
			if err != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("adapt corrected copy %d: %w", exam.StudentExamID, err) })
				return
			}
			if _, err = db.PersistCorrectedMarkingCopyWithQueries(ctx, queries, input); err != nil {
				errOnce.Do(func() { firstErr = fmt.Errorf("persist corrected copy %d: %w", exam.StudentExamID, err) })
				return
			}
			mu.Lock()
			markExams = append(markExams, markExam)
			mu.Unlock()

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

func terminalOutcomeForMarkingError(exam config.Exam, expectedPages int64) (string, int64) {
	seen := make(map[int]struct{}, len(exam.Pages))
	for _, page := range exam.Pages {
		if page.Number >= 1 && int64(page.Number) <= expectedPages {
			seen[page.Number] = struct{}{}
		}
	}
	detected := int64(len(seen))
	if detected != expectedPages || len(exam.Pages) != len(seen) {
		return "incomplete", detected
	}
	return "error", detected
}
