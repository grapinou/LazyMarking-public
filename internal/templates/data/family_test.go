package data

import "testing"

func TestQuestionURL(t *testing.T) {
	got := QuestionURL("/dashboard/questions/answers", 42)
	want := "/dashboard/questions/answers?question_id=42"
	if got != want {
		t.Fatalf("QuestionURL() = %q, want %q", got, want)
	}
}

func TestVariantURL(t *testing.T) {
	got := VariantURL("/dashboard/questions/altquestions/altanswers", 42, 7)
	want := "/dashboard/questions/altquestions/altanswers?question_id=42&alt_question_id=7"
	if got != want {
		t.Fatalf("VariantURL() = %q, want %q", got, want)
	}
}

func TestQCMURL(t *testing.T) {
	got := QCMURL("/dashboard/qcm/qcmquestion", 42)
	want := "/dashboard/qcm/qcmquestion?qcm_id=42"
	if got != want {
		t.Fatalf("QCMURL() = %q, want %q", got, want)
	}
}
