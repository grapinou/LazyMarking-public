package marking

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestBuildMarkingExamSummaryFrenchOrderAndMissingNames(t *testing.T) {
	rows := []db.ListCurrentExamResultsForGenerationRow{
		{StudentExamID: 9, MarkingJobID: 9, LastName: " ", FirstName: " Zoé "},
		{StudentExamID: 7, MarkingJobID: 7, LastName: "Zola", FirstName: "Alice"},
		{StudentExamID: 5, MarkingJobID: 5, LastName: "Éclair", FirstName: "Bob"},
		{StudentExamID: 4, MarkingJobID: 4, LastName: "E\u0301clair", FirstName: "Alice"},
		{StudentExamID: 3, MarkingJobID: 3, LastName: "éclair", FirstName: "alice"},
		{StudentExamID: 2, MarkingJobID: 2, LastName: " Dupont ", FirstName: ""},
		{StudentExamID: 1, MarkingJobID: 1, LastName: "Albert", FirstName: "Charlie"},
		{StudentExamID: 8, MarkingJobID: 8},
	}
	original := append([]db.ListCurrentExamResultsForGenerationRow(nil), rows...)
	summary := buildMarkingExamSummary(rows)
	var ids []int64
	for _, row := range summary.Results {
		ids = append(ids, row.SourceJobID)
	}
	if !reflect.DeepEqual(ids, []int64{1, 2, 3, 4, 5, 7, 8, 9}) {
		t.Fatalf("accent/case/empty/tie ordering: %v", ids)
	}
	if summary.Results[1].StudentName != "Dupont" || summary.Results[6].StudentName != "Élève sans nom" || summary.Results[7].StudentName != "Zoé" {
		t.Fatalf("missing-name rendering: %+v", summary.Results)
	}
	if !reflect.DeepEqual(rows, original) {
		t.Fatal("summary mutated query rows")
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	if !reflect.DeepEqual(summary, buildMarkingExamSummary(rows)) {
		t.Fatal("equal names must keep deterministic student ID order")
	}
}

func TestBuildMarkingExamSummaryHidesPendingScoreAndKeepsSourceJobs(t *testing.T) {
	summary := buildMarkingExamSummary([]db.ListCurrentExamResultsForMarkingJobRow{
		{FirstName: "Alice", LastName: "Alpha", MarkingJobID: 10, Outcome: "corrected", ScoreHalfUnits: sql.NullInt64{Int64: 15, Valid: true}, TotalPoints: sql.NullInt64{Int64: 10, Valid: true}},
		{FirstName: "Basile", LastName: "Beta", MarkingJobID: 10, Outcome: "corrected", ScoreHalfUnits: sql.NullInt64{Int64: 12, Valid: true}, TotalPoints: sql.NullInt64{Int64: 10, Valid: true}, PendingReviews: 1},
		{FirstName: "Chloé", LastName: "Gamma", MarkingJobID: 11, Outcome: "not_seen"},
	})
	if summary.Total != 3 || summary.Corrected != 1 || summary.PendingReview != 1 || summary.NotSeen != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	if !summary.Results[0].HasScore || summary.Results[0].ScoreLabel != "7,5 / 10" || summary.Results[0].SourceJobID != 10 {
		t.Fatalf("corrected result=%+v", summary.Results[0])
	}
	if summary.Results[1].HasScore || !summary.Results[1].Pending || summary.Results[1].ScoreLabel != "" {
		t.Fatalf("pending result exposes score: %+v", summary.Results[1])
	}
}
