package generateexams

import (
	"errors"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestBuildExamGenerationProgressPageData(t *testing.T) {
	progressURL := examGenerationProgressURL(42, url.Values{
		"exam_id":            {"7"},
		"generation_started": {"1"},
	})
	page := buildExamGenerationProgressPageData(42, "running", db.GetExamGeneratedProgressRow{
		ProcessedStudents: 3,
		TotalStudents:     8,
	}, progressURL)

	if page.Context.GenerationID != 42 {
		t.Fatalf("GenerationID=%d, want 42", page.Context.GenerationID)
	}
	if page.Progress.Status != "running" || page.Progress.ProcessedStudents != 3 || page.Progress.TotalStudents != 8 {
		t.Fatalf("Progress=%+v", page.Progress)
	}
	if page.Progress.ExamsURL != data.DefaultDashboardRoutes.ExamURL {
		t.Fatalf("ExamsURL=%q", page.Progress.ExamsURL)
	}
	assertGenerationProgressURL(t, page.Progress.ProgressURL, 42, 7)
}

func TestGenerationProgressURLsDoNotMixGenerationsOrExams(t *testing.T) {
	first := examGenerationProgressURL(10, url.Values{"exam_id": {"1"}, "generation_started": {"1"}})
	second := examGenerationProgressURL(20, url.Values{"exam_id": {"2"}, "generation_started": {"1"}})
	assertGenerationProgressURL(t, first, 10, 1)
	assertGenerationProgressURL(t, second, 20, 2)
}

func TestBuildExamGenerationSuccessPageData(t *testing.T) {
	page := buildExamGenerationSuccessPageData(42, db.GetExamNameAndClassCodeNameRow{
		ExamName:  "Forces",
		ClassName: "3e A",
	}, "historical-name.pdf")

	if page.Context.GenerationID != 42 || page.Context.ExamName != "Forces" || page.Context.ClassName != "3e A" {
		t.Fatalf("Context=%+v", page.Context)
	}
	if page.Success.Status != "success" || page.Success.ExamsURL != data.DefaultDashboardRoutes.ExamURL {
		t.Fatalf("Success=%+v", page.Success)
	}
	copiesURL, err := url.Parse(page.Success.CopiesURL)
	if err != nil {
		t.Fatal(err)
	}
	if copiesURL.Path != data.DefaultGenerateExamRoutes.PdfExam || copiesURL.Query().Get("operation") != "exam-42" || copiesURL.Query().Get("file") != "historical-name.pdf" {
		t.Fatalf("CopiesURL=%q", page.Success.CopiesURL)
	}
}

func TestSuccessKeepsHistoricalPDFAccessAfterClassRename(t *testing.T) {
	conn, queries := setupExamPreflightTest(t)
	const generationID int64 = 9_000_000_000_000_099
	if _, err := conn.Exec("INSERT INTO exams_generated(id,exam_id,total_students,status,user_id) VALUES(?,6,1,'success',1)", generationID); err != nil {
		t.Fatal(err)
	}

	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(current) })

	operation := "exam-" + strconv.FormatInt(generationID, 10)
	workspace, ok := tools.CreateOperationTempDir("teacher", operation)
	if !ok {
		t.Fatal("create historical generation workspace")
	}
	t.Cleanup(func() { _ = tools.RemoveOperationTempDir("teacher", operation) })

	const historicalName = "teacher_exam_workspace failure_1A.pdf"
	const historicalContent = "%PDF-1.4\nhistorical artifact\n%%EOF\n"
	if err := os.WriteFile(filepath.Join(workspace, historicalName), []byte(historicalContent), 0o640); err != nil {
		t.Fatal(err)
	}
	wantURL := examGenerationCopiesURL(generationID, historicalName)
	progressURL := data.DefaultGenerateExamRoutes.ProcessingStudents + "?exam_generated_id=" + strconv.FormatInt(generationID, 10)

	loadSuccess := func() *httptest.ResponseRecorder {
		return serveAuthenticatedGenerationRequestMethod(t, http.MethodGet, progressURL, func(w http.ResponseWriter, r *http.Request) {
			GetExamProgressPageHandler(w, r, queries)
		})
	}
	beforeRename := loadSuccess()
	if beforeRename.Code != http.StatusOK || !strings.Contains(html.UnescapeString(beforeRename.Body.String()), wantURL) {
		t.Fatalf("success before rename status=%d body=%q, want URL %q", beforeRename.Code, beforeRename.Body.String(), wantURL)
	}

	if _, err := conn.Exec("UPDATE class_codes SET name='4e Alpha' WHERE id=1 AND user_id=1"); err != nil {
		t.Fatal(err)
	}
	afterRename := loadSuccess()
	afterBody := html.UnescapeString(afterRename.Body.String())
	if afterRename.Code != http.StatusOK || !strings.Contains(afterBody, wantURL) || !strings.Contains(afterBody, "4e Alpha") {
		t.Fatalf("success after rename status=%d body=%q, want old URL %q and current class label", afterRename.Code, afterRename.Body.String(), wantURL)
	}

	pdfResponse := serveAuthenticatedGenerationRequestMethod(t, http.MethodGet, wantURL, func(w http.ResponseWriter, r *http.Request) {
		ServeFullPdfExamHandler(w, r, queries)
	})
	if pdfResponse.Code != http.StatusOK || pdfResponse.Body.String() != historicalContent {
		t.Fatalf("historical PDF status=%d body=%q", pdfResponse.Code, pdfResponse.Body.String())
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != historicalName {
		t.Fatalf("workspace entries=%v, historical artifact was replaced or duplicated", entries)
	}
}

func TestSuccessWithMissingHistoricalPDFRendersUnavailablePageWithoutDBMutation(t *testing.T) {
	conn, queries := setupExamPreflightTest(t)
	const generationID int64 = 9_000_000_000_000_100
	if _, err := conn.Exec("INSERT INTO exams_generated(id,exam_id,total_students,status,user_id) VALUES(?,6,1,'success',1)", generationID); err != nil {
		t.Fatal(err)
	}

	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(current) })

	workspace := filepath.Join("assets", "tmp", "teacher", "exam-"+strconv.FormatInt(generationID, 10))
	if _, err := os.Lstat(workspace); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("test requires absent workspace %q, err=%v", workspace, err)
	}
	progressURL := data.DefaultGenerateExamRoutes.ProcessingStudents + "?exam_generated_id=" + strconv.FormatInt(generationID, 10)
	response := serveAuthenticatedGenerationRequestMethod(t, http.MethodGet, progressURL, func(w http.ResponseWriter, r *http.Request) {
		GetExamProgressPageHandler(w, r, queries)
	})
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q, want 200", response.Code, response.Body.String())
	}
	body := response.Body.String()
	for _, want := range []string{"Les fichiers PDF de cette génération ne sont plus disponibles.", "workspace failure", "1A", data.DefaultDashboardRoutes.ExamURL} {
		if !strings.Contains(body, want) {
			t.Fatalf("unavailable page missing %q: %s", want, body)
		}
	}
	var status string
	if err := conn.QueryRow("SELECT status FROM exams_generated WHERE id=? AND user_id=1", generationID).Scan(&status); err != nil || status != "success" {
		t.Fatalf("generation status=%q err=%v, want unchanged success", status, err)
	}
}

func TestSuccessWithUnexpectedPDFResolutionErrorReturnsInternalServerError(t *testing.T) {
	_, queries := setupExamPreflightTest(t)
	previousResolver := resolveExamGenerationPDFName
	resolveExamGenerationPDFName = func(string, int64) (string, error) {
		return "", errors.New("filesystem permission failure")
	}
	t.Cleanup(func() { resolveExamGenerationPDFName = previousResolver })

	response := serveAuthenticatedGenerationRequestMethod(t, http.MethodGet, data.DefaultGenerateExamRoutes.ProcessingStudents+"?exam_generated_id=99", func(w http.ResponseWriter, r *http.Request) {
		GetExamProgressPageHandler(w, r, queries)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%q, want 500", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "ne sont plus disponibles") {
		t.Fatal("unexpected filesystem error was presented as a missing historical PDF")
	}
}

func TestGenerationTemplatesRenderTypedData(t *testing.T) {
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(current) })

	t.Run("running", func(t *testing.T) {
		response := httptest.NewRecorder()
		RenderProcessingStudentsPage(response, buildExamGenerationProgressPageData(1, "running", db.GetExamGeneratedProgressRow{ProcessedStudents: 1, TotalStudents: 2}, "/progress?exam_generated_id=1"))
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d, want 200", response.Code)
		}
		body := response.Body.String()
		for _, want := range []string{"1 copies préparées sur 2", "/progress?exam_generated_id=1", "se met à jour automatiquement", "Retour aux évaluations"} {
			if !strings.Contains(body, want) {
				t.Fatalf("running render missing %q", want)
			}
		}
		for _, forbidden := range []string{"Annuler la génération", "Supprimer", "Modifier"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("running render exposes impossible action %q", forbidden)
			}
		}
	})
	t.Run("success", func(t *testing.T) {
		response := httptest.NewRecorder()
		RenderSuccessProcessing(response, buildExamGenerationSuccessPageData(1, db.GetExamNameAndClassCodeNameRow{ExamName: "Exam", ClassName: "Class"}, "historical-name.pdf"))
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d, want 200", response.Code)
		}
		body := response.Body.String()
		for _, want := range []string{"Exam", "Class", "Ouvrir le PDF des copies", data.DefaultDashboardRoutes.ExamURL, data.DefaultGenerateExamRoutes.PdfExam} {
			if !strings.Contains(body, want) {
				t.Fatalf("success render missing %q", want)
			}
		}
		if strings.Contains(body, "window.open") || strings.Contains(body, "GenerationID") {
			t.Fatal("success render contains automatic popup or technical generation ID")
		}
	})
	t.Run("unavailable", func(t *testing.T) {
		response := httptest.NewRecorder()
		RenderUnavailableExamPDF(response, buildExamGenerationUnavailablePageData(1, db.GetExamNameAndClassCodeNameRow{ExamName: "Exam", ClassName: "Class"}))
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d, want 200", response.Code)
		}
		body := response.Body.String()
		for _, want := range []string{"Copies indisponibles", "Exam", "Class", "Les fichiers PDF de cette génération ne sont plus disponibles.", data.DefaultDashboardRoutes.ExamURL} {
			if !strings.Contains(body, want) {
				t.Fatalf("unavailable render missing %q", want)
			}
		}
	})
}

func assertGenerationProgressURL(t *testing.T, rawURL string, generationID, examID int64) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != data.DefaultGenerateExamRoutes.ProcessingStudents ||
		parsed.Query().Get("exam_generated_id") != stringInt64(generationID) ||
		parsed.Query().Get("exam_id") != stringInt64(examID) ||
		parsed.Query().Get("generation_started") != "1" {
		t.Fatalf("ProgressURL=%q for generation=%d Exam=%d", rawURL, generationID, examID)
	}
}

func stringInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}
