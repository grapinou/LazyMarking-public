package tools

import "testing"

func TestAnswerMarks(t *testing.T) {
	tests := []struct {
		name                string
		expected, effective int
		want                answerRenderMarks
	}{
		{name: "correct answer selected", expected: 1, effective: 1, want: answerRenderMarks{expected: true, effective: true}},
		{name: "correct answer not selected", expected: 1, effective: 0, want: answerRenderMarks{expected: true}},
		{name: "incorrect answer selected", expected: 0, effective: 1, want: answerRenderMarks{effective: true, incorrectSelection: true}},
		{name: "incorrect answer not selected", expected: 0, effective: 0, want: answerRenderMarks{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := answerMarks(test.expected, test.effective); got != test.want {
				t.Fatalf("answerMarks(%d, %d) = %+v, want %+v", test.expected, test.effective, got, test.want)
			}
		})
	}
}
