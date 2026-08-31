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

const StudentExamPageReferenceDPI int64 = 300

var (
	ErrStudentExamPageReferenceUnavailable = errors.New("student exam page reference unavailable")
	ErrStudentExamPageReferenceCorrupt     = errors.New("student exam page reference corrupt")
	ErrStudentExamPageReferenceUnsafe      = errors.New("student exam page reference unsafe")
)

type ResolvedStudentExamPageReference struct {
	Path          string
	GenerationID  int64
	StudentExamID int64
	Page          int64
	Width         int64
	Height        int64
	DPI           int64
	SHA256        string
}

// ResolveStudentExamPageReference resolves only a persisted native pre-QR PNG.
// It deliberately has no Typst, live-image, or historical-PDF fallback.
func ResolveStudentExamPageReference(
	ctx context.Context,
	queries *db.Queries,
	userID int64,
	username string,
	studentExamID int64,
	page int64,
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
		return ResolvedStudentExamPageReference{}, fmt.Errorf("load page reference: %w", err)
	}
	if row.Username != username {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnavailable
	}
	if !row.ReferenceStorageKey.Valid || !row.ReferenceWidth.Valid ||
		!row.ReferenceHeight.Valid || !row.ReferenceDpi.Valid || !row.ReferenceSha256.Valid {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnavailable
	}
	if row.ReferenceWidth.Int64 <= 0 || row.ReferenceHeight.Int64 <= 0 ||
		row.ReferenceDpi.Int64 != StudentExamPageReferenceDPI ||
		len(row.ReferenceSha256.String) != sha256.Size*2 {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceCorrupt
	}

	components, err := validatePageReferenceStorageKey(row.ReferenceStorageKey.String, row.StudentExamID, row.Page)
	if err != nil {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnsafe
	}
	operation := "exam-" + strconv.FormatInt(row.GenerationID, 10)
	workspace, err := operationTempDir(username, operation)
	if err != nil {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnsafe
	}
	if err := ensureDirectoryTree(workspace, false, 0); err != nil {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnavailable
	}
	parent := filepath.Join(append([]string{workspace}, components[:len(components)-1]...)...)
	if err := ensureDirectoryTree(parent, false, 0); err != nil {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnsafe
	}
	path := filepath.Join(append([]string{workspace}, components...)...)
	lstatInfo, err := os.Lstat(path)
	if err != nil {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnavailable
	}
	if lstatInfo.Mode()&os.ModeSymlink != 0 || !lstatInfo.Mode().IsRegular() {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnsafe
	}

	file, err := os.Open(path)
	if err != nil {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnavailable
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || !os.SameFile(lstatInfo, openedInfo) {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceUnsafe
	}

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceCorrupt
	}
	actualHash := hex.EncodeToString(digest.Sum(nil))
	if actualHash != row.ReferenceSha256.String {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceCorrupt
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceCorrupt
	}
	img, err := png.Decode(file)
	if err != nil {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceCorrupt
	}
	bounds := img.Bounds()
	if int64(bounds.Dx()) != row.ReferenceWidth.Int64 || int64(bounds.Dy()) != row.ReferenceHeight.Int64 {
		return ResolvedStudentExamPageReference{}, ErrStudentExamPageReferenceCorrupt
	}

	return ResolvedStudentExamPageReference{
		Path: path, GenerationID: row.GenerationID, StudentExamID: row.StudentExamID, Page: row.Page,
		Width: row.ReferenceWidth.Int64, Height: row.ReferenceHeight.Int64,
		DPI: row.ReferenceDpi.Int64, SHA256: row.ReferenceSha256.String,
	}, nil
}

func validatePageReferenceStorageKey(key string, studentExamID, page int64) ([]string, error) {
	if key == "" || filepath.IsAbs(key) || filepath.Clean(key) != key || filepath.ToSlash(key) != key || strings.ContainsRune(key, '\x00') {
		return nil, ErrStudentExamPageReferenceUnsafe
	}
	components := strings.Split(key, "/")
	if len(components) != 3 || components[0] != "references" {
		return nil, ErrStudentExamPageReferenceUnsafe
	}
	for _, component := range components {
		if err := safePathComponent(component); err != nil {
			return nil, ErrStudentExamPageReferenceUnsafe
		}
	}
	if !strings.EqualFold(filepath.Ext(components[len(components)-1]), ".png") {
		return nil, ErrStudentExamPageReferenceUnsafe
	}
	if components[1] != "student-exam-"+strconv.FormatInt(studentExamID, 10) ||
		components[2] != "page-"+strconv.FormatInt(page, 10)+".png" {
		return nil, ErrStudentExamPageReferenceUnsafe
	}
	return components, nil
}
