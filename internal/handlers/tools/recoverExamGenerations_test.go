package tools

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestRecoverRunningExamGenerations(t *testing.T) {
	t.Chdir(t.TempDir())

	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)

	var foreignKeysEnabled int
	if err := conn.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeysEnabled); err != nil {
		t.Fatalf("read foreign_keys setting: %v", err)
	}
	if foreignKeysEnabled != 1 {
		t.Fatal("foreign keys are not enabled")
	}

	if _, err := conn.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			username TEXT UNIQUE NOT NULL
		);
		CREATE TABLE exams (
			id INTEGER PRIMARY KEY
		);
		CREATE TABLE students (
			id INTEGER PRIMARY KEY
		);
		CREATE TABLE exams_generated (
			id INTEGER PRIMARY KEY,
			exam_id INTEGER NOT NULL,
			processed_students INTEGER NOT NULL DEFAULT 0,
			total_students INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'running',
			user_id INTEGER NOT NULL,
			FOREIGN KEY (exam_id) REFERENCES exams(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
		CREATE TABLE student_exam (
			id INTEGER PRIMARY KEY,
			exam_generated_id INTEGER NOT NULL,
			student_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			FOREIGN KEY (exam_generated_id) REFERENCES exams_generated(id) ON DELETE CASCADE,
			FOREIGN KEY (student_id) REFERENCES students(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
		CREATE TABLE student_exam_content (
			id INTEGER PRIMARY KEY,
			student_exam_id INTEGER NOT NULL REFERENCES student_exam(id) ON DELETE CASCADE,
			content TEXT NOT NULL
		);
		CREATE TABLE student_exam_page_content (
			id INTEGER PRIMARY KEY,
			student_exam_id INTEGER NOT NULL REFERENCES student_exam(id) ON DELETE CASCADE,
			content TEXT NOT NULL
		);
		INSERT INTO users (id, username) VALUES (7, 'alice'), (8, 'bob');
		INSERT INTO exams (id) VALUES (1);
		INSERT INTO students (id) VALUES (1);
		INSERT INTO exams_generated (id, exam_id, total_students, status, user_id) VALUES
			(42, 1, 1, 'running', 7),
			(43, 1, 1, 'success', 7),
			(44, 1, 1, 'failed', 7),
			(45, 1, 1, 'running', 8);
		INSERT INTO student_exam (id, exam_generated_id, student_id, user_id)
		VALUES (100, 42, 1, 7), (101, 44, 1, 7);
		INSERT INTO student_exam_content VALUES(200,100,'running content'),(201,101,'failed content');
		INSERT INTO student_exam_page_content VALUES(300,100,'running page'),(301,101,'failed page');
	`); err != nil {
		t.Fatalf("prepare database: %v", err)
	}

	runningWorkspace := createExamRecoveryWorkspace(t, "alice", "exam-42")
	successWorkspace := createExamRecoveryWorkspace(t, "alice", "exam-43")
	failedWorkspace := createExamRecoveryWorkspace(t, "alice", "exam-44")
	wrongUserWorkspace := createExamRecoveryWorkspace(t, "alice", "exam-45")
	absentWorkspace := createExamRecoveryWorkspace(t, "alice", "exam-46")
	for _, workspace := range []string{runningWorkspace, failedWorkspace} {
		partialReference := filepath.Join(workspace, "references", "student-exam-100", "page-1.png")
		if err := os.MkdirAll(filepath.Dir(partialReference), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(partialReference, []byte("partial reference"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	queries := db.New(conn)
	for _, resolved := range []db.ListRunningExamGenerationsRow{
		{ID: 43, UserID: 7, Username: "alice"},
		{ID: 44, UserID: 7, Username: "alice"},
		{ID: 46, UserID: 7, Username: "alice"},
	} {
		if err := recoverRunningExamGeneration(context.Background(), queries, resolved); err != nil {
			t.Fatalf("recover resolved generation %d: %v", resolved.ID, err)
		}
	}
	assertPathPresent(t, successWorkspace)
	assertPathPresent(t, failedWorkspace)
	assertPathPresent(t, absentWorkspace)

	if err := RecoverRunningExamGenerations(context.Background(), queries); err != nil {
		t.Fatalf("first recovery: %v", err)
	}

	assertPathAbsent(t, runningWorkspace)
	assertPathPresent(t, successWorkspace)
	assertPathAbsent(t, failedWorkspace)
	assertPathPresent(t, wrongUserWorkspace)
	assertPathPresent(t, absentWorkspace)
	assertExamGenerationAbsent(t, conn, 42)
	assertExamGenerationAbsent(t, conn, 44)
	assertExamGenerationAbsent(t, conn, 45)
	assertExamGenerationStatus(t, conn, 43, "success")

	var partialCount int
	if err := conn.QueryRow("SELECT COUNT(*) FROM student_exam WHERE id = 100").Scan(&partialCount); err != nil {
		t.Fatalf("count partial student exam: %v", err)
	}
	if partialCount != 0 {
		t.Fatalf("partial student exam count = %d, want 0", partialCount)
	}
	for _, table := range []string{"student_exam_content", "student_exam_page_content"} {
		if err := conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&partialCount); err != nil || partialCount != 0 {
			t.Fatalf("%s partial count = %d, err=%v, want 0", table, partialCount, err)
		}
	}

	if err := RecoverRunningExamGenerations(context.Background(), queries); err != nil {
		t.Fatalf("second recovery: %v", err)
	}
	assertPathPresent(t, successWorkspace)
	assertPathAbsent(t, failedWorkspace)
	assertPathPresent(t, wrongUserWorkspace)
	assertPathPresent(t, absentWorkspace)
}

func createExamRecoveryWorkspace(t *testing.T, username, operation string) string {
	t.Helper()
	workspace, ok := CreateOperationTempDir(username, operation)
	if !ok {
		t.Fatalf("create workspace %s/%s", username, operation)
	}
	if err := os.WriteFile(filepath.Join(workspace, "marker"), []byte("test"), 0o600); err != nil {
		t.Fatalf("create marker in %s: %v", workspace, err)
	}
	return workspace
}

func assertPathAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("path %s still exists: %v", path, err)
	}
}

func assertPathPresent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("path %s should exist: %v", path, err)
	}
}

func assertExamGenerationAbsent(t *testing.T, conn *sql.DB, id int64) {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM exams_generated WHERE id = ?", id).Scan(&count); err != nil {
		t.Fatalf("count exam generation %d: %v", id, err)
	}
	if count != 0 {
		t.Fatalf("exam generation %d still exists", id)
	}
}

func assertExamGenerationStatus(t *testing.T, conn *sql.DB, id int64, want string) {
	t.Helper()
	var got string
	if err := conn.QueryRow("SELECT status FROM exams_generated WHERE id = ?", id).Scan(&got); err != nil {
		t.Fatalf("read exam generation %d: %v", id, err)
	}
	if got != want {
		t.Fatalf("exam generation %d status = %q, want %q", id, got, want)
	}
}
