package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestResolveExamGenerationPDFNameFindsHistoricalFinalPDF(t *testing.T) {
	t.Chdir(t.TempDir())
	workspace := createExamGenerationResolverWorkspace(t, "teacher", 42)
	const historicalName = "teacher_exam_Forces_4e A.pdf"
	if err := os.WriteFile(filepath.Join(workspace, historicalName), []byte("%PDF historical"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "generation.log"), []byte("ignored"), 0o640); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveExamGenerationPDFName("teacher", 42)
	if err != nil {
		t.Fatal(err)
	}
	if got != historicalName {
		t.Fatalf("name=%q, want historical artifact %q", got, historicalName)
	}
}

func TestResolveExamGenerationPDFNameRejectsAmbiguousWorkspace(t *testing.T) {
	t.Chdir(t.TempDir())
	workspace := createExamGenerationResolverWorkspace(t, "teacher", 42)
	for _, name := range []string{"first.pdf", "second.pdf"} {
		if err := os.WriteFile(filepath.Join(workspace, name), []byte("%PDF"), 0o640); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := ResolveExamGenerationPDFName("teacher", 42); !errors.Is(err, ErrAmbiguousExamGenerationPDF) {
		t.Fatalf("error=%v, want ErrAmbiguousExamGenerationPDF", err)
	}
}

func TestResolveExamGenerationPDFNameIgnoresUnsafeAndNonPDFEntries(t *testing.T) {
	t.Chdir(t.TempDir())
	workspace := createExamGenerationResolverWorkspace(t, "teacher", 42)
	if err := os.WriteFile(filepath.Join(workspace, "notes.txt"), []byte("not a PDF"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workspace, "directory.pdf"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("notes.txt", filepath.Join(workspace, "link.pdf")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := ResolveExamGenerationPDFName("teacher", 42); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error=%v, want not found", err)
	}
}

func TestResolveExamGenerationPDFNameRejectsForeignOrMissingWorkspace(t *testing.T) {
	t.Chdir(t.TempDir())
	workspace := createExamGenerationResolverWorkspace(t, "bob", 42)
	if err := os.WriteFile(filepath.Join(workspace, "bob.pdf"), []byte("bob secret"), 0o640); err != nil {
		t.Fatal(err)
	}

	if _, err := ResolveExamGenerationPDFName("alice", 42); err == nil {
		t.Fatal("foreign workspace resolved")
	}
	if _, err := ResolveExamGenerationPDFName("../bob", 42); err == nil {
		t.Fatal("unsafe username accepted")
	}
}

func createExamGenerationResolverWorkspace(t *testing.T, username string, generationID int64) string {
	t.Helper()
	workspace, ok := CreateOperationTempDir(username, "exam-"+stringInt64ForResolverTest(generationID))
	if !ok {
		t.Fatal("create resolver workspace")
	}
	return workspace
}

func stringInt64ForResolverTest(value int64) string {
	return strconv.FormatInt(value, 10)
}
