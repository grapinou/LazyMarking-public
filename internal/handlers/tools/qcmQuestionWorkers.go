package tools

import (
	"sync"

	"github.com/grapinou/LazyMarking/internal/config"
)

type qcmQuestionJob struct {
	Index      int
	QuestionID int64
}

type qcmQuestionResult struct {
	Index    int
	Question config.Question
	Err      error
}

func buildQCMQuestionsInOrder(questionIDs []int64, build func(int64) (config.Question, error)) ([]config.Question, error) {
	questions := make([]config.Question, len(questionIDs))
	jobs := make(chan qcmQuestionJob, len(questionIDs))
	results := make(chan qcmQuestionResult, len(questionIDs))

	const numWorkers = 5
	var workers sync.WaitGroup
	for range numWorkers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobs {
				question, err := build(job.QuestionID)
				results <- qcmQuestionResult{Index: job.Index, Question: question, Err: err}
			}
		}()
	}

	for index, questionID := range questionIDs {
		jobs <- qcmQuestionJob{Index: index, QuestionID: questionID}
	}
	close(jobs)

	workers.Wait()
	close(results)

	var firstErr error
	for result := range results {
		if result.Err != nil {
			if firstErr == nil {
				firstErr = result.Err
			}
			continue
		}
		questions[result.Index] = result.Question
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return questions, nil
}
