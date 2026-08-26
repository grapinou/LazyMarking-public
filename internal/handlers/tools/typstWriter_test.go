package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestTypstWriterExamQCMKeepsMaliciousBusinessNamesInsideTempDir(t *testing.T) {
	markingStudentExamTestChdir(t)

	testRoot := t.TempDir()
	tempDir := filepath.Join(testRoot, "workspace")
	if err := os.Mkdir(tempDir, 0o750); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	qcm := config.QCM{
		Name: "a/b",
		Student: config.StudentQCM{
			FirstName: "../outside",
			LastName:  "../../escape",
			ClassCodes: config.ClassCode{
				Name: "../../../class",
			},
		},
	}

	typstPath, ok := TypstWriter(tempDir, "ignored", qcm, config.ExamQCM)
	if !ok {
		t.Fatal("TypstWriter returned not ok")
	}
	if got, want := filepath.Dir(typstPath), tempDir; got != want {
		t.Fatalf("TypstWriter path directory = %q, want %q", got, want)
	}
	if got, want := filepath.Ext(typstPath), ".typ"; got != want {
		t.Fatalf("TypstWriter path extension = %q, want %q", got, want)
	}
	if _, err := os.Stat(typstPath); err != nil {
		t.Fatalf("stat generated Typst file: %v", err)
	}

	entries, err := os.ReadDir(testRoot)
	if err != nil {
		t.Fatalf("read test root: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "workspace" {
		t.Fatalf("files escaped workspace: entries in test root = %v", entries)
	}
}
