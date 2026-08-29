package db

import (
	"database/sql"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestQCMQuestionPositionMigrationBackfillsAndPreservesIntegrity(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY);
		CREATE TABLE qcm(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE questions(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE qcm_questions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			qcm_id INTEGER NOT NULL REFERENCES qcm(id),
			question_id INTEGER NOT NULL REFERENCES questions(id) ON DELETE RESTRICT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			UNIQUE(qcm_id, question_id)
		);
		CREATE TRIGGER qcm_questions_owner_insert BEFORE INSERT ON qcm_questions
		WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
		BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;
		CREATE TRIGGER qcm_questions_owner_update BEFORE UPDATE OF qcm_id, question_id, user_id ON qcm_questions
		WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
		BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;
		INSERT INTO users VALUES (1), (2);
		INSERT INTO qcm VALUES (1,1), (2,2), (3,1);
		INSERT INTO questions VALUES (100,1), (300,1), (500,1), (600,1), (200,2), (400,2);
		INSERT INTO qcm_questions(id,qcm_id,question_id,user_id) VALUES
			(30,1,100,1), (10,1,300,1), (40,2,400,2), (20,2,200,2);
	`); err != nil {
		t.Fatalf("prepare pre-migration database: %v", err)
	}

	up, _ := readQCMPositionMigration(t)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("apply position migration: %v", err)
	}

	rows, err := conn.Query("SELECT id,qcm_id,question_id,user_id,position FROM qcm_questions ORDER BY qcm_id,position")
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
	want := [][5]int64{{10, 1, 300, 1, 1}, {30, 1, 100, 1, 2}, {20, 2, 200, 2, 1}, {40, 2, 400, 2, 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("migrated relations = %v, want %v", got, want)
	}

	assertSQLRejected(t, conn, "position below one", "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(1,500,1,0)")
	assertSQLRejected(t, conn, "duplicate position", "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(1,500,1,1)")
	if _, err := conn.Exec("INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(3,500,1,1)"); err != nil {
		t.Fatalf("same position in another QCM rejected: %v", err)
	}
	assertSQLRejected(t, conn, "duplicate QCM question", "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(1,300,1,3)")

	assertSQLRejected(t, conn, "foreign QCM", "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(2,500,1,3)")
	assertSQLRejected(t, conn, "foreign question", "INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(1,400,1,3)")
	assertSQLRejected(t, conn, "forged QCM update", "UPDATE qcm_questions SET qcm_id=2 WHERE id=10")
	assertSQLRejected(t, conn, "forged question update", "UPDATE qcm_questions SET question_id=400 WHERE id=10")
	assertSQLRejected(t, conn, "forged user update", "UPDATE qcm_questions SET user_id=2 WHERE id=10")

	for _, trigger := range []string{"qcm_questions_owner_insert", "qcm_questions_owner_update"} {
		var count int
		if err := conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name=?", trigger).Scan(&count); err != nil || count != 1 {
			t.Fatalf("trigger %s count = %d, err = %v", trigger, count, err)
		}
	}
	assertSQLRejected(t, conn, "QCM delete remains restricted", "DELETE FROM qcm WHERE id=1")
	assertSQLRejected(t, conn, "question delete remains restricted", "DELETE FROM questions WHERE id=100")
}

func TestQCMQuestionPositionMigrationDownPreservesOwnershipTriggers(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY);
		CREATE TABLE qcm(id INTEGER PRIMARY KEY,user_id INTEGER);
		CREATE TABLE questions(id INTEGER PRIMARY KEY,user_id INTEGER);
		CREATE TABLE qcm_questions(id INTEGER PRIMARY KEY,qcm_id INTEGER,question_id INTEGER,user_id INTEGER,position INTEGER NOT NULL CHECK(position>=1),UNIQUE(qcm_id,question_id),UNIQUE(qcm_id,position));
		INSERT INTO users VALUES(1),(2); INSERT INTO qcm VALUES(1,1),(2,2); INSERT INTO questions VALUES(1,1),(2,2);
		INSERT INTO qcm_questions VALUES(1,1,1,1,1);
	`); err != nil {
		t.Fatal(err)
	}
	_, down := readQCMPositionMigration(t)
	if _, err := conn.Exec(down); err != nil {
		t.Fatalf("apply down migration: %v", err)
	}
	assertSQLRejected(t, conn, "ownership after down", "INSERT INTO qcm_questions(qcm_id,question_id,user_id) VALUES(2,1,1)")
	if _, err := conn.Exec("SELECT position FROM qcm_questions"); err == nil {
		t.Fatal("position column still exists after down migration")
	}
}

func readQCMPositionMigration(t *testing.T) (string, string) {
	t.Helper()
	migration, err := os.ReadFile("../../db/migrations/0032_add_qcm_question_position.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(migration), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("position migration has no Down section")
	}
	return strings.Replace(parts[0], "-- +goose Up", "", 1), parts[1]
}

func assertSQLRejected(t *testing.T, conn *sql.DB, name, statement string) {
	t.Helper()
	if _, err := conn.Exec(statement); err == nil {
		t.Fatalf("%s was accepted", name)
	}
}
