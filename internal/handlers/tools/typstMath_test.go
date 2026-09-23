package tools

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestTypstQuestionTextKeepsClassicQuestionsLiteral(t *testing.T) {
	for _, input := range []string{
		"Quelle est la capitale de la France ?",
		`L'élève dit "oui" (à 20 °C) : #[] \ chemin`,
	} {
		got, err := typstQuestionText(input)
		if err != nil || got != typstStringLiteral(input) {
			t.Fatalf("classic question changed: %q -> %q, %v", input, got, err)
		}
	}
}

func TestTypstQuestionTextMathAndQCM(t *testing.T) {
	mixed, err := typstQuestionText(`L'élève dit "oui" #panic("x") : $rho = m / V$ (à 20 °C).`)
	if err != nil || !strings.Contains(mixed, `#text("L'élève dit \"oui\" #panic(\"x\") : ")$rho = m / V$`) {
		t.Fatalf("literal text escaped incorrectly beside math: %q, %v", mixed, err)
	}
	question := config.Question{
		Content:     `Calculer $rho = m / V$ avec $V = 60 "mL"$.`,
		Instruction: `Utiliser $E_c = 1/2 m v^2$.`,
		Answers: []config.Answer{
			{Symbol: `\u{25CB}`, Content: `$rho = m / V$`},
			{Symbol: `\u{25CB}`, Content: `$rho = V / m$`},
		},
	}
	source, err := typstQuestionContent(question)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`#text("Calculer ")$rho = m / V$`, `$V = 60 "mL"$`, `$E_c = 1/2 m v^2$`, `answer("\u{25CB}", [$rho = m / V$])`, `answer("\u{25CB}", [$rho = V / m$])`} {
		if !strings.Contains(source, want) {
			t.Fatalf("source missing %q:\n%s", want, source)
		}
	}
	if _, err := typstQuestionText(`bad $sqrt($`); err != nil {
		// Parenthesis syntax is checked by Typst itself, not this tokenizer.
		t.Fatalf("unexpected tokenizer error: %v", err)
	}
	if _, err := typstQuestionText(`bad $rho = m / V`); err == nil {
		t.Fatal("unclosed math should fail before writing")
	}
}

func TestRealTypstMathExamPDFAndInvalidExpression(t *testing.T) {
	dir := layoutWorkspace(t)
	qcm := config.QCM{
		Name:    "Masse volumique",
		Student: config.StudentQCM{FirstName: "Léa", LastName: "Martin", ClassCodes: config.ClassCode{Name: "2nde 3"}},
		Questions: []config.Question{
			{Content: `Une solution possède une masse $m = 78 g$ et un volume $V = 60 "mL"$. Calculer $rho = m / V$.`, Instruction: `Choisir la relation $rho = m / V$.`, Answers: []config.Answer{{Symbol: `\u{25CB}`, Content: `$rho = m / V$`, State: 1}, {Symbol: `\u{25CB}`, Content: `$rho = V / m$`}}},
			{Content: `Déterminer $x^2$ et $E_c$ avec $sqrt(x)$, $pi$ et $Delta$. Comparer $pi × rho ≈ Delta ± 10^n$.`, Answers: []config.Answer{{Symbol: `\u{25CB}`, Content: `$E_c = 1/2 m v^2$`}}},
		},
	}
	path, ok := TypstWriter(dir, "math", qcm, config.PreviewQCM)
	if !ok {
		t.Fatal("TypstWriter failed")
	}
	pdf, err := CompileTypstDetailed(path)
	if err != nil {
		t.Fatalf("real Typst math compilation failed: %v", err)
	}
	if info, err := os.Stat(pdf); err != nil || info.Size() == 0 {
		t.Fatalf("PDF missing or empty: %v", err)
	}
	out, err := exec.Command("pdftotext", "-raw", pdf, "-").CombinedOutput()
	if err != nil {
		t.Fatalf("pdftotext: %v: %s", err, out)
	}
	for _, want := range []string{"Une solution possède une masse", "Calculer", "Choisir la relation", "Déterminer", "78", "60", "𝜌", "𝜋", "Δ"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("PDF missing %q:\n%s", want, out)
		}
	}
	landscape, ok := TypstWriterLandscape(dir, "math-landscape", qcm)
	if !ok {
		t.Fatal("landscape writer failed")
	}
	if _, err := CompileTypstDetailed(landscape); err != nil {
		t.Fatalf("landscape PDF with math failed: %v", err)
	}

	qcm.Questions[0].Content = `Calculer $sqrt($.`
	path, ok = TypstWriter(dir, "invalid", qcm, config.PreviewQCM)
	if !ok {
		t.Fatal("TypstWriter failed before Typst syntax validation")
	}
	if _, err := CompileTypstDetailed(path); err == nil || !strings.Contains(err.Error(), "error:") {
		t.Fatalf("invalid math should produce a short compiler error, got %v", err)
	}
}

func TestValidateQuestionMathFields(t *testing.T) {
	w := httptest.NewRecorder()
	if ValidateQuestionText(w, "Énoncé", `Question $rho = m / V`) {
		t.Fatal("malformed math accepted by form validation")
	}
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "Énoncé") {
		t.Fatalf("unexpected form error: %d %s", w.Code, w.Body.String())
	}
	qcm := config.QCM{Questions: []config.Question{{Content: "Question", Answers: []config.Answer{{Content: "$#read(\"secret\")$"}}}}}
	if err := ValidateMathQCM(qcm); err == nil || !strings.Contains(err.Error(), "réponse 1") {
		t.Fatalf("invalid QCM answer not identified: %v", err)
	}
}
