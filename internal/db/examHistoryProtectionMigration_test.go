package db

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

func TestExamHistoryProtectionMigrationUpPreservesHistoryAndInternalCascades(t *testing.T) {
	conn := newPreExamHistoryProtectionMigrationDB(t)
	up, _ := readExamHistoryProtectionMigration(t)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("apply 0035 Up: %v", err)
	}

	assertExamGeneratedForeignKey(t, conn, "RESTRICT")
	assertGeneratedExamOwnershipTriggers(t, conn)
	assertStudentExamOwnershipTriggers(t, conn, 60)
	assertHistoryChain(t, conn, 1, 1, 1, 1, 1)
	assertExamGenerationValues(t, conn)
	assertExamHistoryForeignKeysValid(t, conn)

	if _, err := conn.Exec("DELETE FROM exams WHERE id=1"); err == nil {
		t.Fatal("generated Exam deletion succeeded after Up")
	}
	assertHistoryChain(t, conn, 1, 1, 1, 1, 1)

	if result, err := conn.Exec("DELETE FROM exams WHERE id=2"); err != nil {
		t.Fatalf("delete never-generated Exam: %v", err)
	} else if rows, _ := result.RowsAffected(); rows != 1 {
		t.Fatalf("deleted Exam rows=%d, want 1", rows)
	}

	assertSQLRejected(t, conn, "generated exam ownership after Up", "INSERT INTO exams_generated(exam_id,total_students,user_id) VALUES(3,1,1)")
	assertSQLRejected(t, conn, "generated exam ownership update after Up", "UPDATE exams_generated SET exam_id=3 WHERE id=10")

	if _, err := conn.Exec("DELETE FROM exams_generated WHERE id=10"); err != nil {
		t.Fatalf("delete generation directly: %v", err)
	}
	assertHistoryChain(t, conn, 1, 0, 0, 0, 0)
}

func TestExamHistoryProtectionMigrationDownRestoresExamCascade(t *testing.T) {
	conn := newPreExamHistoryProtectionMigrationDB(t)
	up, down := readExamHistoryProtectionMigration(t)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("apply 0035 Up: %v", err)
	}
	if _, err := conn.Exec(down); err != nil {
		t.Fatalf("apply 0035 Down: %v", err)
	}

	assertExamGeneratedForeignKey(t, conn, "CASCADE")
	assertGeneratedExamOwnershipTriggers(t, conn)
	assertStudentExamOwnershipTriggers(t, conn, 70)
	assertExamGenerationValues(t, conn)
	assertExamHistoryForeignKeysValid(t, conn)
	assertSQLRejected(t, conn, "generated exam ownership after Down", "INSERT INTO exams_generated(exam_id,total_students,user_id) VALUES(3,1,1)")

	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("reapply 0035 Up after Down: %v", err)
	}
	assertExamGeneratedForeignKey(t, conn, "RESTRICT")
	assertGeneratedExamOwnershipTriggers(t, conn)
	assertStudentExamOwnershipTriggers(t, conn, 80)
	assertExamGenerationValues(t, conn)
	assertExamHistoryForeignKeysValid(t, conn)
	if _, err := conn.Exec(down); err != nil {
		t.Fatalf("reapply 0035 Down before cascade assertion: %v", err)
	}
	if _, err := conn.Exec("DELETE FROM exams WHERE id=1"); err != nil {
		t.Fatalf("delete generated Exam after Down: %v", err)
	}
	assertHistoryChain(t, conn, 0, 0, 0, 0, 0)
	assertExamHistoryForeignKeysValid(t, conn)
}

func newPreExamHistoryProtectionMigrationDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY);
		CREATE TABLE exams(id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE students(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE exams_generated (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			exam_id INTEGER NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
			processed_students INTEGER NOT NULL DEFAULT 0,
			total_students INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status TEXT NOT NULL DEFAULT 'running' CHECK(status IN ('running','success','failed')),
			user_id INTEGER NOT NULL REFERENCES users(id),
			UNIQUE(exam_id,user_id)
		);
		CREATE TABLE student_exam(
			id INTEGER PRIMARY KEY, exam_generated_id INTEGER NOT NULL REFERENCES exams_generated(id) ON DELETE CASCADE,
			student_id INTEGER NOT NULL REFERENCES students(id), user_id INTEGER NOT NULL REFERENCES users(id)
		);
		CREATE TABLE student_exam_content(
			id INTEGER PRIMARY KEY, student_exam_id INTEGER NOT NULL REFERENCES student_exam(id) ON DELETE CASCADE,
			page_tot INTEGER NOT NULL, content TEXT NOT NULL, user_id INTEGER NOT NULL REFERENCES users(id)
		);
		CREATE TABLE student_exam_page_content(
			id INTEGER PRIMARY KEY, student_exam_id INTEGER NOT NULL REFERENCES student_exam(id) ON DELETE CASCADE,
			page INTEGER NOT NULL, content TEXT NOT NULL, user_id INTEGER NOT NULL REFERENCES users(id)
		);
		CREATE TRIGGER generated_exams_owner_insert BEFORE INSERT ON exams_generated
		WHEN NOT EXISTS(SELECT 1 FROM exams WHERE id=NEW.exam_id AND user_id=NEW.user_id)
		BEGIN SELECT RAISE(ABORT, 'exam must belong to user'); END;
		CREATE TRIGGER generated_exams_owner_update BEFORE UPDATE OF exam_id,user_id ON exams_generated
		WHEN NOT EXISTS(SELECT 1 FROM exams WHERE id=NEW.exam_id AND user_id=NEW.user_id)
		BEGIN SELECT RAISE(ABORT, 'exam must belong to user'); END;
		CREATE TRIGGER student_exams_owner_insert BEFORE INSERT ON student_exam
		WHEN NOT EXISTS (SELECT 1 FROM exams_generated WHERE id = NEW.exam_generated_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM students WHERE id = NEW.student_id AND user_id = NEW.user_id)
		BEGIN SELECT RAISE(ABORT, 'generated exam and student must belong to user'); END;
		CREATE TRIGGER student_exams_owner_update BEFORE UPDATE OF exam_generated_id, student_id, user_id ON student_exam
		WHEN NOT EXISTS (SELECT 1 FROM exams_generated WHERE id = NEW.exam_generated_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM students WHERE id = NEW.student_id AND user_id = NEW.user_id)
		BEGIN SELECT RAISE(ABORT, 'generated exam and student must belong to user'); END;
		INSERT INTO users VALUES(1),(2);
		INSERT INTO exams(id,name,user_id) VALUES(1,'generated',1),(2,'free',1),(3,'foreign',2);
		INSERT INTO students VALUES(20,1),(21,2);
		INSERT INTO exams_generated(id,exam_id,processed_students,total_students,created_at,status,user_id)
		VALUES(10,1,1,1,'2026-08-01 10:00:00','success',1),
		      (11,3,0,1,'2026-08-01 11:00:00','running',2);
		INSERT INTO student_exam VALUES(30,10,20,1);
		INSERT INTO student_exam_content VALUES(40,30,2,'snapshot',1);
		INSERT INTO student_exam_page_content VALUES(50,30,1,'page snapshot',1);
	`); err != nil {
		t.Fatalf("prepare pre-0035 schema: %v", err)
	}
	return conn
}

func readExamHistoryProtectionMigration(t *testing.T) (string, string) {
	t.Helper()
	migration, err := os.ReadFile("../../db/migrations/0035_protect_generated_exam_history.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(migration), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("0035 migration has no Down section")
	}
	up := strings.Replace(parts[0], "-- +goose NO TRANSACTION", "", 1)
	up = strings.Replace(up, "-- +goose Up", "", 1)
	return up, parts[1]
}

func assertExamGeneratedForeignKey(t *testing.T, conn *sql.DB, want string) {
	t.Helper()
	rows, err := conn.Query("PRAGMA foreign_key_list(exams_generated)")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatal(err)
		}
		if table == "exams" {
			if onDelete != want {
				t.Fatalf("exams_generated exam FK ON DELETE=%q, want %q", onDelete, want)
			}
			return
		}
	}
	t.Fatal("exams_generated exam FK not found")
}

func assertGeneratedExamOwnershipTriggers(t *testing.T, conn *sql.DB) {
	t.Helper()
	var count int
	if err := conn.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name IN ('generated_exams_owner_insert','generated_exams_owner_update')`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("ownership trigger count=%d err=%v, want 2", count, err)
	}
}

func assertStudentExamOwnershipTriggers(t *testing.T, conn *sql.DB, studentExamID int64) {
	t.Helper()
	var count int
	if err := conn.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name IN ('student_exams_owner_insert','student_exams_owner_update')`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("student Exam ownership trigger count=%d err=%v, want 2", count, err)
	}
	if _, err := conn.Exec("INSERT INTO student_exam(id,exam_generated_id,student_id,user_id) VALUES(?,10,20,1)", studentExamID); err != nil {
		t.Fatalf("owned student Exam insert rejected: %v", err)
	}
	if _, err := conn.Exec("DELETE FROM student_exam WHERE id=?", studentExamID); err != nil {
		t.Fatalf("delete student Exam ownership probe: %v", err)
	}
	assertSQLRejected(t, conn, "foreign generation ownership on student Exam insert", "INSERT INTO student_exam(id,exam_generated_id,student_id,user_id) VALUES(998,11,20,1)")
	assertSQLRejected(t, conn, "foreign student ownership on student Exam insert", "INSERT INTO student_exam(id,exam_generated_id,student_id,user_id) VALUES(999,10,21,1)")
	assertSQLRejected(t, conn, "foreign generation ownership on student Exam update", "UPDATE student_exam SET exam_generated_id=11 WHERE id=30")
	assertSQLRejected(t, conn, "foreign student ownership on student Exam update", "UPDATE student_exam SET student_id=21 WHERE id=30")
}

func assertExamGenerationValues(t *testing.T, conn *sql.DB) {
	t.Helper()
	var id, examID, processed, total, userID int64
	var createdAt, status string
	if err := conn.QueryRow(`SELECT id,exam_id,processed_students,total_students,created_at,status,user_id FROM exams_generated WHERE id=10`).Scan(&id, &examID, &processed, &total, &createdAt, &status, &userID); err != nil {
		t.Fatal(err)
	}
	if id != 10 || examID != 1 || processed != 1 || total != 1 || createdAt != "2026-08-01T10:00:00Z" || status != "success" || userID != 1 {
		t.Fatalf("generation values changed: id=%d exam=%d processed=%d total=%d created=%q status=%q user=%d", id, examID, processed, total, createdAt, status, userID)
	}
}

func assertHistoryChain(t *testing.T, conn *sql.DB, exams, generations, studentExams, contents, pages int) {
	t.Helper()
	checks := []struct {
		query string
		want  int
	}{
		{"SELECT count(*) FROM exams WHERE id=1", exams},
		{"SELECT count(*) FROM exams_generated WHERE id=10", generations},
		{"SELECT count(*) FROM student_exam WHERE id=30", studentExams},
		{"SELECT count(*) FROM student_exam_content WHERE id=40", contents},
		{"SELECT count(*) FROM student_exam_page_content WHERE id=50", pages},
	}
	for _, check := range checks {
		var got int
		if err := conn.QueryRow(check.query).Scan(&got); err != nil || got != check.want {
			t.Fatalf("%s: count=%d err=%v, want %d", check.query, got, err, check.want)
		}
	}
}

func assertExamHistoryForeignKeysValid(t *testing.T, conn *sql.DB) {
	t.Helper()
	rows, err := conn.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign_key_check reported a violation")
	}
}
