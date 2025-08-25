package tools

import (
	"net/http"
	"sync"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func GetQCMQuestionsAnswers(userID, qcmID int64, r *http.Request, queries *db.Queries) ([]config.Question, error) {
	var qcmQuestions []config.Question
	questionsIDs, err := queries.GetQCMQuestionsIDs(r.Context(), db.GetQCMQuestionsIDsParams{
		UserID: userID,
		QcmID:  qcmID,
	})
	if err != nil {
		return qcmQuestions, err
	}
	ShuffleSlice(questionsIDs)

	jobs := make(chan int64, len(questionsIDs))
	results := make(chan config.Question, len(questionsIDs))
	errs := make(chan error, len(questionsIDs))

	const numWorkers = 5
	var wg sync.WaitGroup

	// Workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for questionID := range jobs {
				//  Limite globale DB
				config.DBSemaphore <- struct{}{} // prendre un ticket
				question, err := BuildQuestion(questionID, userID, r, queries)
				<-config.DBSemaphore // libérer le ticket
				if err != nil {
					errs <- err
					return
				}
				results <- question
			}
		}()
	}

	// Jobs
	for _, qID := range questionsIDs {
		jobs <- qID
	}
	close(jobs)

	wg.Wait()
	close(results)
	close(errs)

	if len(errs) > 0 {
		return nil, <-errs
	}

	for question := range results {
		qcmQuestions = append(qcmQuestions, question)
	}

	return qcmQuestions, nil
}
