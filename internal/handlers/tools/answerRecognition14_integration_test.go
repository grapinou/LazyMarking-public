package tools

import (
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestRealTwoPageAnswerRecognition14(t *testing.T) {
	wantPageLengths := []int{20, 4}
	want := []int{
		1, 1, 0, 1,
		1, 0, 0, 1,
		0, 1, 0, 1,
		1, 0, 0, 0,
		1, 0, 0, 1,
		0, 0, 1, 1,
	}
	qcm, got := testRealMultiPageAnswerRecognition(t, "LAZYMARKING_TEST_PDF_2_PAGES", 14, wantPageLengths, want)
	if len(got) != 24 {
		t.Fatalf("recognized answer count = %d, want 24", len(got))
	}

	questionMarks := CountingPoints(qcm, got)
	if got, want := len(questionMarks), 6; got != want {
		t.Fatalf("question mark count = %d, want %d", got, want)
	}
	wantQuestionMarks := []config.QuestionMark{
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 2},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Partial, Score: 0.5, Total: 1},
		{State: config.Incorrect, Score: 0, Total: 1},
		{State: config.Partial, Score: 1, Total: 2},
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
	if score != 1.5 {
		t.Errorf("score = %v, want 1.5", score)
	}
	if total != 8 {
		t.Errorf("total = %d, want 8", total)
	}
}
