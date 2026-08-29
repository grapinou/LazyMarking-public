package db

import (
	"database/sql"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestQCMDeleteCascadeMigrationUpPreservesIntegrityAndDeleteSemantics(t *testing.T) {
	conn := newPreQCMDeleteCascadeMigrationDB(t)
	up, _ := readQCMDeleteCascadeMigration(t)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("apply 0033 Up: %v", err)
	}

	wantRelations := [][5]int64{
		{100, 1, 10, 1, 1},
		{101, 1, 11, 1, 2},
		{200, 2, 10, 1, 1},
		{500, 5, 12, 1, 1},
	}
	assertQCMRelations(t, conn, wantRelations)
	assertQCMQuestionForeignKeys(t, conn, "CASCADE")
	assertQCMQuestionOwnershipTriggers(t, conn)

	assertSQLRejected(t, conn, "position below one after Up", "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(2,12,1,0)")
	assertSQLRejected(t, conn, "duplicate position after Up", "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(1,12,1,1)")
	assertSQLRejected(t, conn, "duplicate question after Up", "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(1,10,1,3)")
	assertSQLRejected(t, conn, "question remains restricted", "DELETE FROM questions WHERE id=11")
	assertSQLRejected(t, conn, "foreign QCM ownership after Up", "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(3,12,1,1)")
	assertSQLRejected(t, conn, "forged ownership update after Up", "UPDATE qcm_questions SET qcm_id=3 WHERE id=100")

	if result, err := conn.Exec("DELETE FROM qcm WHERE id=1 AND user_id=1"); err != nil {
		t.Fatalf("delete composed QCM: %v", err)
	} else if rows, _ := result.RowsAffected(); rows != 1 {
		t.Fatalf("deleted QCM rows = %d, want 1", rows)
	}
	assertQCMDeleteTableCount(t, conn, "qcm", "id=1", 0)
	assertQCMDeleteTableCount(t, conn, "qcm_questions", "qcm_id=1", 0)
	assertQCMDeleteTableCount(t, conn, "qcm", "id=2", 1)
	assertQCMDeleteTableCount(t, conn, "qcm_questions", "qcm_id=2 AND question_id=10", 1)
	assertQCMDeleteTableCount(t, conn, "questions", "id IN (10,11,12)", 3)

	if _, err := conn.Exec("DELETE FROM qcm WHERE id=4 AND user_id=1"); err != nil {
		t.Fatalf("delete empty QCM: %v", err)
	}
	assertQCMDeleteTableCount(t, conn, "qcm", "id=4", 0)

	if _, err := conn.Exec("DELETE FROM qcm WHERE id=5 AND user_id=1"); err == nil {
		t.Fatal("exam-protected QCM deletion succeeded")
	}
	assertQCMDeleteTableCount(t, conn, "qcm", "id=5", 1)
	assertQCMDeleteTableCount(t, conn, "qcm_questions", "id=500 AND qcm_id=5", 1)
	assertQCMDeleteTableCount(t, conn, "exams", "id=50 AND qcm_id=5", 1)

	if result, err := conn.Exec("DELETE FROM qcm WHERE id=3 AND user_id=1"); err != nil {
		t.Fatal(err)
	} else if rows, _ := result.RowsAffected(); rows != 0 {
		t.Fatalf("foreign QCM deleted rows = %d, want 0", rows)
	}
}

func TestQCMDeleteCascadeMigrationDownRestoresPreviousContract(t *testing.T) {
	conn := newPreQCMDeleteCascadeMigrationDB(t)
	up, down := readQCMDeleteCascadeMigration(t)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("apply 0033 Up: %v", err)
	}
	if _, err := conn.Exec(down); err != nil {
		t.Fatalf("apply 0033 Down: %v", err)
	}

	wantRelations := [][5]int64{
		{100, 1, 10, 1, 1},
		{101, 1, 11, 1, 2},
		{200, 2, 10, 1, 1},
		{500, 5, 12, 1, 1},
	}
	assertQCMRelations(t, conn, wantRelations)
	assertQCMQuestionForeignKeys(t, conn, "NO ACTION")
	assertQCMQuestionOwnershipTriggers(t, conn)
	assertSQLRejected(t, conn, "QCM delete restricted after Down", "DELETE FROM qcm WHERE id=1")
	assertSQLRejected(t, conn, "question delete restricted after Down", "DELETE FROM questions WHERE id=11")
	assertSQLRejected(t, conn, "foreign ownership after Down", "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(3,12,1,1)")
	assertSQLRejected(t, conn, "forged update after Down", "UPDATE qcm_questions SET qcm_id=3 WHERE id=100")
}

func newPreQCMDeleteCascadeMigrationDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY);
		CREATE TABLE qcm(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE questions(id INTEGER PRIMARY KEY, content TEXT NOT NULL, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE qcm_questions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			qcm_id INTEGER NOT NULL REFERENCES qcm(id),
			question_id INTEGER NOT NULL REFERENCES questions(id) ON DELETE RESTRICT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			position INTEGER NOT NULL CHECK(position >= 1),
			UNIQUE(qcm_id,question_id), UNIQUE(qcm_id,position)
		);
		CREATE TABLE exams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			qcm_id INTEGER NOT NULL REFERENCES qcm(id) ON DELETE RESTRICT,
			user_id INTEGER NOT NULL REFERENCES users(id)
		);
		CREATE TRIGGER qcm_questions_owner_insert BEFORE INSERT ON qcm_questions
		WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id=NEW.qcm_id AND user_id=NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM questions WHERE id=NEW.question_id AND user_id=NEW.user_id)
		BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;
		CREATE TRIGGER qcm_questions_owner_update BEFORE UPDATE OF qcm_id,question_id,user_id ON qcm_questions
		WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id=NEW.qcm_id AND user_id=NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM questions WHERE id=NEW.question_id AND user_id=NEW.user_id)
		BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;
		INSERT INTO users VALUES(1),(2);
		INSERT INTO qcm VALUES(1,'composed',1),(2,'shared question',1),(3,'foreign',2),(4,'empty',1),(5,'protected',1);
		INSERT INTO questions VALUES(10,'shared',1),(11,'owned',1),(12,'protected',1),(20,'foreign',2);
		INSERT INTO qcm_questions(id,qcm_id,question_id,user_id,position) VALUES
			(100,1,10,1,1),(101,1,11,1,2),(200,2,10,1,1),(500,5,12,1,1);
		INSERT INTO exams(id,qcm_id,user_id) VALUES(50,5,1);
	`); err != nil {
		t.Fatalf("prepare pre-0033 schema: %v", err)
	}
	return conn
}

func readQCMDeleteCascadeMigration(t *testing.T) (string, string) {
	t.Helper()
	migration, err := os.ReadFile("../../db/migrations/0033_cascade_qcm_composition_delete.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(migration), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("0033 migration has no Down section")
	}
	return strings.Replace(parts[0], "-- +goose Up", "", 1), parts[1]
}

func assertQCMRelations(t *testing.T, conn *sql.DB, want [][5]int64) {
	t.Helper()
	rows, err := conn.Query("SELECT id,qcm_id,question_id,user_id,position FROM qcm_questions ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got [][5]int64
	for rows.Next() {
		var row [5]int64
		if err := rows.Scan(&row[0], &row[1], &row[2], &row[3], &row[4]); err != nil {
			t.Fatal(err)
		}
		got = append(got, row)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("relations = %v, want %v", got, want)
	}
}

func assertQCMQuestionForeignKeys(t *testing.T, conn *sql.DB, wantQCMDelete string) {
	t.Helper()
	rows, err := conn.Query("PRAGMA foreign_key_list(qcm_questions)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	foundQCM, foundQuestion := false, false
	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatal(err)
		}
		switch table {
		case "qcm":
			foundQCM = true
			if onDelete != wantQCMDelete {
				t.Fatalf("qcm ON DELETE = %q, want %q", onDelete, wantQCMDelete)
			}
		case "questions":
			foundQuestion = true
			if onDelete != "RESTRICT" {
				t.Fatalf("questions ON DELETE = %q, want RESTRICT", onDelete)
			}
		}
	}
	if !foundQCM || !foundQuestion {
		t.Fatalf("foreign keys found: qcm=%t questions=%t", foundQCM, foundQuestion)
	}
}

func assertQCMQuestionOwnershipTriggers(t *testing.T, conn *sql.DB) {
	t.Helper()
	for _, trigger := range []string{"qcm_questions_owner_insert", "qcm_questions_owner_update"} {
		var count int
		if err := conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name=?", trigger).Scan(&count); err != nil || count != 1 {
			t.Fatalf("trigger %s count=%d err=%v, want 1", trigger, count, err)
		}
	}
}

func assertQCMDeleteTableCount(t *testing.T, conn *sql.DB, table, condition string, want int) {
	t.Helper()
	var got int
	if err := conn.QueryRow("SELECT COUNT(*) FROM " + table + " WHERE " + condition).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s where %s count=%d, want %d", table, condition, got, want)
	}
}
