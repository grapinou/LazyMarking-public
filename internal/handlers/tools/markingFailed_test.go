package tools

import (
	"context"
	"database/sql"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestMarkingFailedSetsRunningJobToFailed(t *testing.T) {
	conn, queries := markingFailedTestDB(t)
	defer conn.Close()

	if err := MarkingFailed(7, 42, context.Background(), queries); err != nil {
		t.Fatalf("MarkingFailed: %v", err)
	}
	if got := markingJobStatus(t, conn, 42, 7); got != "failed" {
		t.Fatalf("status = %q, want failed", got)
	}
	if !markingJobCompletion(t, conn, 42, 7).Valid {
		t.Fatal("completed_at is NULL, want failure timestamp")
	}
}

func TestMarkingFailedRetriesWithFallbackContextWhenCanceled(t *testing.T) {
	conn, queries := markingFailedTestDB(t)
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := MarkingFailed(7, 42, ctx, queries); err != nil {
		t.Fatalf("MarkingFailed: %v", err)
	}
	if got := markingJobStatus(t, conn, 42, 7); got != "failed" {
		t.Fatalf("status = %q, want failed", got)
	}
	if !markingJobCompletion(t, conn, 42, 7).Valid {
		t.Fatal("completed_at is NULL, want failure timestamp")
	}
}

func TestMarkingFailedKeepsJobOwnershipFilter(t *testing.T) {
	conn, queries := markingFailedTestDB(t)
	defer conn.Close()

	if err := MarkingFailed(8, 42, context.Background(), queries); err == nil {
		t.Fatal("MarkingFailed error = nil, want zero-row error")
	}
	if got := markingJobStatus(t, conn, 42, 7); got != "running" {
		t.Fatalf("status = %q, want running", got)
	}
	if markingJobCompletion(t, conn, 42, 7).Valid {
		t.Fatal("completed_at is not NULL for running job")
	}
}

func markingFailedTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`CREATE TABLE marking_jobs (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'running',
		completed_at TIMESTAMP
	)`); err != nil {
		conn.Close()
		t.Fatalf("create marking_jobs table: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO marking_jobs (id, user_id) VALUES (?, ?)", 42, 7); err != nil {
		conn.Close()
		t.Fatalf("insert marking job: %v", err)
	}
	return conn, db.New(conn)
}

func markingJobCompletion(t *testing.T, conn *sql.DB, jobID, userID int64) sql.NullTime {
	t.Helper()
	var completedAt sql.NullTime
	if err := conn.QueryRow(
		"SELECT completed_at FROM marking_jobs WHERE id = ? AND user_id = ?",
		jobID,
		userID,
	).Scan(&completedAt); err != nil {
		t.Fatalf("read marking job completion: %v", err)
	}
	return completedAt
}

func markingJobStatus(t *testing.T, conn *sql.DB, jobID, userID int64) string {
	t.Helper()
	var status string
	if err := conn.QueryRow(
		"SELECT status FROM marking_jobs WHERE id = ? AND user_id = ?",
		jobID,
		userID,
	).Scan(&status); err != nil {
		t.Fatalf("read marking job status: %v", err)
	}
	return status
}
