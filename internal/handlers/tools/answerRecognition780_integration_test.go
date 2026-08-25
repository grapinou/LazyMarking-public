package tools

import "testing"

func TestRealOnePageAnswerRecognition780(t *testing.T) {
	want := []int{
		0, 0, 1, 0,
		0, 0, 0, 1,
		0, 0, 0, 1,
		1, 0, 0, 0,
	}
	testRealOnePageAnswerRecognition(t, 780, want)
}
