package generateexams

import (
	"context"
	"database/sql"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestFailExamGenerationSetsStatusToFailed(t *testing.T) {
	conn, queries := examGenerationFailedTestDB(t)
	defer conn.Close()

	if err := failExamGeneration(7, 42, context.Background(), queries); err != nil {
		t.Fatalf("failExamGeneration: %v", err)
	}
	if got := examGenerationStatus(t, conn, 42, 7); got != "failed" {
		t.Fatalf("status = %q, want failed", got)
	}
}

func TestFailExamGenerationRetriesWithFallbackContextWhenCanceled(t *testing.T) {
	conn, queries := examGenerationFailedTestDB(t)
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := failExamGeneration(7, 42, ctx, queries); err != nil {
		t.Fatalf("failExamGeneration: %v", err)
	}
	if got := examGenerationStatus(t, conn, 42, 7); got != "failed" {
		t.Fatalf("status = %q, want failed", got)
	}
}

func TestFailExamGenerationKeepsOwnershipFilter(t *testing.T) {
	conn, queries := examGenerationFailedTestDB(t)
	defer conn.Close()

	if err := failExamGeneration(8, 42, context.Background(), queries); err == nil {
		t.Fatal("failExamGeneration error = nil, want zero-row error")
	}
	if got := examGenerationStatus(t, conn, 42, 7); got != "running" {
		t.Fatalf("status = %q, want running", got)
	}
}

func examGenerationFailedTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`CREATE TABLE exams_generated (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'running'
	)`); err != nil {
		conn.Close()
		t.Fatalf("create exams_generated table: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO exams_generated (id, user_id) VALUES (?, ?)", 42, 7); err != nil {
		conn.Close()
		t.Fatalf("insert exam generation: %v", err)
	}
	return conn, db.New(conn)
}

func examGenerationStatus(t *testing.T, conn *sql.DB, examGeneratedID, userID int64) string {
	t.Helper()
	var status string
	if err := conn.QueryRow(
		"SELECT status FROM exams_generated WHERE id = ? AND user_id = ?",
		examGeneratedID,
		userID,
	).Scan(&status); err != nil {
		t.Fatalf("read exam generation status: %v", err)
	}
	return status
}
