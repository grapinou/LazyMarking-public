package generateexams

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
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
	}, "teacher")

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
	if copiesURL.Path != data.DefaultGenerateExamRoutes.PdfExam || copiesURL.Query().Get("operation") != "exam-42" || copiesURL.Query().Get("file") != "teacher_exam_Forces_3e A.pdf" {
		t.Fatalf("CopiesURL=%q", page.Success.CopiesURL)
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
		RenderSuccessProcessing(response, buildExamGenerationSuccessPageData(1, db.GetExamNameAndClassCodeNameRow{ExamName: "Exam", ClassName: "Class"}, "teacher"))
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
