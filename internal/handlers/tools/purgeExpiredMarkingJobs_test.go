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
		CREATE TABLE marking_jobs (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			completed_at TIMESTAMP
		);
		INSERT INTO users (id, username) VALUES (7, 'alice');
	`); err != nil {
		t.Fatalf("prepare database: %v", err)
	}

	insertSQLiteTimestampMarkingJobForPurge(t, conn, 42, "success", now.Add(-markingJobRetention-time.Second))
	insertMarkingJobForPurge(t, conn, 43, "failed", now.Add(-markingJobRetention-time.Hour))
	insertMarkingJobForPurge(t, conn, 44, "success", now.Add(-markingJobRetention+time.Second))
	insertMarkingJobForPurge(t, conn, 45, "running", now.Add(-markingJobRetention-time.Hour))
	insertSQLiteTimestampMarkingJobForPurge(t, conn, 46, "success", now.Add(-markingJobRetention))

	expiredWorkspace := createPurgeTestWorkspace(t, "alice", 42)
	recentWorkspace := createPurgeTestWorkspace(t, "alice", 44)
	runningWorkspace := createPurgeTestWorkspace(t, "alice", 45)
	boundaryWorkspace := createPurgeTestWorkspace(t, "alice", 46)

	queries := db.New(conn)
	if err := PurgeExpiredMarkingJobs(context.Background(), queries, now); err != nil {
		t.Fatalf("first purge: %v", err)
	}

	assertPurgeJobExists(t, conn, 42, false)
	assertPurgeJobExists(t, conn, 43, false)
	assertPurgeJobExists(t, conn, 44, true)
	assertPurgeJobExists(t, conn, 45, true)
	assertPurgeJobExists(t, conn, 46, true)
	assertPurgeWorkspaceExists(t, expiredWorkspace, false)
	assertPurgeWorkspaceExists(t, recentWorkspace, true)
	assertPurgeWorkspaceExists(t, runningWorkspace, true)
	assertPurgeWorkspaceExists(t, boundaryWorkspace, true)

	if err := PurgeExpiredMarkingJobs(context.Background(), queries, now); err != nil {
		t.Fatalf("second purge: %v", err)
	}
	assertPurgeJobExists(t, conn, 44, true)
	assertPurgeJobExists(t, conn, 45, true)
	assertPurgeJobExists(t, conn, 46, true)
	assertPurgeWorkspaceExists(t, recentWorkspace, true)
	assertPurgeWorkspaceExists(t, runningWorkspace, true)
	assertPurgeWorkspaceExists(t, boundaryWorkspace, true)
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
