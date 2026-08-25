package tools

import (
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestRealOnePageAnswerRecognition(t *testing.T) {
	want := []int{
		0, 0, 1, 0,
		0, 1, 0, 0,
		1, 0, 0, 0,
		1, 0, 0, 0,
	}
	qcm, got := testRealOnePageAnswerRecognition(t, 781, want)

	questionMarks := CountingPoints(qcm, got)
	if got, want := len(questionMarks), 4; got != want {
		t.Fatalf("question mark count = %d, want %d", got, want)
	}
	wantStates := []config.QuestionState{
		config.Incorrect,
		config.Correct,
		config.Incorrect,
		config.Correct,
	}
	wantScores := []float64{0, 1, 0, 1}
	for i := range questionMarks {
		if questionMarks[i].State != wantStates[i] {
			t.Errorf("Q%d state = %d, want %d", i+1, questionMarks[i].State, wantStates[i])
		}
		if questionMarks[i].Score != wantScores[i] {
			t.Errorf("Q%d score = %v, want %v", i+1, questionMarks[i].Score, wantScores[i])
		}
	}

	score, total := CountingTotalPoint(questionMarks)
	if score != 2 {
		t.Errorf("score = %v, want 2", score)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
}
