package tools

import "testing"

func TestRealOnePageAnswerRecognition801(t *testing.T) {
	want := []int{
		0, 0, 0, 1,
		0, 0, 0, 1,
		1, 0, 1, 1,
		0, 0, 0, 1,
	}
	testRealOnePageAnswerRecognition(t, 801, want)
}
