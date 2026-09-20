package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestMarkingGenerationPDFEmptyAndMultipage(t *testing.T) {
	for _, binary := range []string{"typst", "pdftotext"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("%s unavailable: cannot compile/extract cumulative PDF", binary)
		}
	}
	t.Chdir("../../..")
	username := "bilan-test-" + uuid.NewString()
	userDir := filepath.Join("assets", "tmp", username)
	t.Cleanup(func() { _ = os.RemoveAll(userDir) })
	page := data.MarkingGenerationPageData{GenerationID: 42, ExamName: `Contrôle "A" #panic("unsafe")`, ClassName: "Seconde"}
	render := func() string {
		t.Helper()
		pdf, err := BuildMarkingGenerationPDF(t.Context(), username, page)
		if err != nil {
			t.Fatal(err)
		}
		command := exec.Command("pdftotext", "-layout", "-", "-")
		command.Stdin = bytes.NewReader(pdf)
		text, err := command.Output()
		if err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(userDir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("temporary report not cleaned: entries=%v err=%v", entries, err)
		}
		return string(text)
	}
	if text := render(); !strings.Contains(text, "Bilan de l’évaluation") || !strings.Contains(text, "Seconde") || !strings.Contains(text, `Contrôle "A" #panic("unsafe")`) || !strings.Contains(text, "Aucun résultat disponible") {
		t.Fatalf("empty report or literal escaping: %s", text)
	}
	for i := range 100 {
		page.Summary.Results = append(page.Summary.Results, data.MarkingExamResultView{
			StudentName: fmt.Sprintf("Élève %03d", i), StatusLabel: "Corrigée", HasScore: true, ScoreLabel: "13 / 20", SourceJobID: 8,
		})
	}
	page.Summary.Total, page.Summary.Corrected = 100, 100
	page.Pedagogy = data.MarkingPedagogicalSummaryView{
		IncludedCopies: 100, DetailedCopies: 100,
		ScoreGroups: []data.MarkingScoreStatisticsView{{Count: 100, Total: 20, Mean: "13,00", Median: "13,00", StdDev: "0,00"}},
		Skills:      []data.MarkingSuccessRateView{{Label: `Calculer #panic("unsafe")`, Success: "65,00"}},
	}
	for i := range 60 {
		page.Pedagogy.Questions = append(page.Pedagogy.Questions, data.MarkingQuestionStatisticsView{
			Label: fmt.Sprintf("Question %03d — Calculer un résultat", i), Count: 100, Correct: 50, Success: "65,00",
		})
	}
	text := render()
	if strings.Count(strings.Join(strings.Fields(text), " "), "Question / version") < 2 {
		t.Fatal("multipage question table does not repeat its header")
	}
	for i := range 60 {
		if !strings.Contains(text, fmt.Sprintf("Question %03d", i)) {
			t.Fatalf("missing question %d", i)
		}
	}
	if strings.Count(text, "État courant") < 2 {
		t.Fatal("multipage table does not repeat its header")
	}
	for i := range 100 {
		if !strings.Contains(text, fmt.Sprintf("Élève %03d", i)) {
			t.Fatalf("missing student %d", i)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := BuildMarkingGenerationPDF(ctx, username, page); err == nil {
		t.Fatal("cancelled download should abort rendering")
	}
	entries, err := os.ReadDir(userDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("cancelled report not cleaned: entries=%v err=%v", entries, err)
	}
}
