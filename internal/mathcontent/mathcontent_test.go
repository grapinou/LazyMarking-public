package mathcontent

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseLiteralAndMath(t *testing.T) {
	tests := []struct {
		name, input string
		want        []Segment
	}{
		{"old plain question", "Quelle est la capitale de la France ?", []Segment{{Text: "Quelle est la capitale de la France ?"}}},
		{"punctuation", `L'élève dit : "masse" (en g), é ! # [] \\`, []Segment{{Text: `L'élève dit : "masse" (en g), é ! # [] \\`}}},
		{"simple formula", "Résoudre $2 x + 3 = 7$.", []Segment{{Text: "Résoudre "}, {Text: "2 x + 3 = 7", Math: true}, {Text: "."}}},
		{"several formulas", "Avec $E_c = 1/2 m v^2$ et $rho = m / V$.", []Segment{{Text: "Avec "}, {Text: "E_c = 1/2 m v^2", Math: true}, {Text: " et "}, {Text: "rho = m / V", Math: true}, {Text: "."}}},
		{"root greek and units", `$sqrt(x) = pi + Delta$ ; $V = 60 "mL"$`, []Segment{{Text: `sqrt(x) = pi + Delta`, Math: true}, {Text: " ; "}, {Text: `V = 60 "mL"`, Math: true}}},
		{"decimal", `$rho = 1.3$`, []Segment{{Text: "rho = 1.3", Math: true}}},
		{"escaped dollar", `Coût : \$5 et $x^2$`, []Segment{{Text: "Coût : $5 et "}, {Text: "x^2", Math: true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Parse(%q) = %#v, %v; want %#v", tt.input, got, err, tt.want)
			}
		})
	}
}

func TestParseRejectsMalformedAndExecutableMath(t *testing.T) {
	for _, input := range []string{
		"Avant $rho = m / V", "Texte $$ après", "$#read(\"secret\")$", `$std.read("secret")$`, "$x] #set page(margin: 0pt) [$", "$sqrt(x); panic()$", "$x\\ y$", "$x\x00y$",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := Parse(input); err == nil || !strings.Contains(err.Error(), "Typst") {
				t.Fatalf("expected readable Typst error for %q, got %v", input, err)
			}
		})
	}
}
