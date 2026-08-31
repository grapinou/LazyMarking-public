package tools

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestPurgeExpiredMarkingJobs(t *testing.T) {
	t.Chdir(t.TempDir())
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)

	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			username TEXT UNIQUE NOT NULL
		);
		CREATE TABLE exams_generated (
			id INTEGER PRIMARY KEY,
			status TEXT NOT NULL,
			user_id INTEGER NOT NULL
		);
		CREATE TABLE marking_jobs (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			completed_at TIMESTAMP,
			exam_generated_id INTEGER REFERENCES exams_generated(id) ON DELETE RESTRICT
		);
		CREATE TABLE student_exam (
			id INTEGER PRIMARY KEY,
			exam_generated_id INTEGER NOT NULL REFERENCES exams_generated(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL
		);
		CREATE TABLE marking_copy_results (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			marking_job_id INTEGER NOT NULL REFERENCES marking_jobs(id) ON DELETE CASCADE,
			student_exam_id INTEGER NOT NULL REFERENCES student_exam(id) ON DELETE RESTRICT
		);
		CREATE TABLE marking_question_results (
			id INTEGER PRIMARY KEY,
			copy_result_id INTEGER NOT NULL REFERENCES marking_copy_results(id) ON DELETE CASCADE
		);
		CREATE TABLE marking_answer_detections (
			id INTEGER PRIMARY KEY,
			question_result_id INTEGER NOT NULL REFERENCES marking_question_results(id) ON DELETE CASCADE
		);
		INSERT INTO users (id, username) VALUES (7, 'alice');
		INSERT INTO exams_generated(id, status, user_id) VALUES (10, 'success', 7);
		INSERT INTO student_exam(id, exam_generated_id, user_id) VALUES (100, 10, 7);
	`); err != nil {
		t.Fatalf("prepare database: %v", err)
	}

	insertSQLiteTimestampMarkingJobForPurge(t, conn, 42, "success", now.Add(-100*24*time.Hour))
	insertMarkingJobForPurge(t, conn, 43, "failed", now.Add(-markingJobRetention-time.Hour))
	insertMarkingJobForPurge(t, conn, 44, "success", now.Add(-markingJobRetention+time.Second))
	insertMarkingJobForPurge(t, conn, 45, "running", now.Add(-markingJobRetention-time.Hour))
	insertSQLiteTimestampMarkingJobForPurge(t, conn, 46, "failed", now.Add(-markingJobRetention))
	insertMarkingJobForPurge(t, conn, 47, "failed", now.Add(-markingJobRetention+time.Second))
	insertMarkingJobForPurge(t, conn, 48, "failed", now.Add(-markingJobRetention-time.Hour))
	insertSQLiteTimestampMarkingJobForPurge(t, conn, 49, "success", now.Add(-100*24*time.Hour))
	if _, err := conn.Exec(`
		UPDATE marking_jobs SET exam_generated_id = 10 WHERE id = 42;
		INSERT INTO marking_copy_results(id, user_id, marking_job_id, student_exam_id) VALUES (420, 7, 42, 100);
		INSERT INTO marking_question_results(id, copy_result_id) VALUES (421, 420);
		INSERT INTO marking_answer_detections(id, question_result_id) VALUES (422, 421);
	`); err != nil {
		t.Fatalf("insert durable result hierarchy: %v", err)
	}

	successWorkspace := createPurgeTestWorkspace(t, "alice", 42)
	correctedPDF := filepath.Join(successWorkspace, "corrected.pdf")
	if err := os.WriteFile(correctedPDF, []byte("corrected"), 0o600); err != nil {
		t.Fatalf("write corrected.pdf: %v", err)
	}
	legacySuccessWorkspace := createPurgeTestWorkspace(t, "alice", 49)
	expiredFailedWorkspace := createPurgeTestWorkspace(t, "alice", 43)
	recentWorkspace := createPurgeTestWorkspace(t, "alice", 44)
	runningWorkspace := createPurgeTestWorkspace(t, "alice", 45)
	boundaryWorkspace := createPurgeTestWorkspace(t, "alice", 46)
	recentFailedWorkspace := createPurgeTestWorkspace(t, "alice", 47)

	queries := db.New(conn)
	if err := PurgeExpiredMarkingJobs(context.Background(), queries, now); err != nil {
		t.Fatalf("first purge: %v", err)
	}

	assertPurgeJobExists(t, conn, 42, true)
	assertPurgeJobExists(t, conn, 43, false)
	assertPurgeJobExists(t, conn, 44, true)
	assertPurgeJobExists(t, conn, 45, true)
	assertPurgeJobExists(t, conn, 46, true)
	assertPurgeJobExists(t, conn, 47, true)
	assertPurgeJobExists(t, conn, 48, false)
	assertPurgeJobExists(t, conn, 49, true)
	assertPurgeWorkspaceExists(t, successWorkspace, true)
	assertPurgeWorkspaceExists(t, legacySuccessWorkspace, true)
	assertPurgeWorkspaceExists(t, expiredFailedWorkspace, false)
	assertPurgeWorkspaceExists(t, recentWorkspace, true)
	assertPurgeWorkspaceExists(t, runningWorkspace, true)
	assertPurgeWorkspaceExists(t, boundaryWorkspace, true)
	assertPurgeWorkspaceExists(t, recentFailedWorkspace, true)
	if _, err := os.Stat(correctedPDF); err != nil {
		t.Fatalf("corrected.pdf was removed: %v", err)
	}
	for table, id := range map[string]int64{
		"marking_copy_results":      420,
		"marking_question_results":  421,
		"marking_answer_detections": 422,
	} {
		var count int
		if err := conn.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE id = ?", id).Scan(&count); err != nil || count != 1 {
			t.Fatalf("durable %s count=%d err=%v", table, count, err)
		}
	}

	if err := PurgeExpiredMarkingJobs(context.Background(), queries, now); err != nil {
		t.Fatalf("second purge: %v", err)
	}
	assertPurgeJobExists(t, conn, 44, true)
	assertPurgeJobExists(t, conn, 45, true)
	assertPurgeJobExists(t, conn, 46, true)
	assertPurgeJobExists(t, conn, 47, true)
	assertPurgeJobExists(t, conn, 49, true)
	assertPurgeWorkspaceExists(t, successWorkspace, true)
	assertPurgeWorkspaceExists(t, legacySuccessWorkspace, true)
	assertPurgeWorkspaceExists(t, recentWorkspace, true)
	assertPurgeWorkspaceExists(t, runningWorkspace, true)
	assertPurgeWorkspaceExists(t, boundaryWorkspace, true)
	assertPurgeWorkspaceExists(t, recentFailedWorkspace, true)
}

func TestPurgeExpiredFailedJobKeepsDBRowWhenWorkspaceRemovalFails(t *testing.T) {
	t.Chdir(t.TempDir())
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY, username TEXT NOT NULL);
		CREATE TABLE marking_jobs(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, status TEXT NOT NULL, completed_at TIMESTAMP);
		INSERT INTO users VALUES (7, 'alice');
		INSERT INTO marking_jobs VALUES (60, 7, 'failed', '2026-08-01 00:00:00');
	`); err != nil {
		t.Fatal(err)
	}
	userDir := filepath.Join("assets", "tmp", "alice")
	if err := os.MkdirAll(userDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(userDir, "marking-60")); err != nil {
		t.Fatal(err)
	}
	if err := PurgeExpiredMarkingJobs(context.Background(), db.New(conn), now); err == nil {
		t.Fatal("purge succeeded through unsafe workspace")
	}
	assertPurgeJobExists(t, conn, 60, true)
}

func insertMarkingJobForPurge(t *testing.T, conn *sql.DB, id int64, status string, completedAt time.Time) {
	t.Helper()
	if _, err := conn.Exec(
		"INSERT INTO marking_jobs (id, user_id, status, completed_at) VALUES (?, 7, ?, ?)",
		id,
		status,
		completedAt,
	); err != nil {
		t.Fatalf("insert marking job %d: %v", id, err)
	}
}

func insertSQLiteTimestampMarkingJobForPurge(t *testing.T, conn *sql.DB, id int64, status string, completedAt time.Time) {
	t.Helper()
	if _, err := conn.Exec(
		"INSERT INTO marking_jobs (id, user_id, status, completed_at) VALUES (?, 7, ?, ?)",
		id,
		status,
		completedAt.UTC().Format("2006-01-02 15:04:05"),
	); err != nil {
		t.Fatalf("insert SQLite-timestamp marking job %d: %v", id, err)
	}
}

func createPurgeTestWorkspace(t *testing.T, username string, jobID int64) string {
	t.Helper()
	operation := "marking-" + strconv.FormatInt(jobID, 10)
	workspace, ok := CreateOperationTempDir(username, operation)
	if !ok {
		t.Fatalf("create workspace for job %d", jobID)
	}
	if err := os.WriteFile(filepath.Join(workspace, "result.pdf"), []byte("result"), 0o600); err != nil {
		t.Fatalf("write workspace for job %d: %v", jobID, err)
	}
	return workspace
}

func assertPurgeJobExists(t *testing.T, conn *sql.DB, jobID int64, want bool) {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM marking_jobs WHERE id = ?", jobID).Scan(&count); err != nil {
		t.Fatalf("count marking job %d: %v", jobID, err)
	}
	if got := count == 1; got != want {
		t.Fatalf("marking job %d existence = %v, want %v", jobID, got, want)
	}
}

func assertPurgeWorkspaceExists(t *testing.T, workspace string, want bool) {
	t.Helper()
	info, err := os.Stat(workspace)
	got := err == nil && info.IsDir()
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("stat workspace %s: %v", workspace, err)
	}
	if got != want {
		t.Fatalf("workspace %s existence = %v, want %v", workspace, got, want)
	}
}
