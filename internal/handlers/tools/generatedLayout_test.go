package tools

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func layoutWorkspace(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"typst", "pdftotext"} {
		if _, err := exec.LookPath(name); err != nil {
			t.Skip(name + " is required for real PDF checks")
		}
	}
	chdirToRepositoryRoot(t)
	base := os.Getenv("LAZYMARKING_LAYOUT_ARTIFACTS")
	if base != "" {
		dir := filepath.Join(base, t.Name())
		if err := os.MkdirAll(dir, 0750); err != nil {
			t.Fatal(err)
		}
		run, err := os.MkdirTemp(dir, "run-")
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("PDF artifacts: %s", run)
		return run
	}
	if err := os.MkdirAll("assets/tmp", 0750); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp("assets/tmp", "p5-layout-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func layoutExample() config.QCM {
	return config.QCM{LayoutVersion: 1, Name: "Évaluation des connaissances scientifiques sur la matière, les transformations et les méthodes expérimentales étudiées pendant le premier trimestre", Student: config.StudentQCM{FirstName: "Jean-Baptiste", LastName: "Dupont-Martin", ClassCodes: config.ClassCode{Name: "Sixième internationale — groupe des sciences expérimentales du lundi matin"}}, Questions: []config.Question{{Instruction: "Sélectionnez toutes les réponses exactes.", Content: "Quelle grandeur mesure une balance ?", Answers: []config.Answer{{Symbol: `\u{25CB}`, Content: "La masse", State: 1}, {Symbol: `\u{25CB}`, Content: "Le volume"}, {Symbol: `\u{25CB}`, Content: "La température"}, {Symbol: `\u{25CB}`, Content: "La durée"}}}}}
}

func compileLayout(t *testing.T, path string) string {
	t.Helper()
	pdf, ok := CompileTypst(path)
	if !ok {
		t.Fatal("real PDF compilation failed")
	}
	out, err := exec.Command("pdftotext", "-raw", pdf, "-").CombinedOutput()
	if err != nil {
		t.Fatalf("pdftotext: %v: %s", err, out)
	}
	if err := os.WriteFile(strings.TrimSuffix(pdf, ".pdf")+".txt", out, 0640); err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func assertPDFText(t *testing.T, text, want string) {
	t.Helper()
	// A line break after a hyphen must not count as truncation.
	normalize := func(s string) string { return strings.ReplaceAll(strings.Join(strings.Fields(s), " "), "- ", "-") }
	if !strings.Contains(normalize(text), normalize(want)) {
		t.Fatalf("PDF text missing %q:\n%s", want, text)
	}
}

func TestGeneratedLayoutPDFMatrix(t *testing.T) {
	dir := layoutWorkspace(t)
	long := layoutExample()
	extreme := layoutExample()
	extreme.Name = strings.Repeat("Évaluation des transformations physiques et chimiques au cours des expériences du trimestre. ", 4)
	extreme.Student.FirstName = "Jean-Baptiste Alexandre"
	extreme.Student.LastName = "Dupont-Martin de la Roche Saint-Clair"
	extreme.Student.ClassCodes.Name = strings.Repeat("Sixième internationale des sciences expérimentales ", 3)
	short := layoutExample()
	short.Name = "Matière"
	short.Student.FirstName = "Léa"
	short.Student.LastName = "Li"
	short.Student.ClassCodes.Name = "6eA"
	short.Questions[0].Instruction = ""
	for _, tc := range []struct {
		name string
		qcm  config.QCM
	}{{"short", short}, {"long", long}, {"very_long_identity", extreme}} {
		t.Run(tc.name, func(t *testing.T) {
			path, ok := TypstWriter(dir, tc.name, tc.qcm, config.PreviewQCM)
			if !ok {
				t.Fatal("write")
			}
			text := compileLayout(t, path)
			if strings.Count(text, "\f") != 1 {
				t.Fatalf("expected one page: %q", text)
			}
			for _, want := range []string{tc.qcm.Name, tc.qcm.Student.FirstName + " " + tc.qcm.Student.LastName, tc.qcm.Student.ClassCodes.Name, tc.qcm.Questions[0].Content} {
				assertPDFText(t, text, want)
			}
			if tc.qcm.Questions[0].Instruction != "" {
				assertPDFText(t, text, tc.qcm.Questions[0].Instruction)
			}
			assertIdentityWithinHeader(t, strings.TrimSuffix(path, ".typ")+".pdf")
		})
	}
	// Many answers and long questions exercise page and row breaks.
	multi := long
	multi.Questions = nil
	for i := 0; i < 12; i++ {
		q := long.Questions[0]
		q.Instruction = "Consigne : " + strings.Repeat("Lisez attentivement les données et comparez les propositions avant de répondre. ", 3)
		q.Content = fmt.Sprintf("Question %02d : ", i+1) + strings.Repeat("Un groupe observe les transformations de la matière et relève ses mesures dans un tableau. ", 4)
		q.Answers = append(append([]config.Answer{}, q.Answers...), q.Answers...)
		multi.Questions = append(multi.Questions, q)
	}
	path, ok := TypstWriter(dir, "multipage", multi, config.PreviewQCM)
	if !ok {
		t.Fatal("write")
	}
	text := compileLayout(t, path)
	if strings.Count(text, "\f") < 2 {
		t.Fatal("expected multiple pages")
	}
	for i := range multi.Questions {
		assertPDFText(t, text, fmt.Sprintf("Question %02d", i+1))
	}
	for _, page := range strings.Split(strings.TrimSuffix(text, "\f"), "\f") {
		assertPDFText(t, page, long.Student.FirstName+" "+long.Student.LastName)
		if !strings.Contains(page, "Question") && !strings.Contains(page, "masse") {
			t.Fatal("page without questions or answers")
		}
	}
	t.Logf("multipage: %d pages", strings.Count(text, "\f"))
	// An individual statement longer than one page must remain complete.
	veryLong := layoutExample()
	veryLong.Questions[0].Content = strings.Repeat("Les observations recueillies permettent de comparer les résultats de chaque groupe. ", 90) + "Fin du long énoncé."
	path, ok = TypstWriter(dir, "long-question", veryLong, config.PreviewQCM)
	if !ok {
		t.Fatal("write long question")
	}
	text = compileLayout(t, path)
	assertPDFText(t, text, "Fin du long énoncé.")
	if strings.Count(text, "observations") != 90 {
		t.Fatal("long question was cut")
	}
	// Short instruction/statement pairs must not be orphaned at a page boundary.
	pairs := layoutExample()
	pairs.Questions = nil
	for i := 0; i < 25; i++ {
		q := long.Questions[0]
		q.Instruction = fmt.Sprintf("Consigne%02d : choisissez les réponses exactes.", i)
		q.Content = fmt.Sprintf("Énoncé%02d : quelle grandeur mesure cet appareil ?", i)
		pairs.Questions = append(pairs.Questions, q)
	}
	path, ok = TypstWriter(dir, "instruction-pagination", pairs, config.PreviewQCM)
	if !ok {
		t.Fatal("write instruction pagination")
	}
	text = compileLayout(t, path)
	for _, page := range strings.Split(text, "\f") {
		for i := range pairs.Questions {
			if strings.Contains(page, fmt.Sprintf("Consigne%02d", i)) && !strings.Contains(page, fmt.Sprintf("Énoncé%02d", i)) {
				t.Fatalf("instruction %d separated from its statement", i)
			}
		}
	}
	// Both landscape entry points share the question renderer and title.
	path, ok = TypstWriterLandscape(dir, "landscape", long)
	if !ok {
		t.Fatal("write")
	}
	text = compileLayout(t, path)
	assertPDFText(t, text, long.Name)
	assertPDFText(t, text, long.Questions[0].Instruction)
	content, err := TypstLandscapeContent(long)
	if err != nil {
		t.Fatal(err)
	}
	path, ok = TypstWriterLandscapeAllContent(dir, "mini", []string{content, content})
	if !ok {
		t.Fatal("write")
	}
	text = compileLayout(t, path)
	if strings.Count(text, "\f") != 1 {
		t.Fatalf("two short mini copies should fit one page, got %d", strings.Count(text, "\f"))
	}
	if strings.Count(text, "Sélectionnez") != 2 {
		t.Fatal("missing instruction in mini copy")
	}
}

func assertIdentityWithinHeader(t *testing.T, pdf string) {
	t.Helper()
	out, err := exec.Command("pdftotext", "-bbox", pdf, "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Body struct {
			Pages []struct {
				Words []struct {
					YMin float64 `xml:"yMin,attr"`
					YMax float64 `xml:"yMax,attr"`
					Text string  `xml:",chardata"`
				} `xml:"word"`
			} `xml:"doc>page"`
		} `xml:"body"`
	}
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, p := range doc.Body.Pages {
		identityBottom := 0.0
		for _, w := range p.Words {
			if w.Text == "Répondez" {
				if w.YMin < identityBottom {
					t.Fatalf("marking instruction overlaps identity: y=%f, identity bottom=%f", w.YMin, identityBottom)
				}
				break
			}
			if w.YMax > identityBottom {
				identityBottom = w.YMax
			}
		}
		bodyStart := 1000.0
		for _, w := range p.Words {
			if (w.Text == "Sélectionnez" || w.Text == "Quelle") && w.YMin < bodyStart {
				bodyStart = w.YMin
			}
		}
		for _, w := range p.Words {
			if w.Text == "Jean-Baptiste" || w.Text == "Dupont-Martin" || w.Text == "Léa" || w.Text == "Matière" {
				count++
				if w.YMin < 0 || w.YMax >= bodyStart {
					t.Fatalf("identity outside header: %+v", w)
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("no identity bbox found")
	}
}

func TestGeneratedLayoutQRAndGeometry(t *testing.T) {
	for _, pageNumber := range []int{1, 10} {
		t.Run(fmt.Sprintf("page_%d", pageNumber), func(t *testing.T) { checkLayoutQRGeometry(t, pageNumber) })
	}
}

func checkLayoutQRGeometry(t *testing.T, pageNumber int) {
	setPageNumber := func(path string) {
		t.Helper()
		// Exercise the typography of a two-digit page without rendering nine
		// otherwise empty pages. Both layouts receive the same page number.
		source := readTestFile(t, path)
		source = strings.ReplaceAll(source, "#here().page()", fmt.Sprint(pageNumber))
		if err := os.WriteFile(path, []byte(source), 0640); err != nil {
			t.Fatal(err)
		}
	}
	dir := layoutWorkspace(t)
	qcm := layoutExample()
	qcm.Name = strings.Repeat(qcm.Name+" ", 3)
	qcm.Student.LastName = "Dupont-Martin de la Roche Saint-Clair"
	path, ok := TypstWriter(dir, "new", qcm, config.ExamQCM)
	if !ok {
		t.Fatal("write")
	}
	setPageNumber(path)
	text := compileLayout(t, path)
	assertPDFText(t, text, qcm.Student.FirstName+" "+qcm.Student.LastName)
	pages, ok := ExportTypstToPNGs(path)
	if !ok || len(pages) != 1 {
		t.Fatalf("render: %v %v", pages, ok)
	}
	old, ok := typstWriterLegacy(dir, "old", qcm, config.ExamQCM)
	if !ok {
		t.Fatal("legacy write")
	}
	setPageNumber(old)
	oldPages, ok := ExportTypstToPNGs(old)
	if !ok || len(oldPages) != 1 {
		t.Fatal("legacy render")
	}
	read := func(path string) image.Image {
		f, e := os.Open(path)
		if e != nil {
			t.Fatal(e)
		}
		defer f.Close()
		im, e := png.Decode(f)
		if e != nil {
			t.Fatal(e)
		}
		return im
	}
	before, after := read(oldPages[0]), read(pages[0])
	if after.Bounds() != image.Rect(0, 0, 2480, 3508) {
		t.Fatalf("page dimensions: %v", after.Bounds())
	}
	// These zones contain the QR reservation, right-hand alignment markers and footer.
	for _, r := range []image.Rectangle{image.Rect(110, 0, 525, 420), image.Rect(1900, 0, 2480, 420), image.Rect(0, 3320, 2480, 3508)} {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if color.NRGBAModel.Convert(before.At(x, y)) != color.NRGBAModel.Convert(after.At(x, y)) {
					t.Fatalf("fixed marker changed at %d,%d", x, y)
				}
			}
		}
	}
	circles, ok := CircleDetection(dir, filepath.Base(pages[0]))
	if !ok || len(circles) != 1 {
		t.Fatalf("question detection: %+v %v", circles, ok)
	}
	answers, ok := CircleDetectionAnswer(dir, filepath.Base(pages[0]), circles[0].Position.Y+35, 3390)
	if !ok || len(answers) != 4 {
		t.Fatalf("answer detection: %+v %v", answers, ok)
	}
	qr, ok := QrCodeMaker(dir, filepath.Base(pages[0]), config.QrCodeInfo{StudentExamID: 123, PageExam: 1})
	if !ok {
		t.Fatal("QR")
	}
	pasted, ok := PasteQrCodeOnPage(dir, qr, filepath.Base(pages[0]))
	if !ok {
		t.Fatal("paste")
	}
	decoded, err := DecodeWithGozxing(filepath.Join(dir, pasted))
	if err != nil {
		t.Fatal(err)
	}
	var info config.QrCodeInfo
	if err := json.Unmarshal([]byte(decoded), &info); err != nil {
		t.Fatal(err)
	}
	if info.StudentExamID != 123 || info.PageExam != 1 {
		t.Fatalf("QR: %+v", info)
	}
	if _, err := ConvertPngTopdf(dir, pasted); err != nil {
		t.Fatal(err)
	}
}
