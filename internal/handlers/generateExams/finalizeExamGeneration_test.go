package generateexams

import "testing"

func TestExamGenerationPDFName(t *testing.T) {
	got := examGenerationPDFName("teacher", "sample exam", "class A")
	want := "teacher_exam_sample exam_class A.pdf"
	if got != want {
		t.Fatalf("examGenerationPDFName() = %q, want %q", got, want)
	}
}
