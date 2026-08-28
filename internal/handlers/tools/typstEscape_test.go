package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTypstStringLiteral(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "ASCII", value: "hello", want: `"hello"`},
		{name: "double quote", value: `a"b`, want: `"a\"b"`},
		{name: "backslash", value: `a\b`, want: `"a\\b"`},
		{name: "newline", value: "a\nb", want: `"a\nb"`},
		{name: "carriage return", value: "a\rb", want: `"a\rb"`},
		{name: "tab", value: "a\tb", want: `"a\tb"`},
		{name: "Unicode", value: "é, µ, ° et 6,02 × 10²³", want: `"é, µ, ° et 6,02 × 10²³"`},
		{name: "injection payload", value: `"; #let injected = true; //`, want: `"\"; #let injected = true; //"`},
		{name: "Typst punctuation", value: `# $ [ ] *`, want: `"# $ [ ] *"`},
		{name: "combined", value: "quote \" slash \\ line\ncarriage\rreturn\ttab µ°é", want: `"quote \" slash \\ line\ncarriage\rreturn\ttab µ°é"`},
		{name: "other controls", value: "a\x00\x07\x1fb\x7f", want: `"a\u{0}\u{7}\u{1f}b\u{7f}"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := typstStringLiteral(tt.value); got != tt.want {
				t.Fatalf("typstStringLiteral(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestTypstStringLiteralCompilesWithTypst(t *testing.T) {
	typstBinary, err := exec.LookPath("typst")
	if err != nil {
		t.Skip("typst executable not available")
	}

	value := "Quelle grandeur appelle-t-on \"masse volumique\" ?\nchemin \\ exemple\nUnicode : µ, °, é"
	dir, err := os.MkdirTemp(".", ".typst-integration-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sourcePath := filepath.Join(dir, "escaped-content.typ")
	pdfPath := filepath.Join(dir, "escaped-content.pdf")
	source := []byte("#let question = " + typstStringLiteral(value) + "\n#question\n")
	if err := os.WriteFile(sourcePath, source, 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(typstBinary, "compile", sourcePath, pdfPath).CombinedOutput(); err != nil {
		t.Fatalf("typst compile failed: %v\n%s", err, output)
	}
}
