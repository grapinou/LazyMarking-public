package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

// Exercise the real generation path, its persisted coordinates, QR insertion,
// immutable snapshot, native references and historical correction together.
func TestGeneratedLayoutPipelineSnapshots(t *testing.T) {
	artifacts := layoutWorkspace(t)
	if _, err := exec.LookPath("goose"); err != nil {
		t.Skip("goose is required for full generation fixture")
	}
	dbPath := filepath.Join(t.TempDir(), "p5.db")
	if out, err := exec.Command("goose", "-dir", "db/migrations", "sqlite3", dbPath, "up").CombinedOutput(); err != nil {
		t.Fatalf("migrations: %v\n%s", err, out)
	}
	conn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	queries := db.New(conn)
	ctx := context.Background()
	userDir, err := os.MkdirTemp("assets/tmp", "p5-generation-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(userDir) })
	username := filepath.Base(userDir)
	sql := func(query string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	sql("INSERT INTO users(id,username,email,hashpassword) VALUES(1,?,'p5@example.test','unused')", username)
	for _, table := range []string{"subjects", "themes", "year_levels", "skills", "difficulties", "class_codes", "periods", "years", "qcm"} {
		sql("INSERT INTO " + table + "(id,name,user_id) VALUES(1,'P5',1)")
	}
	sql("INSERT INTO points(id,point_value,user_id) VALUES(1,1,1)")
	sample := layoutExample()
	sql("INSERT INTO students(id,first_name,last_name,user_id) VALUES(1,?,?,1)", sample.Student.FirstName, sample.Student.LastName)
	sql("INSERT INTO exams(id,name,qcm_id,class_code_id,period_id,year_id,user_id) VALUES(1,?,1,1,1,1,1)", sample.Name)
	sql("INSERT INTO exams_generated(id,exam_id,total_students,user_id) VALUES(1,1,1,1)")
	for i := 1; i <= 6; i++ {
		sql("INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,content,instruction,user_id) VALUES(?,1,1,1,1,1,1,?,?,1)", i, fmt.Sprintf("Question %d. ", i)+strings.Repeat("Observez les données de cette expérience et comparez les mesures obtenues. ", 4), sample.Questions[0].Instruction)
		sql("INSERT INTO qcm_questions(qcm_id,question_id,position,user_id) VALUES(1,?,?,1)", i, i)
		if i == 2 {
			continue
		} // This family must select its only answerable variant.
		for a := 0; a < 8; a++ {
			sql("INSERT INTO answers(question_id,state,content,user_id) VALUES(?,?,?,1)", i, a%2, fmt.Sprintf("Proposition %d : une mesure expérimentale", a+1))
		}
	}
	sql("INSERT INTO alt_questions(id,question_id,content,user_id) VALUES(1,2,'Variante illustrée de la deuxième famille',1)")
	for a := 0; a < 8; a++ {
		sql("INSERT INTO alt_answers(alt_question_id,state,content,user_id) VALUES(1,?,?,1)", a%2, fmt.Sprintf("Réponse variante %d", a+1))
	}
	if err := os.MkdirAll("assets/images", 0750); err != nil {
		t.Fatal(err)
	}
	imgFile, err := os.CreateTemp("assets/images", "p5-*.png")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(imgFile.Name()) })
	img := image.NewRGBA(image.Rect(0, 0, 320, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 320; x++ {
			img.Set(x, y, color.RGBA{R: 80, G: uint8(100 + x/3), B: 190, A: 255})
		}
	}
	if err := png.Encode(imgFile, img); err != nil {
		t.Fatal(err)
	}
	imgFile.Close()
	sql("INSERT INTO alt_images(alt_question_id,image_name,resize_percentage,user_id) VALUES(1,?,40,1)", filepath.Base(imgFile.Name()))
	request := httptest.NewRequest("GET", "/", nil)
	main, err := GetQuestionAnswer(1, 1, queries, request)
	if err != nil || main.Instruction != sample.Questions[0].Instruction {
		t.Fatalf("main preview: %+v %v", main, err)
	}
	variant, err := GetAltQuestionAltAnswer(1, 1, queries, request)
	if err != nil || variant.Instruction != main.Instruction {
		t.Fatalf("variant preview: %+v %v", variant, err)
	}
	if _, err := GetAltQuestionAltAnswerCtx(2, 1, queries, ctx); err == nil {
		t.Fatal("foreign variant read allowed")
	}
	if _, err := queries.GetQuestionInstruction(ctx, db.GetQuestionInstructionParams{ID: 1, UserID: 2}); err == nil {
		t.Fatal("foreign instruction read allowed")
	}
	tempDir, ok := CreateOperationTempDir(username, "exam-1")
	if !ok {
		t.Fatal("generation workspace")
	}
	exam, err := queries.GetExamByID(ctx, db.GetExamByIDParams{ID: 1, UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := BuildQcmStudentCtx(db.Student{ID: 1, FirstName: sample.Student.FirstName, LastName: sample.Student.LastName}, exam, 1, 1, tempDir, username, sample.Student.ClassCodes.Name, ctx, queries)
	if err != nil {
		t.Fatal(err)
	}
	if generated.LayoutVersion != 1 || len(generated.Questions) != 6 {
		t.Fatalf("snapshot: %+v", generated)
	}
	for _, q := range generated.Questions {
		if q.Instruction != main.Instruction {
			t.Fatal("missing snapshot instruction")
		}
	}
	stored, err := queries.GetStudentContentExam(ctx, db.GetStudentContentExamParams{StudentExamID: 1, UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if stored.PageTot < 2 {
		t.Fatalf("expected multipage copy, got %d", stored.PageTot)
	}
	sql("UPDATE questions SET instruction='Nouvelle consigne de banque',content='Nouvel énoncé'")
	fresh, err := BuildQuestionCtx(2, 1, ctx, queries)
	if err != nil || fresh.Instruction != "Nouvelle consigne de banque" {
		t.Fatalf("updated family: %+v %v", fresh, err)
	}
	after, err := queries.GetStudentContentExam(ctx, db.GetStudentContentExamParams{StudentExamID: 1, UserID: 1})
	if err != nil || after.Content != stored.Content {
		t.Fatal("bank mutation changed snapshot")
	}
	var historical config.QCM
	if err := json.Unmarshal([]byte(after.Content), &historical); err != nil {
		t.Fatal(err)
	}
	path, ok := TypstWriter(artifacts, "historical", historical, config.PreviewQCM)
	if !ok {
		t.Fatal("historical PDF")
	}
	text := compileLayout(t, path)
	assertPDFText(t, text, main.Instruction)
	if strings.Contains(text, "Nouvelle consigne") {
		t.Fatal("historical PDF reads live bank")
	}
	// The versioned snapshot can also reproduce the native page geometry.
	reproduced, _, err := renderLegacyMarkingReferences(artifacts, username, historical)
	if err != nil || len(reproduced) != int(stored.PageTot) {
		t.Fatalf("snapshot reproduction: %v %v", reproduced, err)
	}
	for i, path := range reproduced {
		ref, err := ResolveStudentExamPageReference(ctx, queries, 1, username, 1, int64(i+1))
		if err != nil {
			t.Fatal(err)
		}
		original, err := os.ReadFile(ref.Path)
		if err != nil {
			t.Fatal(err)
		}
		copy, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(original, copy) {
			t.Fatalf("snapshot page %d differs from persisted reference", i+1)
		}
	}
	keepPDF := func(name string) {
		t.Helper()
		if os.Getenv("LAZYMARKING_LAYOUT_ARTIFACTS") == "" {
			return
		}
		pdf, err := os.ReadFile(filepath.Join(tempDir, "student-exam-1.pdf"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(artifacts, name), pdf, 0640); err != nil {
			t.Fatal(err)
		}
	}
	keepPDF("individualized.pdf")
	// Re-read QR from every final page, then correct using stored references and
	// page coordinates after the bank has been edited.
	scans, err := filepath.Glob(filepath.Join(tempDir, "qr_student-exam-*page-*.png"))
	if err != nil || len(scans) != int(stored.PageTot) {
		t.Fatalf("scan pages %v: %v", scans, err)
	}
	markedExam := config.Exam{StudentExamID: 1}
	for _, scan := range scans {
		decoded, err := DecodeWithGozxing(scan)
		if err != nil {
			t.Fatal(err)
		}
		var info config.QrCodeInfo
		if err := json.Unmarshal([]byte(decoded), &info); err != nil {
			t.Fatal(err)
		}
		if info.StudentExamID != 1 {
			t.Fatalf("QR: %+v", info)
		}
		markedExam.Pages = append(markedExam.Pages, config.Page{Number: info.PageExam, Name: filepath.Base(scan)})
	}
	result, err := MarkingStudentExam(1, username, tempDir, markedExam, ctx, queries)
	if err != nil || !result.Status || result.Total != 6 {
		t.Fatalf("historical correction: status=%v total=%d err=%v", result.Status, result.Total, err)
	}
	keepPDF("corrected.pdf")
	t.Logf("generated and corrected %d pages, 6 questions, 48 answers, illustrated variant; snapshot unchanged after bank edit", stored.PageTot)
}

func TestOldSnapshotUsesFrozenLayout(t *testing.T) {
	dir := layoutWorkspace(t)
	var old config.QCM
	if err := json.Unmarshal([]byte(`{"name":"Ancienne copie","student_qcm":{"first_name":"Jean","last_name":"Dupont"},"questions":[{"content":"Ancien énoncé","answers":[{"symbol":"\\u{25CB}","content":"Oui"}]}]}`), &old); err != nil {
		t.Fatal(err)
	}
	if old.LayoutVersion != 0 || old.Questions[0].Instruction != "" {
		t.Fatal("legacy defaults")
	}
	pages, _, err := renderLegacyMarkingReferences(dir, "legacy", old)
	if err != nil || len(pages) != 1 {
		t.Fatalf("legacy render: %v %v", pages, err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "student-exam-*.typ"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("legacy sources: %v", files)
	}
	source := readTestFile(t, files[0])
	if !strings.Contains(source, "#list(marker: [‣]") || strings.Contains(source, "#let instruction=") {
		t.Fatal("old snapshot used P5 layout")
	}
	assertPDFText(t, compileLayout(t, files[0]), "Ancien énoncé")
}
