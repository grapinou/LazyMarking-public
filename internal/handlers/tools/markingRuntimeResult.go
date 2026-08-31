package tools

import (
	"fmt"
	"math"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

const halfPointConversionTolerance = 1e-9

func BuildMarkingCopyResult(
	studentExamID int64,
	expectedPages int,
	detectedPages int,
	qcm config.QCM,
	questionMarks []config.QuestionMark,
	detections []config.AnswerDetection,
) (config.MarkingCopyResult, error) {
	if len(questionMarks) != len(qcm.Questions) {
		return config.MarkingCopyResult{}, fmt.Errorf("question mark count is %d, want %d", len(questionMarks), len(qcm.Questions))
	}

	expectedDetections := 0
	for _, question := range qcm.Questions {
		expectedDetections += len(question.Answers)
	}
	if len(detections) != expectedDetections {
		return config.MarkingCopyResult{}, fmt.Errorf("answer detection count is %d, want %d", len(detections), expectedDetections)
	}

	result := config.MarkingCopyResult{
		StudentExamID: studentExamID,
		ExpectedPages: expectedPages,
		DetectedPages: detectedPages,
		Questions:     make([]config.MarkingQuestionResult, len(qcm.Questions)),
	}
	detectionOffset := 0
	for questionIndex, question := range qcm.Questions {
		answerCount := len(question.Answers)
		questionDetections := append([]config.AnswerDetection(nil), detections[detectionOffset:detectionOffset+answerCount]...)
		for answerIndex, detection := range questionDetections {
			classified, err := answerDetectionFromMean(detection.MeanGray)
			if err != nil {
				return config.MarkingCopyResult{}, fmt.Errorf("question %d answer %d: %w", questionIndex, answerIndex, err)
			}
			if detection.State != 0 && detection.State != 1 {
				return config.MarkingCopyResult{}, fmt.Errorf("question %d answer %d state is %d, want 0 or 1", questionIndex, answerIndex, detection.State)
			}
			if detection.State != classified.State {
				return config.MarkingCopyResult{}, fmt.Errorf("question %d answer %d state does not match mean gray", questionIndex, answerIndex)
			}
		}
		result.Questions[questionIndex] = config.MarkingQuestionResult{
			QuestionIndex:    questionIndex,
			Mark:             questionMarks[questionIndex],
			AnswerDetections: questionDetections,
		}
		detectionOffset += answerCount
	}
	result.Score, result.Total = CountingTotalPoint(questionMarks)
	return result, nil
}

func ScoreHalfUnits(score float64) (int64, error) {
	if math.IsNaN(score) || math.IsInf(score, 0) || score < 0 {
		return 0, fmt.Errorf("invalid score %v", score)
	}
	scaled := score * 2
	rounded := math.Round(scaled)
	if math.Abs(scaled-rounded) > halfPointConversionTolerance {
		return 0, fmt.Errorf("score %v is not representable in half-points", score)
	}
	if rounded >= float64(math.MaxInt64) {
		return 0, fmt.Errorf("score %v exceeds half-point range", score)
	}
	return int64(rounded), nil
}

func MarkingQuestionStateName(state config.QuestionState) (string, error) {
	switch state {
	case config.Incorrect:
		return "incorrect", nil
	case config.Partial:
		return "partial", nil
	case config.Correct:
		return "correct", nil
	default:
		return "", fmt.Errorf("unknown marking question state %d", state)
	}
}

func MarkingCopyResultToPersistedInput(userID, markingJobID int64, result config.MarkingCopyResult) (db.PersistedMarkingCopyInput, error) {
	scoreHalfUnits, err := ScoreHalfUnits(result.Score)
	if err != nil {
		return db.PersistedMarkingCopyInput{}, err
	}
	input := db.PersistedMarkingCopyInput{
		UserID:         userID,
		MarkingJobID:   markingJobID,
		StudentExamID:  result.StudentExamID,
		ExpectedPages:  int64(result.ExpectedPages),
		DetectedPages:  int64(result.DetectedPages),
		ScoreHalfUnits: scoreHalfUnits,
		TotalPoints:    int64(result.Total),
		Questions:      make([]db.PersistedQuestionInput, len(result.Questions)),
	}
	var questionScoreTotal int64
	var questionPointTotal int64
	for questionPosition, question := range result.Questions {
		if question.QuestionIndex != questionPosition {
			return db.PersistedMarkingCopyInput{}, fmt.Errorf("question index is %d at position %d", question.QuestionIndex, questionPosition)
		}
		state, err := MarkingQuestionStateName(question.Mark.State)
		if err != nil {
			return db.PersistedMarkingCopyInput{}, err
		}
		questionScoreHalfUnits, err := ScoreHalfUnits(question.Mark.Score)
		if err != nil {
			return db.PersistedMarkingCopyInput{}, fmt.Errorf("question %d: %w", question.QuestionIndex, err)
		}
		expectedScoreHalfUnits := int64(0)
		switch question.Mark.State {
		case config.Partial:
			expectedScoreHalfUnits = question.Mark.Total
		case config.Correct:
			expectedScoreHalfUnits = 2 * question.Mark.Total
		}
		if question.Mark.Total < 1 || questionScoreHalfUnits != expectedScoreHalfUnits {
			return db.PersistedMarkingCopyInput{}, fmt.Errorf("question %d score/state is incompatible with persistence contract", question.QuestionIndex)
		}
		questionScoreTotal += questionScoreHalfUnits
		questionPointTotal += question.Mark.Total
		persistedQuestion := db.PersistedQuestionInput{
			QuestionIndex:  int64(question.QuestionIndex),
			State:          state,
			ScoreHalfUnits: questionScoreHalfUnits,
			TotalPoints:    question.Mark.Total,
			Answers:        make([]db.PersistedAnswerDetectionInput, len(question.AnswerDetections)),
		}
		for answerIndex, detection := range question.AnswerDetections {
			classified, err := answerDetectionFromMean(detection.MeanGray)
			if err != nil {
				return db.PersistedMarkingCopyInput{}, fmt.Errorf("question %d answer %d: %w", question.QuestionIndex, answerIndex, err)
			}
			if detection.State != classified.State {
				return db.PersistedMarkingCopyInput{}, fmt.Errorf("question %d answer %d state does not match mean gray", question.QuestionIndex, answerIndex)
			}
			persistedQuestion.Answers[answerIndex] = db.PersistedAnswerDetectionInput{
				AnswerIndex:   int64(answerIndex),
				DetectedState: int64(detection.State),
				MeanGray:      detection.MeanGray,
			}
		}
		input.Questions[questionPosition] = persistedQuestion
	}
	if questionScoreTotal != input.ScoreHalfUnits || questionPointTotal != input.TotalPoints {
		return db.PersistedMarkingCopyInput{}, fmt.Errorf("copy totals do not match detailed questions")
	}
	return input, nil
}
