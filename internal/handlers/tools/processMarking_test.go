package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

type failingMarkingReader struct{}

func (failingMarkingReader) Read([]byte) (int, error) {
	return 0, errors.New("forced upload read failure")
}

func TestProcessMarkingCleansWorkspaceOnFailure(t *testing.T) {
	t.Chdir(t.TempDir())

	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.Exec(`CREATE TABLE marking_jobs (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'running'
	)`); err != nil {
		t.Fatalf("create marking_jobs table: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO marking_jobs (id, user_id) VALUES (?, ?)", 42, 7); err != nil {
		t.Fatalf("insert marking job: %v", err)
	}

	ProcessMarking(context.Background(), 7, "alice", 42, failingMarkingReader{}, db.New(conn))

	workspace := filepath.Join("assets", "tmp", "alice", "marking-42")
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("failed marking workspace still exists: %v", err)
	}

	var status string
	if err := conn.QueryRow("SELECT status FROM marking_jobs WHERE id = ? AND user_id = ?", 42, 7).Scan(&status); err != nil {
		t.Fatalf("read marking job status: %v", err)
	}
	if status != "failed" {
		t.Fatalf("marking job status = %q, want failed", status)
	}
}
