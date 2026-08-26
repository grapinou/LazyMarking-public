package tools

import (
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestRealThreePageAnswerRecognition387(t *testing.T) {
	wantPageLengths := []int{18, 16, 12}
	want := []int{
		0, 0, 0, 1,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
		0, 1,

		0, 0, 0, 0,
		0, 1, 1, 0,
		0, 0, 1, 0,
		1, 0, 1, 0,

		0, 1, 0, 0,
		0, 0, 0, 1,
		0, 1, 0, 0,
	}
	qcm, got := testRealMultiPageAnswerRecognition(t, "LAZYMARKING_TEST_PDF", 387, wantPageLengths, want)
	if len(got) != 46 {
		t.Fatalf("recognized answer count = %d, want 46", len(got))
	}

	questionMarks := CountingPoints(qcm, got)
	if got, want := len(questionMarks), 13; got != want {
		t.Fatalf("question mark count = %d, want %d", got, want)
	}
	wantQuestionMarks := []config.QuestionMark{
		{State: config.Partial, Score: 0.5, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Correct, Score: 1, Total: 1},
		{State: config.Partial, Score: 1, Total: 2},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Correct, Score: 1, Total: 1},
		{State: config.Partial, Score: 1, Total: 2},
		{State: config.Correct, Score: 1, Total: 1},
		{State: config.Correct, Score: 1, Total: 1},
	}
	for i := range wantQuestionMarks {
		if questionMarks[i].State != wantQuestionMarks[i].State ||
			questionMarks[i].Score != wantQuestionMarks[i].Score ||
			questionMarks[i].Total != wantQuestionMarks[i].Total {
			t.Errorf(
				"Q%d: got state=%d score=%v total=%d, want state=%d score=%v total=%d",
				i+1,
				questionMarks[i].State,
				questionMarks[i].Score,
				questionMarks[i].Total,
				wantQuestionMarks[i].State,
				wantQuestionMarks[i].Score,
				wantQuestionMarks[i].Total,
			)
		}
	}

	score, total := CountingTotalPoint(questionMarks)
	if score != 6.5 {
		t.Errorf("score = %v, want 6.5", score)
	}
	if total != 15 {
		t.Errorf("total = %d, want 15", total)
	}
}
