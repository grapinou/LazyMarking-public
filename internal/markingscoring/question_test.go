package markingscoring

import (
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestScoreQuestionTransitions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		expected []int
		actual   []int
		state    config.QuestionState
		score    float64
	}{
		{name: "correct", expected: []int{1, 1}, actual: []int{1, 1}, state: config.Correct, score: 2},
		{name: "correct to partial", expected: []int{1, 1}, actual: []int{1, 0}, state: config.Partial, score: 1},
		{name: "partial to incorrect", expected: []int{1, 1}, actual: []int{0, 0}, state: config.Incorrect, score: 0},
		{name: "incorrect to correct", expected: []int{1}, actual: []int{1}, state: config.Correct, score: 2},
		{name: "wrong selection", expected: []int{1, 0}, actual: []int{1, 1}, state: config.Incorrect, score: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mark := ScoreQuestion(tc.expected, tc.actual, 2)
			if mark.State != tc.state || mark.Score != tc.score || mark.Total != 2 {
				t.Fatalf("mark=%+v", mark)
			}
		})
	}
}
