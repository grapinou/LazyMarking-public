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
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
)

var (
	ErrMarkingAlignedPageUnavailable = errors.New("marking aligned page unavailable")
	ErrMarkingAlignedPageCorrupt     = errors.New("marking aligned page corrupt")
	ErrMarkingAlignedPageUnsafe      = errors.New("marking aligned page unsafe")
)

type ResolvedMarkingAlignedPage struct {
	Path          string
	CopyResultID  int64
	MarkingJobID  int64
	StudentExamID int64
	PageExam      int64
	Width         int64
	Height        int64
	SHA256        string
}

// StageMarkingAlignedPage preserves the exact non-annotated homography PNG
// before DrawMarking rewrites the working image in place.
func StageMarkingAlignedPage(tempDir string, studentExamID int64, pageExam int, sourceName string) (string, error) {
	if studentExamID <= 0 || pageExam < 1 || safePathComponent(sourceName) != nil {
		return "", ErrMarkingAlignedPageUnsafe
	}
	sourcePath, sourceInfo, err := validateRegularFile(tempDir, sourceName)
	if err != nil {
		return "", ErrMarkingAlignedPageUnsafe
	}
	stagingDir := filepath.Join(tempDir, ".aligned-staging", "student-exam-"+strconv.FormatInt(studentExamID, 10))
	if err := ensureDirectoryTree(stagingDir, true, 0o750); err != nil {
		return "", fmt.Errorf("create aligned page staging directory: %w", err)
	}
	destination := filepath.Join(stagingDir, "page-"+strconv.Itoa(pageExam)+".png")
	if err := copyRegularFileNoReplace(sourcePath, sourceInfo, destination); err != nil {
		return "", fmt.Errorf("stage aligned page: %w", err)
	}
	return destination, nil
}

func StoreMarkingAlignedPage(ctx context.Context, queries *db.Queries, userID int64, username string, markingJobID, copyResultID, studentExamID, pageExam int64, sourcePath string) (ResolvedMarkingAlignedPage, error) {
	if _, err := queries.ValidateMarkingAlignedPageTarget(ctx, db.ValidateMarkingAlignedPageTargetParams{
		CopyResultID: copyResultID, MarkingJobID: markingJobID, StudentExamID: studentExamID, UserID: userID, PageExam: pageExam,
	}); errors.Is(err, sql.ErrNoRows) {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnavailable
	} else if err != nil {
		return ResolvedMarkingAlignedPage{}, fmt.Errorf("validate aligned page target: %w", err)
	}

	workspace, err := operationTempDir(username, "marking-"+strconv.FormatInt(markingJobID, 10))
	if err != nil || ensureDirectoryTree(workspace, false, 0) != nil {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnsafe
	}
	expectedSourceDir := filepath.Join(workspace, ".aligned-staging", "student-exam-"+strconv.FormatInt(studentExamID, 10))
	if filepath.Clean(filepath.Dir(sourcePath)) != filepath.Clean(expectedSourceDir) {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnsafe
	}
	sourceInfo, err := os.Lstat(sourcePath)
	if err != nil || sourceInfo.Mode()&os.ModeSymlink != 0 || !sourceInfo.Mode().IsRegular() {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnsafe
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnavailable
	}
	defer source.Close()
	openedInfo, err := source.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || !os.SameFile(sourceInfo, openedInfo) {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnsafe
	}
	config, err := png.DecodeConfig(source)
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageCorrupt
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageCorrupt
	}

	storageKey := alignedPageStorageKey(studentExamID, pageExam)
	components, err := validateAlignedPageStorageKey(storageKey, studentExamID, pageExam)
	if err != nil {
		return ResolvedMarkingAlignedPage{}, err
	}
	destinationDir := filepath.Join(append([]string{workspace}, components[:len(components)-1]...)...)
	if err := ensureDirectoryTree(destinationDir, true, 0o750); err != nil {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnsafe
	}
	destinationPath := filepath.Join(append([]string{workspace}, components...)...)
	digest, err := publishOpenFileNoReplace(source, destinationPath)
	if err != nil {
		return ResolvedMarkingAlignedPage{}, err
	}

	if _, err := queries.CreateMarkingAlignedPage(ctx, db.CreateMarkingAlignedPageParams{
		UserID: userID, CopyResultID: copyResultID, PageExam: pageExam, StorageKey: storageKey,
		Width: int64(config.Width), Height: int64(config.Height), Sha256: digest,
	}); err != nil {
		return ResolvedMarkingAlignedPage{}, fmt.Errorf("attach aligned page metadata: %w", err)
	}
	return ResolveMarkingAlignedPage(ctx, queries, userID, username, markingJobID, copyResultID, studentExamID, pageExam)
}

func ResolveMarkingAlignedPage(ctx context.Context, queries *db.Queries, userID int64, username string, markingJobID, copyResultID, studentExamID, pageExam int64) (ResolvedMarkingAlignedPage, error) {
	row, err := queries.GetMarkingAlignedPage(ctx, db.GetMarkingAlignedPageParams{CopyResultID: copyResultID, PageExam: pageExam, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnavailable
	}
	if err != nil {
		return ResolvedMarkingAlignedPage{}, fmt.Errorf("load aligned page: %w", err)
	}
	if row.Username != username || row.MarkingJobID != markingJobID || row.StudentExamID != studentExamID {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnavailable
	}
	components, err := validateAlignedPageStorageKey(row.StorageKey, studentExamID, pageExam)
	if err != nil {
		return ResolvedMarkingAlignedPage{}, err
	}
	workspace, err := operationTempDir(username, "marking-"+strconv.FormatInt(markingJobID, 10))
	if err != nil || ensureDirectoryTree(workspace, false, 0) != nil {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnsafe
	}
	parent := filepath.Join(append([]string{workspace}, components[:len(components)-1]...)...)
	if ensureDirectoryTree(parent, false, 0) != nil {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnsafe
	}
	path := filepath.Join(append([]string{workspace}, components...)...)
	info, err := os.Lstat(path)
	if err != nil {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnavailable
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnsafe
	}
	file, err := os.Open(path)
	if err != nil {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnavailable
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageUnsafe
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageCorrupt
	}
	if hex.EncodeToString(hasher.Sum(nil)) != row.Sha256 {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageCorrupt
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageCorrupt
	}
	imageConfig, err := png.DecodeConfig(file)
	if err != nil || int64(imageConfig.Width) != row.Width || int64(imageConfig.Height) != row.Height {
		return ResolvedMarkingAlignedPage{}, ErrMarkingAlignedPageCorrupt
	}
	return ResolvedMarkingAlignedPage{Path: path, CopyResultID: copyResultID, MarkingJobID: markingJobID, StudentExamID: studentExamID, PageExam: pageExam, Width: row.Width, Height: row.Height, SHA256: row.Sha256}, nil
}

func alignedPageStorageKey(studentExamID, pageExam int64) string {
	return "aligned/student-exam-" + strconv.FormatInt(studentExamID, 10) + "/page-" + strconv.FormatInt(pageExam, 10) + ".png"
}

func validateAlignedPageStorageKey(key string, studentExamID, pageExam int64) ([]string, error) {
	if key == "" || filepath.IsAbs(key) || filepath.Clean(key) != key || filepath.ToSlash(key) != key || strings.ContainsRune(key, '\x00') {
		return nil, ErrMarkingAlignedPageUnsafe
	}
	parts := strings.Split(key, "/")
	if len(parts) != 3 || parts[0] != "aligned" || parts[1] != "student-exam-"+strconv.FormatInt(studentExamID, 10) || parts[2] != "page-"+strconv.FormatInt(pageExam, 10)+".png" {
		return nil, ErrMarkingAlignedPageUnsafe
	}
	for _, part := range parts {
		if safePathComponent(part) != nil {
			return nil, ErrMarkingAlignedPageUnsafe
		}
	}
	return parts, nil
}

func copyRegularFileNoReplace(sourcePath string, sourceInfo os.FileInfo, destination string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	opened, err := source.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(sourceInfo, opened) {
		return ErrMarkingAlignedPageUnsafe
	}
	_, err = publishOpenFileNoReplace(source, destination)
	return err
}

func publishOpenFileNoReplace(source *os.File, destination string) (string, error) {
	dir := filepath.Dir(destination)
	temporary, err := os.CreateTemp(dir, ".aligned-page-*.tmp")
	if err != nil {
		return "", err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = temporary.Close(); _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		return "", err
	}
	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(temporary, hasher), source); err != nil {
		return "", err
	}
	digest := hex.EncodeToString(hasher.Sum(nil))
	if err := temporary.Sync(); err != nil {
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	if existing, statErr := os.Lstat(destination); statErr == nil {
		if existing.Mode()&os.ModeSymlink != 0 || !existing.Mode().IsRegular() {
			return "", ErrMarkingAlignedPageUnsafe
		}
		existingHash, hashErr := hashRegularFile(destination, existing)
		if hashErr != nil || existingHash != digest {
			return "", fmt.Errorf("aligned page destination already contains different bytes")
		}
		return digest, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", statErr
	}
	if err := os.Link(temporaryPath, destination); err != nil {
		return "", err
	}
	if directory, err := os.Open(dir); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return digest, nil
}
