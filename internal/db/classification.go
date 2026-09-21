package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/grapinou/LazyMarking/internal/classification"
)

type ClassificationResult struct {
	ID       int64
	Name     string
	Existing bool
}

// GetOrCreateClassification reuses a typographically equivalent personal label.
// The lowest ID is chosen deterministically if historical duplicates exist.
// Queries backed by a transaction participate in it; standalone calls start one
// so concurrent creators cannot both commit a new equivalent label.
func (q *Queries) GetOrCreateClassification(ctx context.Context, table, name string, userID int64) (ClassificationResult, error) {
	switch table {
	case "subjects", "themes", "year_levels", "skills", "difficulties":
	default:
		return ClassificationResult{}, errors.New("unsupported classification")
	}
	if classification.Key(name) == "" {
		return ClassificationResult{}, errors.New("classification name is empty")
	}
	if conn, ok := q.db.(*sql.DB); ok {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return ClassificationResult{}, err
		}
		defer tx.Rollback()
		result, err := q.WithTx(tx).GetOrCreateClassification(ctx, table, name, userID)
		if err != nil {
			return ClassificationResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return ClassificationResult{}, err
		}
		return result, nil
	}
	rows, err := q.db.QueryContext(ctx, "SELECT id,name FROM "+table+" WHERE user_id=? ORDER BY id", userID)
	if err != nil {
		return ClassificationResult{}, err
	}
	key := classification.Key(name)
	var match ClassificationResult
	for rows.Next() {
		var item ClassificationResult
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			rows.Close()
			return ClassificationResult{}, err
		}
		if match.ID == 0 && classification.Key(item.Name) == key {
			match = item
			match.Existing = true
		}
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil || closeErr != nil {
		return ClassificationResult{}, errors.Join(err, closeErr)
	}
	if match.Existing {
		return match, nil
	}
	result := ClassificationResult{Name: name}
	err = q.db.QueryRowContext(ctx, fmt.Sprintf("INSERT INTO %s(name,user_id) VALUES(?,?) RETURNING id", table), name, userID).Scan(&result.ID)
	return result, err
}
