package tools

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestRecoverRunningMarkingJobsCleansWorkspaceAndIsIdempotent(t *testing.T) {
	t.Chdir(t.TempDir())

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
			status TEXT NOT NULL DEFAULT 'running',
			completed_at TIMESTAMP
		);
		CREATE TABLE marking_copy_results(id INTEGER PRIMARY KEY, marking_job_id INTEGER NOT NULL REFERENCES marking_jobs(id) ON DELETE CASCADE);
		INSERT INTO users (id, username) VALUES (7, 'alice');
		INSERT INTO marking_jobs (id, user_id, status) VALUES (42, 7, 'running');
		INSERT INTO marking_jobs (id, user_id, status, completed_at) VALUES (43, 7, 'success', CURRENT_TIMESTAMP);
		INSERT INTO marking_copy_results(id, marking_job_id) VALUES(420, 42);
	`); err != nil {
		t.Fatalf("prepare database: %v", err)
	}

	workspace, ok := CreateOperationTempDir("alice", "marking-42")
	if !ok {
		t.Fatal("create running job workspace")
	}
	if err := os.WriteFile(filepath.Join(workspace, "partial.pdf"), []byte("partial"), 0o600); err != nil {
		t.Fatalf("create workspace marker: %v", err)
	}
	successWorkspace, ok := CreateOperationTempDir("alice", "marking-43")
	if !ok {
		t.Fatal("create success job workspace")
	}
	correctedPDF := filepath.Join(successWorkspace, "corrected.pdf")
	if err := os.WriteFile(correctedPDF, []byte("corrected"), 0o600); err != nil {
		t.Fatalf("create success workspace marker: %v", err)
	}

	queries := db.New(conn)
	if err := RecoverRunningMarkingJobs(context.Background(), queries); err != nil {
		t.Fatalf("first recovery: %v", err)
	}

	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("running job workspace still exists: %v", err)
	}
	assertMarkingJobStatus(t, conn, 42, "failed")
	assertMarkingJobStatus(t, conn, 43, "success")
	var partialResults int
	if err := conn.QueryRow("SELECT COUNT(*) FROM marking_copy_results WHERE marking_job_id=42").Scan(&partialResults); err != nil || partialResults != 1 {
		t.Fatalf("recovered partial results=%d err=%v, want retained", partialResults, err)
	}
	if _, err := os.Stat(correctedPDF); err != nil {
		t.Fatalf("success workspace changed by recovery: %v", err)
	}
	if !markingJobCompletion(t, conn, 42, 7).Valid {
		t.Fatal("recovered job completed_at is NULL")
	}

	if err := RecoverRunningMarkingJobs(context.Background(), queries); err != nil {
		t.Fatalf("second recovery: %v", err)
	}
	assertMarkingJobStatus(t, conn, 42, "failed")
	assertMarkingJobStatus(t, conn, 43, "success")
	if _, err := os.Stat(correctedPDF); err != nil {
		t.Fatalf("success workspace changed by second recovery: %v", err)
	}
	if !markingJobCompletion(t, conn, 42, 7).Valid {
		t.Fatal("recovered job completed_at is NULL after second recovery")
	}
}

func assertMarkingJobStatus(t *testing.T, conn *sql.DB, jobID int64, want string) {
	t.Helper()
	var got string
	if err := conn.QueryRow("SELECT status FROM marking_jobs WHERE id = ?", jobID).Scan(&got); err != nil {
		t.Fatalf("read marking job %d status: %v", jobID, err)
	}
	if got != want {
		t.Fatalf("marking job %d status = %q, want %q", jobID, got, want)
	}
}
