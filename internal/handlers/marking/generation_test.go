package marking

import (
	"bytes"
	"context"
	"mime"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func newCumulativeFixture(t *testing.T) (reviewPageFixture, *http.ServeMux) {
	t.Helper()
	f := newReviewPageFixture(t)
	_, err := f.conn.Exec(`
		ALTER TABLE marking_jobs ADD COLUMN exam_generated_id INTEGER;
		ALTER TABLE marking_jobs ADD COLUMN status_pdf TEXT NOT NULL DEFAULT 'success';
		ALTER TABLE marking_jobs ADD COLUMN completed_at TIMESTAMP;
		CREATE TABLE class_codes(id INTEGER PRIMARY KEY,name TEXT NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE exams(id INTEGER PRIMARY KEY,name TEXT NOT NULL,class_code_id INTEGER NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE exams_generated(id INTEGER PRIMARY KEY,exam_id INTEGER NOT NULL,user_id INTEGER NOT NULL,status TEXT NOT NULL,created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE students(id INTEGER PRIMARY KEY,first_name TEXT NOT NULL,last_name TEXT NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE student_exam(id INTEGER PRIMARY KEY,exam_generated_id INTEGER NOT NULL,student_id INTEGER NOT NULL,user_id INTEGER NOT NULL);
		INSERT INTO class_codes VALUES(1,'2nde 3',1),(2,'Autre classe',2);
		INSERT INTO exams VALUES(1,'Contrôle de physique n°2',1,1),(2,'Évaluation privée',2,2);
		INSERT INTO exams_generated(id,exam_id,user_id,status) VALUES(10,1,1,'success'),(11,1,1,'running'),(20,2,2,'success'),(12,1,1,'success');
		INSERT INTO students VALUES(100,'Alice','Zola',1),(101,'Bob','Éclair',1),(103,'Charlie','Albert',1),(104,'David','Dupont',1),(106,'Eva','Éclair',1),(102,'Privé','Secret',2);
		INSERT INTO student_exam VALUES(100,10,100,1),(101,10,101,1),(103,10,103,1),(104,10,104,1),(106,10,106,1),(102,20,102,2);
		UPDATE marking_jobs SET exam_generated_id=10,completed_at=CURRENT_TIMESTAMP,exam_name='corrected.pdf',mark_table_name='mark-table.pdf',source_pdf_filename='lot-1.pdf' WHERE id=50;
		UPDATE marking_jobs SET exam_generated_id=20 WHERE id=51;
		UPDATE marking_answer_detections SET mean_gray=220 WHERE id IN (700,701,710);
		UPDATE marking_copy_results SET outcome='corrected',score_half_units=2,total_points=1 WHERE id=501;
		INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,expected_pages,detected_pages,score_half_units,total_points)
		VALUES(503,1,50,103,'corrected',1,1,2,1),(504,1,50,104,'not_seen',1,0,NULL,NULL),(506,1,50,106,'not_seen',1,0,NULL,NULL);
	`)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, f.queries, context.Background(), &sync.WaitGroup{})
	return f, mux
}

func cumulativeGET(t *testing.T, mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(markingSessionCookie(t, req))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestCumulativeResultsAndPDFThroughImportsAndHumanDecisions(t *testing.T) {
	f, mux := newCumulativeFixture(t)
	havePDFTools := true
	for _, binary := range []string{"typst", "pdftotext"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Logf("%s unavailable: actual PDF extraction is skipped; HTML and state assertions still run", binary)
			havePDFTools = false
		}
	}
	check := func(wantCorrected, wantPending int64, wantScore, wantMean, wantProgress string) {
		t.Helper()
		page, err := loadMarkingGeneration(t.Context(), f.queries, 1, 10)
		if err != nil {
			t.Fatal(err)
		}
		if page.Summary.Corrected != wantCorrected || page.Summary.PendingReview != wantPending {
			t.Fatalf("summary=%+v", page.Summary)
		}
		if page.Progress.StatusLabel != wantProgress {
			t.Fatalf("progress=%+v, want %q", page.Progress, wantProgress)
		}
		if page.Pedagogy.IncludedCopies != int(wantCorrected) {
			t.Fatalf("statistics population differs from current results: %+v", page.Pedagogy)
		}
		mean := ""
		for _, group := range page.Pedagogy.ScoreGroups {
			if group.Total == 2 {
				mean = group.Mean
			}
		}
		if mean != wantMean {
			t.Fatalf("current mean=%q, want %q", mean, wantMean)
		}
		screen := cumulativeGET(t, mux, pageURL(10))
		if screen.Code != http.StatusOK {
			t.Fatalf("screen %d: %s", screen.Code, screen.Body.String())
		}
		if !strings.Contains(screen.Body.String(), `<h1 class="h2 text-break">2nde 3 — Contrôle de physique n°2</h1>`) {
			t.Fatalf("screen does not identify class and evaluation: %s", screen.Body.String())
		}
		wantOrder := []string{"Albert Charlie", "Dupont David", "Éclair Bob", "Éclair Eva", "Zola Alice"}
		assertNamesInOrder(t, screen.Body.String(), wantOrder)
		if strings.Contains(screen.Body.String(), "<td><a") {
			t.Fatal("source import must be plain text")
		}
		alice := page.Summary.Results[4]
		if alice.ScoreLabel != wantScore {
			t.Fatalf("Alice score=%q, want %q", alice.ScoreLabel, wantScore)
		}
		if !havePDFTools {
			return
		}
		text := cumulativePDFText(t, mux, page.PDFURL)
		assertNamesInOrder(t, text, wantOrder)
		// Every rendered row must agree with the shared screen model, including
		// hidden provisional scores. Read a physical PDF, not a mocked renderer.
		for _, result := range page.Summary.Results {
			line := ""
			for _, candidate := range strings.Split(text, "\n") {
				if strings.Contains(candidate, result.StudentName) {
					line = candidate
					break
				}
			}
			if !strings.Contains(line, result.StatusLabel) {
				t.Fatalf("missing status in PDF line %q", line)
			}
			if result.HasScore && !strings.Contains(strings.ReplaceAll(line, " ", ""), strings.ReplaceAll(result.ScoreLabel, " ", "")) {
				t.Fatalf("PDF score disagrees: %q", line)
			}
			if result.Pending && !strings.Contains(line, "Après vérification") {
				t.Fatalf("pending PDF row=%q", line)
			}
		}
	}
	check(3, 0, "2 / 2", "2,00", "Correction partielle") // first batch; two absent pupils remain ungraded.
	_, err := f.conn.Exec(`
		INSERT INTO marking_jobs(id,user_id,status,status_pdf,review_revision,artifacts_revision,exam_generated_id,source_pdf_filename,completed_at)
		VALUES(60,1,'success','success',0,0,10,'lot-2.pdf',CURRENT_TIMESTAMP);
		INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,expected_pages,detected_pages,score_half_units,total_points)
		VALUES(6000,1,60,100,'not_seen',1,0,NULL,NULL),(6001,1,60,101,'not_seen',1,0,NULL,NULL),
		(6003,1,60,103,'not_seen',1,0,NULL,NULL),(6004,1,60,104,'corrected',1,1,3,2),(6006,1,60,106,'corrected',1,1,2,2);
	`)
	if err != nil {
		t.Fatal(err)
	}
	check(5, 0, "2 / 2", "1,50", "Corrigé") // catch-up must preserve all earlier corrected rows.
	if _, err := f.conn.Exec("UPDATE marking_answer_detections SET mean_gray=150 WHERE id=701"); err != nil {
		t.Fatal(err)
	}
	check(4, 1, "", "1,25", "Correction partielle") // a pending decision masks the persisted score in both outputs.
	_, err = db.ApplyMarkingAnswerReview(t.Context(), f.queries, db.ApplyMarkingAnswerReviewInput{
		UserID: 1, MarkingJobID: 50, AnswerDetectionID: 701, ReviewedState: 0, ExpectedJobReviewRevision: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	check(5, 0, "2 / 2", "1,50", "Corrigé")
	_, err = db.ApplyMarkingAnswerReview(t.Context(), f.queries, db.ApplyMarkingAnswerReviewInput{
		UserID: 1, MarkingJobID: 50, AnswerDetectionID: 701, ReviewedState: 1, ExpectedJobReviewRevision: 2, ExpectedAnswerReviewRevision: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	check(5, 0, "1 / 2", "1,17", "Corrigé") // next download reflects the committed review immediately.
	_, err = f.conn.Exec(`
		INSERT INTO marking_jobs(id,user_id,status,status_pdf,review_revision,artifacts_revision,exam_generated_id)
		VALUES(61,1,'success','success',0,0,10),(62,1,'failed','success',0,0,10),(63,1,'running','running',0,0,10);
		INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,expected_pages,detected_pages,score_half_units,total_points)
		VALUES(6100,1,61,100,'corrected',1,1,3,2),(6200,1,62,100,'corrected',1,1,0,2),(6300,1,63,100,'corrected',1,1,0,2);
	`)
	if err != nil {
		t.Fatal(err)
	}
	check(5, 0, "1,5 / 2", "1,33", "Corrigé") // newer successful copy wins; failed/running copies cannot.
	if got := f.copyScore(t, 500); got != 2 {
		t.Fatalf("historical reviewed score overwritten: %d", got)
	}
}

func cumulativePDFText(t *testing.T, mux *http.ServeMux, url string) string {
	t.Helper()
	for _, binary := range []string{"typst", "pdftotext"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("%s unavailable: PDF extraction skipped", binary)
		}
	}
	doc := cumulativeGET(t, mux, url)
	if doc.Code != http.StatusOK || !bytes.HasPrefix(doc.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("PDF %d: %s", doc.Code, doc.Body.String())
	}
	if doc.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("report may be cached")
	}
	if got := doc.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("Content-Type=%q", got)
	}
	disposition, params, err := mime.ParseMediaType(doc.Header().Get("Content-Disposition"))
	if err != nil || disposition != "attachment" || params["filename"] != "2nde-3-controle-de-physique-n2-bilan.pdf" {
		t.Fatalf("Content-Disposition=%q parsed=(%q,%v,%v)", doc.Header().Get("Content-Disposition"), disposition, params, err)
	}
	command := exec.Command("pdftotext", "-layout", "-", "-")
	command.Stdin = bytes.NewReader(doc.Body.Bytes())
	extracted, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	text := string(extracted)
	for _, want := range []string{"Bilan de l’évaluation", "2nde 3", "Contrôle de physique n°2"} {
		if !strings.Contains(text, want) {
			t.Fatalf("PDF does not contain %q: %s", want, text)
		}
	}
	return text
}

func pageURL(id int64) string {
	return data.DefaultMarkingRoutes.GenerationResults + "?exam_generated_id=" + strconv.FormatInt(id, 10)
}

func assertNamesInOrder(t *testing.T, text string, names []string) {
	t.Helper()
	previous := -1
	for _, name := range names {
		index := strings.Index(text, name)
		if index <= previous {
			t.Fatalf("missing or unordered %q: %s", name, text)
		}
		previous = index
	}
}

func TestCumulativeNavigationAndOwnership(t *testing.T) {
	f, mux := newCumulativeFixture(t)
	main := cumulativeGET(t, mux, "/dashboard/marking").Body.String()
	for _, want := range []string{"Correction partielle", "3 corrigées · 2 sans correction finale", "Non corrigé", "Aucune copie finalisée"} {
		if !strings.Contains(main, want) {
			t.Fatalf("missing generation progress %q: %s", want, main)
		}
	}
	for _, want := range []string{`href="/dashboard/marking?exam_generated_id=10">Ajouter les copies manquantes`, `href="/dashboard/marking/results?exam_generated_id=10">Voir les résultats`} {
		if !strings.Contains(main, want) || strings.Index(main, want) > strings.Index(main, "Historique des imports récents") {
			t.Fatalf("primary action missing or hidden in history: %s", want)
		}
	}
	if strings.Contains(main, "exam_generated_id=20") || strings.Contains(main, "exam_generated_id=11") {
		t.Fatal("main correction page exposes foreign or unready generation")
	}
	result := cumulativeGET(t, mux, "/dashboard/marking/success?job_id=50")
	if result.Code != http.StatusSeeOther || result.Header().Get("Location") != pageURL(10) {
		t.Fatalf("result redirect=%v", result)
	}
	progress := cumulativeGET(t, mux, "/dashboard/marking/progress?job_id=50")
	if progress.Code != http.StatusSeeOther || progress.Header().Get("Location") != "/dashboard/marking/success?job_id=50" {
		t.Fatalf("progress redirect=%v", progress)
	}
	for _, path := range []string{
		"/dashboard/marking?exam_generated_id=10", pageURL(10),
		"/dashboard/marking/success?job_id=50&import=1", pageURL(12),
	} {
		response := cumulativeGET(t, mux, path)
		if response.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, response.Code, response.Body.String())
		}
	}
	form := cumulativeGET(t, mux, "/dashboard/marking?exam_generated_id=10").Body.String()
	if !strings.Contains(form, `type="hidden" name="exam_generated_id" value="10"`) || strings.Contains(form, `<select class="form-select" id="exam_generated_id"`) {
		t.Fatal("catch-up form does not lock the generation")
	}
	if strings.Contains(form, "exam_generated_id=12") || strings.Contains(form, "Historique des imports récents") {
		t.Fatal("catch-up upload should stay focused on the selected evaluation")
	}
	page := cumulativeGET(t, mux, pageURL(10)).Body.String()
	if !strings.Contains(page, `href="/dashboard/marking?exam_generated_id=10"`) || !strings.Contains(page, "Ajouter les copies manquantes") {
		t.Fatal("missing catch-up action on the evaluation")
	}
	for _, route := range []string{"/dashboard/marking", data.DefaultMarkingRoutes.GenerationResults, data.DefaultMarkingRoutes.GenerationPDF} {
		for _, id := range []string{"20", "11", "999"} {
			response := cumulativeGET(t, mux, route+"?exam_generated_id="+id)
			if response.Code != http.StatusNotFound {
				t.Fatalf("%s generation %s: %d", route, id, response.Code)
			}
		}
		if response := cumulativeGET(t, mux, route+"?exam_generated_id=abc"); response.Code != http.StatusBadRequest {
			t.Fatalf("invalid ID: %d", response.Code)
		}
	}
	// A legacy job without a generation keeps its useful historical result page.
	if _, err := f.conn.Exec("UPDATE marking_jobs SET exam_name='corrected.pdf',mark_table_name='mark-table.pdf' WHERE id=56"); err != nil {
		t.Fatal(err)
	}
	legacy := cumulativeGET(t, mux, "/dashboard/marking/success?job_id=56")
	if legacy.Code != http.StatusOK || !strings.Contains(legacy.Body.String(), "Ouvrir les copies corrigées") {
		t.Fatalf("legacy result: %d %s", legacy.Code, legacy.Body.String())
	}
}
