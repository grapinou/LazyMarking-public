package sharedlibrary

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/db"
)

var ErrOwnFamily = errors.New("this question already belongs to you")

// CopyShared commits a complete private family, or rolls back all rows and
// removes newly created files. Source reads and destination writes share one
// SQLite transaction, including the publication check.
func CopyShared(ctx context.Context, conn *sql.DB, directory string, sourceID, userID int64) (id int64, err error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var files []string
	defer func() {
		if err != nil {
			for _, name := range files {
				removeErr := os.Remove(filepath.Join(directory, name))
				if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
					err = errors.Join(err, fmt.Errorf("remove copied image: %w", removeErr))
				}
			}
		}
	}()
	family, err := LoadShared(ctx, db.New(tx), sourceID)
	if err != nil {
		return 0, err
	}
	if family.Metadata.UserID == userID {
		return 0, ErrOwnFamily
	}
	id, err = cloneFamily(ctx, tx, directory, family, userID, &files)
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// cloneFamily is transaction-scoped so a future QCM copy can reuse it for all
// families and then insert the ordered composition before a single commit.
func cloneFamily(ctx context.Context, tx *sql.Tx, directory string, f Family, userID int64, files *[]string) (int64, error) {
	m := f.Metadata
	var features []int64
	for _, feature := range []struct {
		table, column string
		value         any
	}{
		{"subjects", "name", m.SubjectName}, {"themes", "name", m.ThemeName},
		{"year_levels", "name", m.YearLevelName}, {"skills", "name", m.SkillName},
		{"difficulties", "name", m.DifficultyName}, {"points", "point_value", m.PointValue},
	} {
		if feature.column == "name" {
			result, err := db.New(tx).GetOrCreateClassification(ctx, feature.table, feature.value.(string), userID)
			if err != nil {
				return 0, err
			}
			features = append(features, result.ID)
			continue
		}
		// Identifiers come exclusively from the constants above.
		_, err := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(%s,user_id) VALUES(?,?) ON CONFLICT(%s,user_id) DO NOTHING", feature.table, feature.column, feature.column), feature.value, userID)
		if err != nil {
			return 0, err
		}
		var id int64
		err = tx.QueryRowContext(ctx, fmt.Sprintf("SELECT id FROM %s WHERE %s=? AND user_id=?", feature.table, feature.column), feature.value, userID).Scan(&id)
		if err != nil {
			return 0, err
		}
		features = append(features, id)
	}
	var id int64
	err := tx.QueryRowContext(ctx, `INSERT INTO questions(subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,content,instruction,user_id)
	VALUES(?,?,?,?,?,?,?,?,?) RETURNING id`, features[0], features[1], features[2], features[3], features[4], features[5], m.Content, m.Instruction, userID).Scan(&id)
	if err != nil {
		return 0, err
	}
	for i, v := range f.Versions {
		parentID := id
		answerTable, imageTable, parentColumn := "answers", "images", "question_id"
		if i > 0 {
			err := tx.QueryRowContext(ctx, `INSERT INTO alt_questions(question_id,content,user_id) VALUES(?,?,?) RETURNING id`, id, v.Content, userID).Scan(&parentID)
			if err != nil {
				return 0, err
			}
			answerTable, imageTable, parentColumn = "alt_answers", "alt_images", "alt_question_id"
		}
		for _, a := range v.Answers {
			_, err := tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(%s,content,state,user_id) VALUES(?,?,?,?)", answerTable, parentColumn), parentID, a.Content, a.State, userID)
			if err != nil {
				return 0, err
			}
		}
		if v.Image != nil {
			name, err := copyImage(directory, v.Image.Name)
			if err != nil {
				return 0, fmt.Errorf("copy family image: %w", err)
			}
			*files = append(*files, name)
			_, err = tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(%s,image_name,resize_percentage,user_id) VALUES(?,?,?,?)", imageTable, parentColumn), parentID, name, v.Image.Resize, userID)
			if err != nil {
				return 0, err
			}
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO question_copy_origins(question_id,source_question_id,source_author) VALUES(?,?,?)`, id, m.ID, m.Author)
	return id, err
}
