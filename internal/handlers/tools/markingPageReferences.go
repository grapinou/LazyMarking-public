package tools

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

type legacyMarkingReferenceRenderer func() ([]string, []string, error)

// resolveMarkingPageReferences returns one reference path for each sorted page
// number. Persisted metadata always wins: resolver failures never fall back to
// the live Typst path. Only an owned page whose five metadata fields are all
// NULL is explicitly legacy.
func resolveMarkingPageReferences(
	ctx context.Context,
	queries *db.Queries,
	userID int64,
	username string,
	studentExamID int64,
	pageNumbers []int,
	renderLegacy legacyMarkingReferenceRenderer,
) ([]string, []string, error) {
	references := make([]string, len(pageNumbers))
	legacyIndexes := make([]int, 0)

	for index, pageNumber := range pageNumbers {
		resolved, err := ResolveStudentExamPageReference(ctx, queries, userID, username, studentExamID, int64(pageNumber))
		if err == nil {
			absolutePath, absoluteErr := filepath.Abs(resolved.Path)
			if absoluteErr != nil {
				return nil, nil, fmt.Errorf("resolve absolute historical reference path for page %d: %w", pageNumber, absoluteErr)
			}
			references[index] = absolutePath
			continue
		}
		if !errors.Is(err, ErrStudentExamPageReferenceUnavailable) {
			return nil, nil, fmt.Errorf("resolve historical reference for page %d: %w", pageNumber, err)
		}

		metadata, metadataErr := queries.GetStudentExamPageReference(ctx, db.GetStudentExamPageReferenceParams{
			StudentExamID: studentExamID,
			Page:          int64(pageNumber),
			UserID:        userID,
		})
		if metadataErr != nil {
			if errors.Is(metadataErr, sql.ErrNoRows) {
				return nil, nil, fmt.Errorf("load historical reference metadata for page %d: %w", pageNumber, ErrStudentExamPageReferenceUnavailable)
			}
			return nil, nil, fmt.Errorf("load historical reference metadata for page %d: %w", pageNumber, metadataErr)
		}
		allNull := !metadata.ReferenceStorageKey.Valid && !metadata.ReferenceWidth.Valid &&
			!metadata.ReferenceHeight.Valid && !metadata.ReferenceDpi.Valid && !metadata.ReferenceSha256.Valid
		if !allNull {
			return nil, nil, fmt.Errorf("historical reference for page %d is incomplete or corrupt", pageNumber)
		}
		legacyIndexes = append(legacyIndexes, index)
	}

	if len(legacyIndexes) == 0 {
		return references, nil, nil
	}
	legacyPages, cleanup, err := renderLegacy()
	if err != nil {
		return nil, nil, err
	}
	if len(legacyPages) != len(pageNumbers) {
		return nil, nil, fmt.Errorf("legacy reference page count is %d, want %d", len(legacyPages), len(pageNumbers))
	}
	for _, index := range legacyIndexes {
		absolutePath, absoluteErr := filepath.Abs(legacyPages[index])
		if absoluteErr != nil {
			return nil, nil, fmt.Errorf("resolve absolute legacy reference path for page %d: %w", pageNumbers[index], absoluteErr)
		}
		references[index] = absolutePath
	}
	for _, reference := range references {
		if reference == "" || !filepath.IsAbs(reference) {
			return nil, nil, fmt.Errorf("resolved marking reference is not an absolute path")
		}
	}
	return references, cleanup, nil
}

func renderLegacyMarkingReferences(tempDir, username string, qcm config.QCM) ([]string, []string, error) {
	typstFilePath, ok := TypstWriter(tempDir, username, qcm, config.ExamQCM)
	if !ok {
		return nil, nil, ErrMarkingStudentExam
	}
	pages, ok := ExportTypstToPNGs(typstFilePath)
	if !ok {
		return nil, nil, ErrMarkingStudentExam
	}
	cleanup := append([]string{typstFilePath}, pages...)
	return pages, cleanup, nil
}
