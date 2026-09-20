package marking

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func loadMarkingGenerationList(ctx context.Context, queries *db.Queries, userID int64, generations []db.GetExamsGeneratedSuccessRow) ([]data.MarkingGenerationListView, error) {
	items := make([]data.MarkingGenerationListView, 0, len(generations))
	for _, generation := range generations {
		rows, err := queries.ListCurrentExamResultsForGeneration(ctx, db.ListCurrentExamResultsForGenerationParams{
			UserID: userID, GenerationID: generation.ExamGeneratedID,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, data.MarkingGenerationListView{
			ExamGeneratedID: generation.ExamGeneratedID,
			ExamName:        generation.ExamName,
			ClassCodeName:   generation.ClassCodeName,
			CreatedAt:       generation.CreatedAt,
			Progress:        buildMarkingProgress(buildMarkingExamSummary(rows)),
		})
	}
	return items, nil
}

func buildMarkingProgress(summary data.MarkingExamSummaryView) data.MarkingProgressView {
	progress := data.MarkingProgressView{StatusLabel: "Non corrigé", BadgeClass: "text-bg-secondary"}
	if summary.Corrected > 0 {
		progress.StatusLabel = "Correction partielle"
		progress.BadgeClass = "text-bg-warning"
	}
	if summary.Total > 0 && summary.Corrected == summary.Total {
		progress.StatusLabel = "Corrigé"
		progress.BadgeClass = "text-bg-success"
	}
	parts := make([]string, 0, 4)
	if summary.Corrected > 0 {
		parts = append(parts, markingCopyCount(summary.Corrected, "corrigée", "corrigées"))
	}
	if summary.PendingReview > 0 {
		parts = append(parts, markingCopyCount(summary.PendingReview, "à vérifier", "à vérifier"))
	}
	if summary.Issues > 0 {
		parts = append(parts, markingCopyCount(summary.Issues, "à contrôler", "à contrôler"))
	}
	if summary.NotSeen > 0 {
		parts = append(parts, markingCopyCount(summary.NotSeen, "sans correction finale", "sans correction finale"))
	}
	if len(parts) == 0 {
		progress.Detail = "Aucune copie finalisée"
	} else {
		progress.Detail = strings.Join(parts, " · ")
	}
	return progress
}

func markingCopyCount(count int64, singular, plural string) string {
	label := plural
	if count == 1 {
		label = singular
	}
	return fmt.Sprintf("%d %s", count, label)
}

func buildMarkingJobHistory(rows []db.ListRecentMarkingJobsRow) []data.MarkingJobHistoryView {
	jobs := make([]data.MarkingJobHistoryView, 0, len(rows))
	for _, row := range rows {
		source := row.SourcePdfFilename.String
		if source == "" {
			source = "Fichier non renseigné"
		}
		job := data.MarkingJobHistoryView{
			JobID: row.ID, ExamName: row.ExamName, ClassCodeName: row.ClassCodeName,
			SourceFilename: source,
		}
		if row.CompletedAt.Valid {
			job.CompletedLabel = row.CompletedAt.Time.Format("02/01/2006")
		}
		if row.ExamGeneratedID.Valid {
			job.ResultURL = markingGenerationURL(row.ExamGeneratedID.Int64)
		}
		jobID := url.QueryEscape(strconv.FormatInt(row.ID, 10))
		switch row.Status {
		case "running":
			job.StatusLabel = "Traitement en cours"
			job.Detail = "Le traitement peut être repris depuis cette page."
			job.ActionURL = data.DefaultMarkingRoutes.ProgressMarking + "?job_id=" + jobID
			job.ActionLabel = "Voir la progression"
		case "success":
			job.Detail = fmt.Sprintf("%d corrigée(s), %d à contrôler, %d non détectée(s)", row.CorrectedCopies, row.IssueCopies, row.NotSeenCopies)
			if row.PendingReviews > 0 {
				job.StatusLabel = "Vérification nécessaire"
				job.ActionURL = data.DefaultMarkingRoutes.ReviewURL + "?job_id=" + jobID
				job.ActionLabel = "Reprendre la vérification"
			} else {
				job.StatusLabel = "Résultat disponible"
				job.ActionURL = data.DefaultMarkingRoutes.SuccessURL + "?job_id=" + jobID + "&import=1"
				job.ActionLabel = "Documents de cet import"
			}
		default:
			job.StatusLabel = "Traitement interrompu"
			job.Detail = "Relancez un import avec le fichier source."
		}
		jobs = append(jobs, job)
	}
	return jobs
}

func buildMarkingExamSummary(rows []db.ListCurrentExamResultsForGenerationRow) data.MarkingExamSummaryView {
	// SQLite NOCASE only folds ASCII. French collation handles accents and
	// canonical Unicode forms here, once, for both the screen and its PDF.
	rows = append([]db.ListCurrentExamResultsForGenerationRow(nil), rows...)
	order := collate.New(language.French, collate.IgnoreCase)
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		an, bn := strings.TrimSpace(a.LastName), strings.TrimSpace(b.LastName)
		if (an == "") != (bn == "") {
			return an != ""
		}
		if comparison := order.CompareString(an, bn); comparison != 0 {
			return comparison < 0
		}
		if comparison := order.CompareString(strings.TrimSpace(a.FirstName), strings.TrimSpace(b.FirstName)); comparison != 0 {
			return comparison < 0
		}
		return a.StudentExamID < b.StudentExamID
	})
	summary := data.MarkingExamSummaryView{
		Total:   int64(len(rows)),
		Results: make([]data.MarkingExamResultView, 0, len(rows)),
	}
	for _, row := range rows {
		result := data.MarkingExamResultView{
			StudentName: strings.TrimSpace(strings.TrimSpace(row.LastName) + " " + strings.TrimSpace(row.FirstName)),
			SourceJobID: row.MarkingJobID,
		}
		if result.StudentName == "" {
			result.StudentName = "Élève sans nom"
		}
		switch row.Outcome {
		case "corrected":
			if row.PendingReviews > 0 {
				result.StatusLabel = "À vérifier"
				result.Pending = true
				summary.PendingReview++
			} else {
				result.StatusLabel = "Corrigée"
				result.HasScore = hasFinalMarkingScore(row)
				if result.HasScore {
					result.ScoreLabel = formatMarkingScore(row.ScoreHalfUnits.Int64, row.TotalPoints.Int64)
				}
				summary.Corrected++
			}
		case "incomplete":
			result.StatusLabel = "Copie incomplète"
			summary.Issues++
		case "error":
			result.StatusLabel = "Erreur de traitement"
			summary.Issues++
		default:
			result.StatusLabel = "Copie non détectée"
			summary.NotSeen++
		}
		summary.Results = append(summary.Results, result)
	}
	return summary
}

func hasFinalMarkingScore(row db.ListCurrentExamResultsForGenerationRow) bool {
	return row.Outcome == "corrected" && row.PendingReviews == 0 && row.ScoreHalfUnits.Valid && row.TotalPoints.Valid
}

func formatMarkingScore(halfUnits, totalPoints int64) string {
	if halfUnits%2 == 0 {
		return fmt.Sprintf("%d / %d", halfUnits/2, totalPoints)
	}
	return fmt.Sprintf("%d,5 / %d", halfUnits/2, totalPoints)
}
