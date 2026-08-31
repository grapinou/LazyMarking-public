package tools

import (
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func testQuestion(points int64, expected ...int64) config.Question {
	answers := make([]config.Answer, len(expected))
	for i, state := range expected {
		answers[i].State = state
	}
	return config.Question{
		Tags:    config.Tags{Point: config.Point{PointValue: points}},
		Answers: answers,
	}
}

func TestCountingPoints(t *testing.T) {
	tests := []struct {
		name    string
		qcm     config.QCM
		answers []int
		want    []config.QuestionMark
	}{
		{
			name:    "exact answer",
			qcm:     config.QCM{Questions: []config.Question{testQuestion(3, 1, 0, 1)}},
			answers: []int{1, 0, 1},
			want:    []config.QuestionMark{{Score: 3, Total: 3, State: config.Correct}},
		},
		{
			name:    "odd partial score",
			qcm:     config.QCM{Questions: []config.Question{testQuestion(3, 1, 0, 1)}},
			answers: []int{1, 0, 0},
			want:    []config.QuestionMark{{Score: 1.5, Total: 3, State: config.Partial}},
		},
		{
			name:    "wrong selection cancels partial",
			qcm:     config.QCM{Questions: []config.Question{testQuestion(4, 1, 0, 1)}},
			answers: []int{1, 1, 0},
			want:    []config.QuestionMark{{Score: 0, Total: 4, State: config.Incorrect}},
		},
		{
			name:    "no answer is incorrect",
			qcm:     config.QCM{Questions: []config.Question{testQuestion(2, 1, 0)}},
			answers: []int{0, 0},
			want:    []config.QuestionMark{{Score: 0, Total: 2, State: config.Incorrect}},
		},
		{
			name:    "over-selection is incorrect",
			qcm:     config.QCM{Questions: []config.Question{testQuestion(2, 1, 1, 0)}},
			answers: []int{1, 1, 1},
			want:    []config.QuestionMark{{Score: 0, Total: 2, State: config.Incorrect}},
		},
		{
			name: "short answer state never panics",
			qcm: config.QCM{Questions: []config.Question{
				testQuestion(2, 1, 1),
				testQuestion(2, 1),
			}},
			answers: []int{1},
			want: []config.QuestionMark{
				{Score: 1, Total: 2, State: config.Partial},
				{Score: 0, Total: 2, State: config.Incorrect},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountingPoints(tt.qcm, tt.answers)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d marks, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("mark %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
