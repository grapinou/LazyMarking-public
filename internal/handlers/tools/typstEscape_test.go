package tools

import "testing"

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
		{name: "Unicode", value: "élève 東京", want: `"élève 東京"`},
		{name: "injection payload", value: `"; #let injected = true; //`, want: `"\"; #let injected = true; //"`},
		{name: "Typst punctuation", value: `# $ [ ] *`, want: `"# $ [ ] *"`},
		{name: "combined", value: "quote \" slash \\ line\n", want: `"quote \" slash \\ line\n"`},
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
