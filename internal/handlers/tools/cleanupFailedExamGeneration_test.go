package tools

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestDeleteFailedExamGeneratedStatusAndOwnershipMatrix(t *testing.T) {
	conn, queries := setupFailedGenerationCleanupTest(t)
	ctx := context.Background()
	tests := []struct {
		name   string
		params db.DeleteFailedExamGeneratedParams
		want   int64
	}{
		{"failed", db.DeleteFailedExamGeneratedParams{ID: 10, UserID: 1}, 1},
		{"success", db.DeleteFailedExamGeneratedParams{ID: 11, UserID: 1}, 0},
		{"running", db.DeleteFailedExamGeneratedParams{ID: 12, UserID: 1}, 0},
		{"absent", db.DeleteFailedExamGeneratedParams{ID: 999, UserID: 1}, 0},
		{"foreign", db.DeleteFailedExamGeneratedParams{ID: 13, UserID: 1}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := queries.DeleteFailedExamGenerated(ctx, tc.params)
			if err != nil || rows != tc.want {
				t.Fatalf("rows=%d err=%v, want %d", rows, err, tc.want)
			}
		})
	}
	assertCleanupRowCount(t, conn, "exams_generated", "id=11", 1)
	assertCleanupRowCount(t, conn, "exams_generated", "id=12", 1)
	assertCleanupRowCount(t, conn, "exams_generated", "id=13", 1)
}

func TestCleanupFailedExamGenerationIsIdempotentAndCascades(t *testing.T) {
	conn, queries := setupFailedGenerationCleanupTest(t)
	t.Chdir(t.TempDir())
	workspace, ok := CreateOperationTempDir("alice", "exam-10")
	if !ok {
		t.Fatal("create failed-generation workspace")
	}
	if err := os.WriteFile(filepath.Join(workspace, "partial.pdf"), []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := CleanupFailedExamGeneration(context.Background(), queries, 10, 1, "alice"); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct{ table, where string }{
		{"exams_generated", "id=10"},
		{"student_exam", "exam_generated_id=10"},
		{"student_exam_content", "student_exam_id=100"},
		{"student_exam_page_content", "student_exam_id=100"},
	} {
		assertCleanupRowCount(t, conn, check.table, check.where, 0)
	}
	assertCleanupRowCount(t, conn, "exams", "id=1", 1)
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("workspace remains after cleanup: %v", err)
	}
	if err := CleanupFailedExamGeneration(context.Background(), queries, 10, 1, "alice"); err != nil {
		t.Fatalf("idempotent cleanup: %v", err)
	}
	if _, err := queries.CreateExamGenerated(context.Background(), db.CreateExamGeneratedParams{ExamID: 1, TotalStudents: 1, UserID: 1}); err != nil {
		t.Fatalf("immediate retry CreateExamGenerated: %v", err)
	}
}

func TestCleanupFailedExamGenerationNeverDeletesSuccessOrRunning(t *testing.T) {
	conn, queries := setupFailedGenerationCleanupTest(t)
	for _, tc := range []struct {
		id     int64
		status string
	}{{11, "success"}, {12, "running"}} {
		if err := CleanupFailedExamGeneration(context.Background(), queries, tc.id, 1, "alice"); err == nil {
			t.Fatalf("cleanup %s generation succeeded", tc.status)
		}
		assertCleanupRowCount(t, conn, "exams_generated", "id="+strconv.FormatInt(tc.id, 10), 1)
	}
}

func TestCleanupFailedExamGenerationFilesystemErrorDoesNotReblockDB(t *testing.T) {
	conn, queries := setupFailedGenerationCleanupTest(t)
	previous := removeFailedExamGenerationWorkspace
	removeFailedExamGenerationWorkspace = func(string, string) error { return errors.New("forced workspace failure") }
	t.Cleanup(func() { removeFailedExamGenerationWorkspace = previous })

	if err := CleanupFailedExamGeneration(context.Background(), queries, 10, 1, "alice"); err == nil {
		t.Fatal("cleanup error=nil, want filesystem error")
	}
	assertCleanupRowCount(t, conn, "exams_generated", "id=10", 0)
	assertCleanupRowCount(t, conn, "exams", "id=1", 1)
}

func setupFailedGenerationCleanupTest(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY,username TEXT);
		CREATE TABLE exams(id INTEGER PRIMARY KEY,user_id INTEGER);
		CREATE TABLE students(id INTEGER PRIMARY KEY,user_id INTEGER);
		CREATE TABLE exams_generated(
			id INTEGER PRIMARY KEY,exam_id INTEGER NOT NULL REFERENCES exams(id) ON DELETE RESTRICT,
			processed_students INTEGER DEFAULT 0,total_students INTEGER NOT NULL,status TEXT NOT NULL DEFAULT 'running',user_id INTEGER NOT NULL,
			UNIQUE(exam_id,user_id));
		CREATE TABLE student_exam(id INTEGER PRIMARY KEY,exam_generated_id INTEGER NOT NULL REFERENCES exams_generated(id) ON DELETE CASCADE,student_id INTEGER,user_id INTEGER);
		CREATE TABLE student_exam_content(id INTEGER PRIMARY KEY,student_exam_id INTEGER NOT NULL REFERENCES student_exam(id) ON DELETE CASCADE,content TEXT);
		CREATE TABLE student_exam_page_content(id INTEGER PRIMARY KEY,student_exam_id INTEGER NOT NULL REFERENCES student_exam(id) ON DELETE CASCADE,content TEXT);
		INSERT INTO users VALUES(1,'alice'),(2,'bob');
		INSERT INTO exams VALUES(1,1),(2,1),(3,1),(4,2);
		INSERT INTO students VALUES(1,1);
		INSERT INTO exams_generated(id,exam_id,total_students,status,user_id) VALUES(10,1,1,'failed',1),(11,2,1,'success',1),(12,3,1,'running',1),(13,4,1,'failed',2);
		INSERT INTO student_exam VALUES(100,10,1,1);
		INSERT INTO student_exam_content VALUES(200,100,'partial content');
		INSERT INTO student_exam_page_content VALUES(300,100,'partial page');
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func assertCleanupRowCount(t *testing.T, conn *sql.DB, table, where string, want int) {
	t.Helper()
	var got int
	if err := conn.QueryRow("SELECT count(*) FROM " + table + " WHERE " + where).Scan(&got); err != nil || got != want {
		t.Fatalf("%s where %s count=%d err=%v, want %d", table, where, got, err, want)
	}
}
