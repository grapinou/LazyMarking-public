package generateexams

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanupExamGenerationFilesPreservesReferencesAndFinalPDF(t *testing.T) {
	dir := t.TempDir()
	intermediatePDF := filepath.Join(dir, "student-exam-1.pdf")
	intermediatePNG := filepath.Join(dir, "native.png")
	intermediateTypst := filepath.Join(dir, "native.typ")
	finalPDF := filepath.Join(dir, "final.pdf")
	reference := filepath.Join(dir, "references", "student-exam-1", "page-1.png")
	for path, content := range map[string]string{
		intermediatePDF: "intermediate", intermediatePNG: "png", intermediateTypst: "typst",
		finalPDF: "final", reference: "reference",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cleanupExamGenerationFiles(dir, []string{intermediatePDF})
	for _, removed := range []string{intermediatePDF, intermediatePNG, intermediateTypst} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("intermediate %s remains: %v", removed, err)
		}
	}
	for _, preserved := range []string{finalPDF, reference} {
		if _, err := os.Stat(preserved); err != nil {
			t.Fatalf("durable file %s missing: %v", preserved, err)
		}
	}
}

func TestExamGenerationPDFName(t *testing.T) {
	got := examGenerationPDFName("teacher", "sample exam", "class A")
	want := "teacher_exam_sample exam_class A.pdf"
	if got != want {
		t.Fatalf("examGenerationPDFName() = %q, want %q", got, want)
	}
}

func TestSafeExamFilenamePartPreservesOrdinaryValues(t *testing.T) {
	for _, value := range []string{"sample exam", "6A", "Mélange"} {
		if got := safeExamFilenamePart(value); got != value {
			t.Fatalf("safeExamFilenamePart(%q) = %q, want unchanged", value, got)
		}
	}
}

func TestExamGenerationPDFNameNeutralizesUnsafeComponents(t *testing.T) {
	tests := []struct {
		name      string
		username  string
		examName  string
		className string
	}{
		{name: "parent components", username: "../", examName: "../../escape", className: "6A\\outside"},
		{name: "controls", username: "teach\x00er", examName: "exam\nname", className: "class\tname\x7f"},
		{name: "empty and dots", username: "", examName: ".", className: ".."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := examGenerationPDFName(tt.username, tt.examName, tt.className)
			if filepath.Base(name) != name {
				t.Fatalf("filepath.Base(%q) != name", name)
			}
			if strings.ContainsAny(name, `/\\`) {
				t.Fatalf("generated name contains a path separator: %q", name)
			}
			for _, r := range name {
				if r < 0x20 || r == 0x7f {
					t.Fatalf("generated name contains control character %U: %q", r, name)
				}
			}
		})
	}
}
