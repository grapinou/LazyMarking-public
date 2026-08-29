package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/mattn/go-sqlite3"
)

func TestPedagogicalReferencesRestrictQuestionBackedDeletes(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		table      string
		questionFK string
		delete     func(*Queries, int64, int64) (int64, error)
	}{
		{name: "subject", table: "subjects", questionFK: "subject_id", delete: func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteSubject(ctx, DeleteSubjectParams{ID: id, UserID: userID})
		}},
		{name: "theme", table: "themes", questionFK: "theme_id", delete: func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteTheme(ctx, DeleteThemeParams{ID: id, UserID: userID})
		}},
		{name: "year level", table: "year_levels", questionFK: "year_level_id", delete: func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteYearLevel(ctx, DeleteYearLevelParams{ID: id, UserID: userID})
		}},
		{name: "skill", table: "skills", questionFK: "skill_id", delete: func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteSkill(ctx, DeleteSkillParams{ID: id, UserID: userID})
		}},
		{name: "difficulty", table: "difficulties", questionFK: "difficulty_id", delete: func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteDifficulty(ctx, DeleteDifficultyParams{ID: id, UserID: userID})
		}},
		{name: "point", table: "points", questionFK: "point_id", delete: func(q *Queries, id, userID int64) (int64, error) {
			return q.DeletePoint(ctx, DeletePointParams{ID: id, UserID: userID})
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, queries := newReferenceDeleteIntegrityDB(t)

			if rows, err := tc.delete(queries, 1, 1); err != nil || rows != 1 {
				t.Fatalf("free delete rows=%d err=%v, want 1 and nil", rows, err)
			}
			assertReferenceIntegrityCount(t, conn, tc.table, 1, 0)

			if rows, err := tc.delete(queries, 2, 1); err != nil || rows != 0 {
				t.Fatalf("foreign delete rows=%d err=%v, want 0 and nil", rows, err)
			}
			assertReferenceIntegrityCount(t, conn, tc.table, 2, 1)

			rows, err := tc.delete(queries, 3, 1)
			var sqliteError sqlite3.Error
			if err == nil || rows != 0 || !errors.As(err, &sqliteError) || sqliteError.Code != sqlite3.ErrConstraint {
				t.Fatalf("used delete rows=%d err=%T %v, want SQLite constraint", rows, err, err)
			}
			assertReferenceIntegrityCount(t, conn, tc.table, 3, 1)
			assertReferenceIntegrityQuestion(t, conn, tc.questionFK, 3)
		})
	}
}

func newReferenceDeleteIntegrityDB(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE themes(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE year_levels(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE skills(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE difficulties(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE points(id INTEGER PRIMARY KEY, point_value INTEGER NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE questions(
			id INTEGER PRIMARY KEY,
			subject_id INTEGER NOT NULL REFERENCES subjects(id) ON DELETE RESTRICT,
			theme_id INTEGER NOT NULL REFERENCES themes(id) ON DELETE RESTRICT,
			year_level_id INTEGER NOT NULL REFERENCES year_levels(id) ON DELETE RESTRICT,
			skill_id INTEGER NOT NULL REFERENCES skills(id) ON DELETE RESTRICT,
			difficulty_id INTEGER NOT NULL REFERENCES difficulties(id) ON DELETE RESTRICT,
			point_id INTEGER NOT NULL REFERENCES points(id) ON DELETE RESTRICT
		);
		INSERT INTO subjects VALUES(1,'free',1),(2,'foreign',2),(3,'used',1);
		INSERT INTO themes VALUES(1,'free',1),(2,'foreign',2),(3,'used',1);
		INSERT INTO year_levels VALUES(1,'free',1),(2,'foreign',2),(3,'used',1);
		INSERT INTO skills VALUES(1,'free',1),(2,'foreign',2),(3,'used',1);
		INSERT INTO difficulties VALUES(1,'free',1),(2,'foreign',2),(3,'used',1);
		INSERT INTO points VALUES(1,1,1),(2,2,2),(3,3,1);
		INSERT INTO questions VALUES(10,3,3,3,3,3,3);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, New(conn)
}

func assertReferenceIntegrityCount(t *testing.T, conn *sql.DB, table string, id int64, want int) {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE id=?", id).Scan(&count); err != nil || count != want {
		t.Fatalf("%s id=%d count=%d err=%v, want %d", table, id, count, err, want)
	}
}

func assertReferenceIntegrityQuestion(t *testing.T, conn *sql.DB, column string, want int64) {
	t.Helper()
	var got int64
	if err := conn.QueryRow("SELECT " + column + " FROM questions WHERE id=10").Scan(&got); err != nil || got != want {
		t.Fatalf("question %s=%d err=%v, want %d", column, got, err, want)
	}
}
