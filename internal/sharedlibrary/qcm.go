package sharedlibrary

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/grapinou/LazyMarking/internal/db"
)

var ErrOwnQCM = errors.New("this QCM already belongs to you")

type PositionedFamily struct {
	Position int64
	Family   Family
}

type QCM struct {
	Metadata db.GetSharedQCMRow
	Families []PositionedFamily
}

// LoadSharedQCMFamily grants reading only through a published QCM containing
// this family, with matching ownership. It never requires or creates a family
// publication. Callers keep this check and child reads in the same transaction.
func LoadSharedQCMFamily(ctx context.Context, q *db.Queries, qcmID, questionID int64) (Family, error) {
	meta, err := q.GetSharedQCMFamily(ctx, db.GetSharedQCMFamilyParams{QcmID: qcmID, QuestionID: questionID})
	if err != nil {
		return Family{}, err
	}
	return loadFamily(ctx, q, db.GetSharedQuestionRow(meta))
}

// LoadSharedQCM reads the current composition and all versions inside the
// caller's transaction. A missing or inconsistent family aborts the whole read.
func LoadSharedQCM(ctx context.Context, q *db.Queries, id int64) (QCM, error) {
	meta, err := q.GetSharedQCM(ctx, id)
	if err != nil {
		return QCM{}, err
	}
	composition, err := q.GetSharedQCMComposition(ctx, id)
	if err != nil {
		return QCM{}, err
	}
	if int64(len(composition)) != meta.QuestionCount {
		return QCM{}, fmt.Errorf("inconsistent shared QCM composition")
	}
	result := QCM{Metadata: meta}
	for _, row := range composition {
		family, err := LoadSharedQCMFamily(ctx, q, id, row.QuestionID)
		if err != nil {
			return QCM{}, err
		}
		result.Families = append(result.Families, PositionedFamily{row.Position, family})
	}
	return result, nil
}

// CopySharedQCM uses ONE transaction for the publication check, source reads,
// new QCM, all families, ordered links and provenance. Files created by any
// family are removed if a later copy, insert or commit fails.
func CopySharedQCM(ctx context.Context, conn *sql.DB, directory string, sourceID, userID int64) (id int64, err error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var files []string
	defer func() {
		if err != nil {
			err = cleanupCopiedImages(directory, files, err)
		}
	}()
	source, err := LoadSharedQCM(ctx, db.New(tx), sourceID)
	if err != nil {
		return 0, err
	}
	if source.Metadata.UserID == userID {
		return 0, ErrOwnQCM
	}
	id, err = cloneQCM(ctx, tx, directory, source, userID, &files)
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func cloneQCM(ctx context.Context, tx *sql.Tx, directory string, source QCM, userID int64, files *[]string) (int64, error) {
	name := source.Metadata.Name
	for suffix := 1; ; suffix++ {
		var exists bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM qcm WHERE user_id=? AND name=?)", userID, name).Scan(&exists); err != nil {
			return 0, err
		}
		if !exists {
			break
		}
		name = source.Metadata.Name + " — copie"
		if suffix > 1 {
			name += fmt.Sprintf(" %d", suffix)
		}
	}
	var id int64
	if err := tx.QueryRowContext(ctx, "INSERT INTO qcm(name,user_id) VALUES(?,?) RETURNING id", name, userID).Scan(&id); err != nil {
		return 0, err
	}
	// The current UNIQUE(qcm_id,question_id) forbids duplicates. Keep an ID map
	// so each source family is cloned once even if composition later evolves.
	clones := make(map[int64]int64)
	for _, entry := range source.Families {
		sourceID := entry.Family.Metadata.ID
		newID, exists := clones[sourceID]
		if !exists {
			var err error
			newID, err = cloneFamily(ctx, tx, directory, entry.Family, userID, files)
			if err != nil {
				return 0, err
			}
			clones[sourceID] = newID
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(?,?,?,?)", id, newID, userID, entry.Position); err != nil {
			return 0, err
		}
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO qcm_copy_origins(qcm_id,source_qcm_id,source_author) VALUES(?,?,?)", id, source.Metadata.ID, source.Metadata.Author)
	return id, err
}
