package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestStudentExamPDFNameUsesOnlyID(t *testing.T) {
	got := studentExamPDFName(42)
	if want := "student-exam-42.pdf"; got != want {
		t.Fatalf("studentExamPDFName(42) = %q, want %q", got, want)
	}
	for _, businessValue := range []string{"FirstName", "LastName", "ExamName"} {
		if strings.Contains(got, businessValue) {
			t.Fatalf("student PDF name contains business value %q: %q", businessValue, got)
		}
	}
}

func TestTypstBuildMarkTableUsesFixedWorkspaceFilename(t *testing.T) {
	chdirToRepositoryRoot(t)

	testRoot := t.TempDir()
	tempDir := filepath.Join(testRoot, "workspace")
	if err := os.Mkdir(tempDir, 0o750); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	markExams := []config.MarkExam{{
		ExamName:  "../../outside",
		ClassName: "classe/../../escape\\backslash",
		FirstName: "Anonymous",
		LastName:  "Student",
		Total:     1,
	}}

	typstPath, ok := TypstBuildMarkTable(tempDir, markExams, 0, 0, 0, nil, nil, nil, nil)
	if !ok {
		t.Fatal("TypstBuildMarkTable() failed")
	}
	want := filepath.Join(tempDir, "mark-table.typ")
	if filepath.Clean(typstPath) != want {
		t.Fatalf("TypstBuildMarkTable() path = %q, want %q", typstPath, want)
	}

	entries, err := os.ReadDir(testRoot)
	if err != nil {
		t.Fatalf("read test root: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "workspace" {
		t.Fatalf("files escaped workspace: %v", entries)
	}
}
