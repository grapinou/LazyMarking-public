package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestTypstBuildContentAssociatesPDFsByStudentExamID(t *testing.T) {
	chdirToRepositoryRoot(t)

	markExams := []config.MarkExam{
		{StudentExamID: 7, FirstName: "Alex", LastName: "Same", Pages: 2},
		{StudentExamID: 9, FirstName: "Alex", LastName: "Same", Pages: 3},
		{StudentExamID: 12, FirstName: "Casey", LastName: "Other", Pages: 1},
	}
	pdfFiles := []string{
		filepath.Join("ignored", "student-exam-9.pdf"),
		filepath.Join("ignored", "student-exam-7.pdf"),
		filepath.Join("ignored", "student-exam-12.pdf"),
	}

	typstPath, ok := TypstBuildContent(t.TempDir(), markExams, pdfFiles)
	if !ok {
		t.Fatal("TypstBuildContent() failed")
	}
	content, err := os.ReadFile(typstPath)
	if err != nil {
		t.Fatalf("read generated Typst content: %v", err)
	}

	want := `"Alex Same", "1","Alex Same", "4","Casey Other", "6",)`
	if !strings.HasSuffix(string(content), want) {
		t.Fatalf("generated content does not preserve PDF order and page counts:\n%s\nwant suffix:\n%s", content, want)
	}
}

func TestTypstBuildContentAcceptsStudentExamIDSeven(t *testing.T) {
	chdirToRepositoryRoot(t)

	markExams := []config.MarkExam{{StudentExamID: 7, FirstName: "Taylor", LastName: "Student", Pages: 1}}

	typstPath, ok := TypstBuildContent(t.TempDir(), markExams, []string{"student-exam-7.pdf"})
	if !ok {
		t.Fatal("TypstBuildContent() failed")
	}
	content, err := os.ReadFile(typstPath)
	if err != nil {
		t.Fatalf("read generated Typst content: %v", err)
	}
	if !strings.HasSuffix(string(content), `"Taylor Student", "1",)`) {
		t.Fatalf("unexpected generated content: %s", content)
	}
}

func TestTypstBuildContentRejectsInvalidOrUnknownPDF(t *testing.T) {
	chdirToRepositoryRoot(t)

	tests := []struct {
		name string
		file string
	}{
		{name: "legacy student name", file: "Taylor_Student.pdf"},
		{name: "missing identifier", file: "student-exam-.pdf"},
		{name: "non numeric identifier", file: "student-exam-seven.pdf"},
		{name: "zero identifier", file: "student-exam-0.pdf"},
		{name: "unexpected suffix", file: "student-exam-7-copy.pdf"},
		{name: "unknown identifier", file: "student-exam-8.pdf"},
	}
	markExams := []config.MarkExam{{StudentExamID: 7, FirstName: "Taylor", LastName: "Student", Pages: 1}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if path, ok := TypstBuildContent(t.TempDir(), markExams, []string{tt.file}); ok || path != "" {
				t.Fatalf("TypstBuildContent(%q) = (%q, %v), want empty path and false", tt.file, path, ok)
			}
		})
	}
}

func chdirToRepositoryRoot(t *testing.T) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	t.Chdir(filepath.Clean(filepath.Join(filepath.Dir(filename), "../../..")))
}
