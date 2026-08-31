package tools

import (
	"context"
	"fmt"

	"github.com/grapinou/LazyMarking/internal/db"
)

func ValidateExamGenerationReferences(ctx context.Context, queries *db.Queries, userID int64, username string, generationID int64) error {
	coverage, err := queries.GetExamGenerationReferenceCoverage(ctx, db.GetExamGenerationReferenceCoverageParams{
		GenerationID: generationID,
		UserID:       userID,
	})
	if err != nil {
		return fmt.Errorf("load generation reference coverage: %w", err)
	}
	if coverage.ExpectedPages <= 0 || coverage.ReferencedPages != coverage.ExpectedPages || coverage.AmbiguousPages != 0 {
		return fmt.Errorf("generation reference coverage is incomplete or ambiguous")
	}
	pages, err := queries.ListExamGenerationPageReferences(ctx, db.ListExamGenerationPageReferencesParams{
		GenerationID: generationID,
		UserID:       userID,
	})
	if err != nil {
		return fmt.Errorf("list generation page references: %w", err)
	}
	if int64(len(pages)) != coverage.ExpectedPages {
		return fmt.Errorf("generation reference page list is incomplete")
	}
	seen := make(map[[2]int64]struct{}, len(pages))
	for _, page := range pages {
		identity := [2]int64{page.StudentExamID, page.Page}
		if _, exists := seen[identity]; exists {
			return fmt.Errorf("generation reference page list is ambiguous")
		}
		seen[identity] = struct{}{}
		if _, err := ResolveStudentExamPageReference(ctx, queries, userID, username, page.StudentExamID, page.Page); err != nil {
			return fmt.Errorf("validate generation page reference: %w", err)
		}
	}
	return nil
}
