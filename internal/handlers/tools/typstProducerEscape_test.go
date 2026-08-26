package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestTypstWriterEscapesBusinessData(t *testing.T) {
	chdirToRepositoryRoot(t)
	qcm := hostileTypstQCM()

	typstPath, ok := TypstWriter(t.TempDir(), "anonymous", qcm, config.ExamQCM)
	if !ok {
		t.Fatal("TypstWriter() failed")
	}
	content := readTestFile(t, typstPath)

	assertContains(t, content, `#let exam="Exam\"; #let pwned = true; //"`)
	assertContains(t, content, `#let student="Student\\Name Last\"Name"`)
	assertContains(t, content, `#let classCode="Class\nName"`)
	assertContains(t, content, `#let question="Question \"#import \"evil.typ\""`)
	assertContains(t, content, `image("../../images/image\"\\name.png", width: 40%)`)
	assertContains(t, content, `answer("\u{25CB}", "Answer\nwith newline"),`)
}

func TestTypstLandscapeContentEscapesBusinessData(t *testing.T) {
	content := TypstLandscapeContent(hostileTypstQCM())

	assertContains(t, content, `#let question="Question \"#import \"evil.typ\""`)
	assertContains(t, content, `image("../../images/image\"\\name.png", width: 40%)`)
	assertContains(t, content, `answer("\u{25CB}", "Answer\nwith newline"),`)
}

func TestTypstBuildContentEscapesStudentName(t *testing.T) {
	chdirToRepositoryRoot(t)
	markExams := []config.MarkExam{{
		StudentExamID: 7,
		FirstName:     `Student"; #let pwned = true; //`,
		LastName:      `Back\Slash`,
		Pages:         1,
	}}

	typstPath, ok := TypstBuildContent(t.TempDir(), markExams, []string{"student-exam-7.pdf"})
	if !ok {
		t.Fatal("TypstBuildContent() failed")
	}
	content := readTestFile(t, typstPath)
	assertContains(t, content, `"Student\"; #let pwned = true; // Back\\Slash", "1",`)
}

func TestTypstBuildMarkTableEscapesBusinessData(t *testing.T) {
	chdirToRepositoryRoot(t)
	markExams := []config.MarkExam{{
		ExamName:  "AnonymousExam",
		ClassName: "AnonymousClass",
		FirstName: `Student"; #panic("name")`,
		LastName:  `Back\Slash`,
		Score:     1,
		Total:     2,
	}}
	globalSkills := map[int64]config.CounterTag{
		1: {Name: `Skill"; #panic("x")`, Score: 1, Total: 2},
	}
	globalThemeSkills := map[string]config.CounterTag{
		"theme": {Name: "Theme\nSkill", Score: 1, Total: 2},
	}
	notMarked := []config.MarkExam{{FirstName: `Missing"Name`, LastName: `Back\Slash`}}

	typstPath, ok := TypstBuildMarkTable(
		t.TempDir(), markExams, 1, 0, 1,
		globalSkills, globalThemeSkills,
		[]string{`QR"; #panic("qr")`}, notMarked,
	)
	if !ok {
		t.Fatal("TypstBuildMarkTable() failed")
	}
	content := readTestFile(t, typstPath)

	assertContains(t, content, `"Student\"; #panic(\"name\") Back\\Slash", "1.00/2",`)
	assertContains(t, content, `"Skill\"; #panic(\"x\")", "50.00",`)
	assertContains(t, content, `"Theme\nSkill", "50.00",`)
	assertContains(t, content, `"QR\"; #panic(\"qr\")"`)
	assertContains(t, content, `"Missing\"Name Back\\Slash "`)
}

func hostileTypstQCM() config.QCM {
	return config.QCM{
		Name: `Exam"; #let pwned = true; //`,
		Student: config.StudentQCM{
			FirstName: `Student\Name`,
			LastName:  `Last"Name`,
			ClassCodes: config.ClassCode{
				Name: "Class\nName",
			},
		},
		Questions: []config.Question{{
			Content: `Question "#import "evil.typ"`,
			Image:   config.Image{Name: "image\"\\name.png", Width: "40"},
			Answers: []config.Answer{{
				Symbol:  `\u{25CB}`,
				Content: "Answer\nwith newline",
			}},
		}},
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func assertContains(t *testing.T, content, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("generated Typst does not contain %q:\n%s", want, content)
	}
}
