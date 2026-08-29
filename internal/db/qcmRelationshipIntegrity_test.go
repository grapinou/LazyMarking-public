package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestOwnedQCMMutationRows(t *testing.T) {
	conn, queries := newQCMRelationshipTestDB(t)
	ctx := context.Background()

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.UpdateQCM(ctx, UpdateQCMParams{Name: "updated", ID: 3, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateQCM(ctx, UpdateQCMParams{Name: "missing", ID: 999, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateQCM(ctx, UpdateQCMParams{Name: "forged", ID: 2, UserID: 1})
	})
	var foreignName string
	if err := conn.QueryRow("SELECT name FROM qcm WHERE id = 2 AND user_id = 2").Scan(&foreignName); err != nil || foreignName != "foreign" {
		t.Fatalf("foreign QCM name = %q, err = %v; want unchanged", foreignName, err)
	}

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.DeleteQCM(ctx, DeleteQCMParams{ID: 3, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.DeleteQCM(ctx, DeleteQCMParams{ID: 999, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.DeleteQCM(ctx, DeleteQCMParams{ID: 2, UserID: 1})
	})
}

func TestOwnedQCMQuestionMutationRows(t *testing.T) {
	conn, queries := newQCMRelationshipTestDB(t)
	ctx := context.Background()

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.CreateQCMQuestion(ctx, CreateQCMQuestionParams{QcmID: 3, QuestionID: 10, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.CreateQCMQuestion(ctx, CreateQCMQuestionParams{QcmID: 3, QuestionID: 20, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.CreateQCMQuestion(ctx, CreateQCMQuestionParams{QcmID: 2, QuestionID: 10, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.CreateQCMQuestion(ctx, CreateQCMQuestionParams{QcmID: 3, QuestionID: 10, UserID: 2})
	})
	if rows, err := queries.CreateQCMQuestion(ctx, CreateQCMQuestionParams{QcmID: 3, QuestionID: 10, UserID: 1}); err == nil || rows != 0 {
		t.Fatalf("duplicate membership rows = %d, err = %v; want constraint error", rows, err)
	}

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.DeleteQCMQuestion(ctx, DeleteQCMQuestionParams{ID: 100, QcmID: 1, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.DeleteQCMQuestion(ctx, DeleteQCMQuestionParams{ID: 100, QcmID: 1, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.DeleteQCMQuestion(ctx, DeleteQCMQuestionParams{ID: 200, QcmID: 2, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.DeleteQCMQuestion(ctx, DeleteQCMQuestionParams{ID: 200, QcmID: 1, UserID: 2})
	})
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM qcm_questions WHERE id = 200 AND user_id = 2").Scan(&count); err != nil || count != 1 {
		t.Fatalf("foreign relation count = %d, err = %v; want 1", count, err)
	}
}

func TestQCMRelationReadsRequireOwnedParent(t *testing.T) {
	_, queries := newQCMRelationshipTestDB(t)
	ctx := context.Background()
	rows, err := queries.GetAllQuestionsByQCMID(ctx, GetAllQuestionsByQCMIDParams{UserID: 1, QcmID: 2})
	if err != nil || len(rows) != 0 {
		t.Fatalf("foreign QCM questions = %d rows, err = %v; want none", len(rows), err)
	}
	ids, err := queries.GetQCMQuestionsIDs(ctx, GetQCMQuestionsIDsParams{UserID: 1, QcmID: 2})
	if err != nil || len(ids) != 0 {
		t.Fatalf("foreign QCM question IDs = %v, err = %v; want none", ids, err)
	}
	_, err = queries.GetQuestionContentByQCMQuestionID(ctx, GetQuestionContentByQCMQuestionIDParams{UserID: 1, QcmQuestionID: 200, QcmID: 2})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("foreign relation content error = %v, want sql.ErrNoRows", err)
	}
}

func TestQCMDeletionPreservesForeignKeyRestrictions(t *testing.T) {
	conn, queries := newQCMRelationshipTestDB(t)
	if rows, err := queries.DeleteQCM(context.Background(), DeleteQCMParams{ID: 1, UserID: 1}); err == nil || rows != 0 {
		t.Fatalf("referenced QCM delete rows = %d, err = %v; want FK error", rows, err)
	}
	if _, err := conn.Exec("DELETE FROM questions WHERE id = 10 AND user_id = 1"); err == nil {
		t.Fatal("referenced question deletion succeeded, want FK error")
	}
}

func newQCMRelationshipTestDB(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE qcm (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL, UNIQUE(name, user_id));
		CREATE TABLE questions (id INTEGER PRIMARY KEY, content TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE qcm_questions (
			id INTEGER PRIMARY KEY, qcm_id INTEGER NOT NULL REFERENCES qcm(id),
			question_id INTEGER NOT NULL REFERENCES questions(id) ON DELETE RESTRICT,
			user_id INTEGER NOT NULL, position INTEGER NOT NULL CHECK(position >= 1),
			UNIQUE(qcm_id, question_id), UNIQUE(qcm_id, position)
		);
		INSERT INTO qcm VALUES (1, 'owned', 1), (2, 'foreign', 2), (3, 'mutable', 1);
		INSERT INTO questions VALUES (10, 'owned content', 1), (11, 'owned second', 1), (12, 'owned third', 1), (20, 'foreign content', 2);
		INSERT INTO qcm_questions VALUES (100, 1, 10, 1, 1), (200, 2, 20, 2, 1);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, New(conn)
}
