package tools

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
)

var (
	ErrMarkingArtifactsRegeneration = errors.New("marking artifacts regeneration failed")
	ErrMarkingArtifactsConflict     = errors.New("marking artifacts revision conflict")
	ErrMarkingArtifactsUnavailable  = errors.New("marking artifacts regeneration unavailable")
)

type MarkingArtifactsRegenerationResult struct {
	Regenerated       bool
	ReviewRevision    int64
	ArtifactsRevision int64
}

type markingArtifactsQueries interface {
	GetMarkingArtifactsRegenerationTarget(context.Context, db.GetMarkingArtifactsRegenerationTargetParams) (db.GetMarkingArtifactsRegenerationTargetRow, error)
	AdvanceMarkingArtifactsRevision(context.Context, db.AdvanceMarkingArtifactsRevisionParams) (int64, error)
}

type markingArtifactsGenerator interface {
	Generate(context.Context, *db.Queries, MarkingArtifactsGenerationInput) (MarkingArtifactsGenerationOutput, error)
}

type MarkingArtifactsGenerationInput struct {
	UserID       int64
	Username     string
	MarkingJobID int64
	Workspace    string
	StagingDir   string
}

type MarkingArtifactsGenerationOutput struct {
	CorrectedPDF string
	MarkTablePDF string
}

// RegenerateMarkingArtifacts rebuilds both review-sensitive PDFs and publishes
// their shared revision only after both files have been safely installed.
func RegenerateMarkingArtifacts(ctx context.Context, queries *db.Queries, userID int64, username string, markingJobID int64) (MarkingArtifactsRegenerationResult, error) {
	return regenerateMarkingArtifacts(ctx, queries, queries, defaultMarkingArtifactsGenerator{}, userID, username, markingJobID)
}

func regenerateMarkingArtifacts(ctx context.Context, queries *db.Queries, revisionQueries markingArtifactsQueries, generator markingArtifactsGenerator, userID int64, username string, markingJobID int64) (result MarkingArtifactsRegenerationResult, err error) {
	if queries == nil || revisionQueries == nil || generator == nil || userID <= 0 || markingJobID <= 0 || safePathComponent(username) != nil {
		return result, ErrMarkingArtifactsUnavailable
	}
	target, err := revisionQueries.GetMarkingArtifactsRegenerationTarget(ctx, db.GetMarkingArtifactsRegenerationTargetParams{MarkingJobID: markingJobID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return result, ErrMarkingArtifactsUnavailable
	}
	if err != nil {
		return result, fmt.Errorf("%w: load target: %v", ErrMarkingArtifactsRegeneration, err)
	}
	result.ReviewRevision, result.ArtifactsRevision = target.ReviewRevision, target.ArtifactsRevision
	if target.ReviewPolicyVersion.Valid && target.PendingCandidates > 0 {
		return result, ErrMarkingArtifactsUnavailable
	}
	if target.ArtifactsRevision > target.ReviewRevision {
		return result, fmt.Errorf("%w: invalid revision order", ErrMarkingArtifactsRegeneration)
	}
	if target.ArtifactsRevision == target.ReviewRevision {
		return result, nil
	}
	if !target.AmbiguityDelta.Valid || !target.ExamName.Valid || !target.MarkTableName.Valid {
		return result, ErrMarkingArtifactsUnavailable
	}

	workspace, err := operationTempDir(username, "marking-"+strconv.FormatInt(markingJobID, 10))
	if err != nil || ensureDirectoryTree(workspace, false, 0) != nil {
		return result, ErrMarkingArtifactsUnavailable
	}
	correctedCanonical := filepath.Join(workspace, "corrected.pdf")
	markTableCanonical := filepath.Join(workspace, "mark-table.pdf")
	if filepath.Clean(target.ExamName.String) != filepath.Clean(correctedCanonical) || filepath.Clean(target.MarkTableName.String) != filepath.Clean(markTableCanonical) {
		return result, ErrMarkingArtifactsUnavailable
	}
	if err := validateCanonicalArtifact(correctedCanonical); err != nil {
		return result, err
	}
	if err := validateCanonicalArtifact(markTableCanonical); err != nil {
		return result, err
	}

	stagingDir, err := os.MkdirTemp(workspace, ".review-artifacts-*")
	if err != nil {
		return result, fmt.Errorf("%w: create staging directory", ErrMarkingArtifactsRegeneration)
	}
	defer func() { _ = os.RemoveAll(stagingDir) }()
	generated, err := generator.Generate(ctx, queries, MarkingArtifactsGenerationInput{UserID: userID, Username: username, MarkingJobID: markingJobID, Workspace: workspace, StagingDir: stagingDir})
	if err != nil {
		return result, fmt.Errorf("%w: generate: %v", ErrMarkingArtifactsRegeneration, err)
	}
	if err := validateGeneratedPDF(stagingDir, generated.CorrectedPDF); err != nil {
		return result, err
	}
	if err := validateGeneratedPDF(stagingDir, generated.MarkTablePDF); err != nil {
		return result, err
	}

	publication, err := publishMarkingArtifactPair(generated.CorrectedPDF, generated.MarkTablePDF, correctedCanonical, markTableCanonical)
	if err != nil {
		return result, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = publication.rollback()
		}
		publication.cleanup()
	}()
	rows, err := revisionQueries.AdvanceMarkingArtifactsRevision(ctx, db.AdvanceMarkingArtifactsRevisionParams{
		MarkingJobID: markingJobID, UserID: userID,
		ExpectedReviewRevision: target.ReviewRevision, ExpectedArtifactsRevision: target.ArtifactsRevision,
	})
	if err != nil {
		return result, fmt.Errorf("%w: publish revision: %v", ErrMarkingArtifactsRegeneration, err)
	}
	if rows != 1 {
		return result, ErrMarkingArtifactsConflict
	}
	committed = true
	result.Regenerated = true
	result.ArtifactsRevision = target.ReviewRevision
	return result, nil
}

func MarkingArtifactsCurrent(reviewRevision, artifactsRevision int64) bool {
	return reviewRevision >= 0 && artifactsRevision == reviewRevision
}

func validateCanonicalArtifact(path string) error {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return ErrMarkingArtifactsUnavailable
	}
	return nil
}

func validateGeneratedPDF(stagingDir, path string) error {
	cleanDir, cleanPath := filepath.Clean(stagingDir), filepath.Clean(path)
	rel, err := filepath.Rel(cleanDir, cleanPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%w: unsafe generated path", ErrMarkingArtifactsRegeneration)
	}
	info, err := os.Lstat(cleanPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() < 5 {
		return fmt.Errorf("%w: invalid generated PDF", ErrMarkingArtifactsRegeneration)
	}
	file, err := os.Open(cleanPath)
	if err != nil {
		return fmt.Errorf("%w: open generated PDF", ErrMarkingArtifactsRegeneration)
	}
	defer file.Close()
	header := make([]byte, 5)
	if _, err := file.Read(header); err != nil || string(header) != "%PDF-" {
		return fmt.Errorf("%w: unreadable generated PDF", ErrMarkingArtifactsRegeneration)
	}
	return nil
}

type artifactPairPublication struct {
	corrected, markTable             string
	correctedBackup, markTableBackup string
}

func publishMarkingArtifactPair(correctedSource, markTableSource, correctedDestination, markTableDestination string) (artifactPairPublication, error) {
	p := artifactPairPublication{corrected: correctedDestination, markTable: markTableDestination, correctedBackup: correctedDestination + ".review-backup", markTableBackup: markTableDestination + ".review-backup"}
	p.cleanup()
	if err := os.Rename(correctedDestination, p.correctedBackup); err != nil {
		return p, fmt.Errorf("%w: backup corrected PDF", ErrMarkingArtifactsRegeneration)
	}
	if err := os.Rename(markTableDestination, p.markTableBackup); err != nil {
		_ = os.Rename(p.correctedBackup, correctedDestination)
		return p, fmt.Errorf("%w: backup mark table PDF", ErrMarkingArtifactsRegeneration)
	}
	if err := os.Rename(correctedSource, correctedDestination); err != nil {
		_ = p.rollback()
		return p, fmt.Errorf("%w: publish corrected PDF", ErrMarkingArtifactsRegeneration)
	}
	if err := os.Rename(markTableSource, markTableDestination); err != nil {
		_ = p.rollback()
		return p, fmt.Errorf("%w: publish mark table PDF", ErrMarkingArtifactsRegeneration)
	}
	return p, nil
}

func (p artifactPairPublication) rollback() error {
	_ = os.Remove(p.corrected)
	_ = os.Remove(p.markTable)
	first := os.Rename(p.correctedBackup, p.corrected)
	second := os.Rename(p.markTableBackup, p.markTable)
	if first != nil {
		return first
	}
	return second
}

func (p artifactPairPublication) cleanup() {
	_ = os.Remove(p.correctedBackup)
	_ = os.Remove(p.markTableBackup)
}
