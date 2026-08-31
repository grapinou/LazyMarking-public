package tools

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
)

// StoreStudentExamPageReference copies the exact native pre-QR PNG bytes into
// durable generation storage and then attaches their metadata to the page row.
func StoreStudentExamPageReference(
	ctx context.Context,
	queries *db.Queries,
	userID int64,
	username string,
	generationID int64,
	studentExamID int64,
	page int64,
	sourcePath string,
) (ResolvedStudentExamPageReference, error) {
	row, err := queries.GetStudentExamPageReference(ctx, db.GetStudentExamPageReferenceParams{
		StudentExamID: studentExamID,
		Page:          page,
		UserID:        userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnavailable
	}
	if err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("load page before storing reference: %w", err)
	}
	if row.GenerationID != generationID || row.Username != username {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnavailable
	}

	operation := "exam-" + strconv.FormatInt(generationID, 10)
	workspace, err := operationTempDir(username, operation)
	if err != nil || filepath.Clean(filepath.Dir(sourcePath)) != filepath.Clean(workspace) {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnsafe
	}
	if err := ensureDirectoryTree(workspace, false, 0); err != nil {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnsafe
	}

	sourceInfo, err := os.Lstat(sourcePath)
	if err != nil || sourceInfo.Mode()&os.ModeSymlink != 0 || !sourceInfo.Mode().IsRegular() {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnsafe
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("open native page reference: %w", err)
	}
	defer source.Close()
	openedSourceInfo, err := source.Stat()
	if err != nil || !openedSourceInfo.Mode().IsRegular() || !os.SameFile(sourceInfo, openedSourceInfo) {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnsafe
	}
	config, err := png.DecodeConfig(source)
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceCorrupt
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("rewind native page reference: %w", err)
	}

	storageKey := pageReferenceStorageKey(studentExamID, page)
	components, err := validatePageReferenceStorageKey(storageKey, studentExamID, page)
	if err != nil {
		return ResolvedStudentExamPageReference{}, err
	}
	destinationDir := filepath.Join(append([]string{workspace}, components[:len(components)-1]...)...)
	if err := ensureDirectoryTree(destinationDir, true, 0o750); err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("create reference directory: %w", err)
	}
	destinationPath := filepath.Join(append([]string{workspace}, components...)...)

	temporary, err := os.CreateTemp(destinationDir, ".page-reference-*.tmp")
	if err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("create temporary page reference: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		_ = temporary.Close()
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("secure temporary page reference: %w", err)
	}
	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(temporary, hasher), source); err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("copy native page reference: %w", err)
	}
	digest := hex.EncodeToString(hasher.Sum(nil))
	if err := temporary.Sync(); err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("sync temporary page reference: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("close temporary page reference: %w", err)
	}

	if existingInfo, statErr := os.Lstat(destinationPath); statErr == nil {
		if existingInfo.Mode()&os.ModeSymlink != 0 || !existingInfo.Mode().IsRegular() {
			return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnsafe
		}
		existingDigest, hashErr := hashRegularFile(destinationPath, existingInfo)
		if hashErr != nil || existingDigest != digest {
			return ResolvedStudentExamPageReference{}, fmt.Errorf("page reference destination already contains different bytes")
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("inspect page reference destination: %w", statErr)
	} else {
		// A hard-link publishes the fully synced temporary inode atomically and,
		// unlike os.Rename, can never overwrite an existing historical file.
		if err := os.Link(temporaryPath, destinationPath); err != nil {
			return ResolvedStudentExamPageReference{}, fmt.Errorf("publish page reference atomically: %w", err)
		}
		if err := os.Remove(temporaryPath); err != nil {
			return ResolvedStudentExamPageReference{}, fmt.Errorf("remove published reference temporary name: %w", err)
		}
		removeTemporary = false
		if directory, openErr := os.Open(destinationDir); openErr == nil {
			_ = directory.Sync()
			_ = directory.Close()
		}
	}

	rows, err := queries.SetStudentExamPageReference(ctx, db.SetStudentExamPageReferenceParams{
		ReferenceStorageKey: sql.NullString{String: storageKey, Valid: true},
		ReferenceWidth:      sql.NullInt64{Int64: int64(config.Width), Valid: true},
		ReferenceHeight:     sql.NullInt64{Int64: int64(config.Height), Valid: true},
		ReferenceDpi:        sql.NullInt64{Int64: StudentExamPageReferenceDPI, Valid: true},
		ReferenceSha256:     sql.NullString{String: digest, Valid: true},
		StudentExamID:       studentExamID,
		Page:                page,
		UserID:              userID,
	})
	if err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("attach page reference metadata: %w", err)
	}
	if rows != 1 {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("attach page reference metadata: affected %d rows", rows)
	}
	resolved, err := ResolveStudentExamPageReference(ctx, queries, userID, username, studentExamID, page)
	if err != nil {
		return ResolvedStudentExamPageReference{}, fmt.Errorf("verify stored page reference: %w", err)
	}
	return resolved, nil
}

func pageReferenceStorageKey(studentExamID, page int64) string {
	return "references/student-exam-" + strconv.FormatInt(studentExamID, 10) + "/page-" + strconv.FormatInt(page, 10) + ".png"
}

func hashRegularFile(path string, lstatInfo os.FileInfo) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || !os.SameFile(lstatInfo, openedInfo) {
		return "", ErrStudentExamPageReferenceUnsafe
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
