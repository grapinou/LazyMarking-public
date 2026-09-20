package tools

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestTypstBuildMarkTableSupportsBatchWithoutCorrectedCopy(t *testing.T) {
	t.Chdir("../../..")
	path, ok := TypstBuildMarkTable(
		t.TempDir(), nil, 0, 0, 0,
		map[int64]config.CounterTag{}, map[string]config.CounterTag{},
		[]string{"page-1.png"}, []config.MarkExam{{FirstName: "Copie", LastName: "Incomplète"}},
	)
	if !ok {
		t.Fatal("empty batch mark table was rejected")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(contents)
	for _, want := range []string{`#let mean="0.00/0"`, `"page-1.png"`, `"Copie Incomplète `} {
		if !strings.Contains(text, want) {
			t.Fatalf("mark table missing %q", want)
		}
	}
}

func TestEmptyBatchTypstArtifactsCompileWhenTypstIsAvailable(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst is not installed")
	}
	t.Chdir("../../..")
	workspace, err := os.MkdirTemp(".", ".empty-marking-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workspace) })
	contentsPath, ok := TypstBuildContent(workspace, nil, nil)
	if !ok {
		t.Fatal("build empty corrected contents")
	}
	if pdf, ok := CompileTypst(contentsPath); !ok {
		t.Fatal("compile empty corrected contents")
	} else if info, err := os.Stat(pdf); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("empty corrected PDF info=%v err=%v", info, err)
	}
	markTablePath, ok := TypstBuildMarkTable(
		workspace, nil, 0, 0, 0,
		map[int64]config.CounterTag{}, map[string]config.CounterTag{}, nil, nil,
	)
	if !ok {
		t.Fatal("build empty mark table")
	}
	if pdf, ok := CompileTypst(markTablePath); !ok {
		t.Fatal("compile empty mark table")
	} else if info, err := os.Stat(pdf); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("empty mark table PDF info=%v err=%v", info, err)
	}
}
