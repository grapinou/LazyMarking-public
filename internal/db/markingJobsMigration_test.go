package db

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

func TestMarkingJobsCompletedAtMigrationUpAndDown(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.Exec(`
		CREATE TABLE marking_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			total_pages INTEGER DEFAULT 0,
			done_pages INTEGER DEFAULT 0,
			total_exams INTEGER DEFAULT 0,
			done_exams INTEGER DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed')),
			status_pdf TEXT NOT NULL DEFAULT 'running' CHECK (status_pdf IN ('running', 'success', 'failed')),
			exam_name TEXT,
			mark_table_name TEXT
		);
		INSERT INTO marking_jobs (id, user_id, status) VALUES (1, 7, 'running');
		INSERT INTO marking_jobs (id, user_id, status) VALUES (2, 7, 'success');
		INSERT INTO marking_jobs (id, user_id, status) VALUES (3, 7, 'failed');
	`); err != nil {
		t.Fatalf("prepare pre-migration schema: %v", err)
	}

	migration, err := os.ReadFile("../../db/migrations/0031_add_marking_jobs_completed_at.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	parts := strings.SplitN(string(migration), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("migration has no Down section")
	}
	up := strings.Replace(parts[0], "-- +goose Up", "", 1)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("apply Up migration: %v", err)
	}

	assertMigrationCompletion(t, conn, 1, false)
	assertMigrationCompletion(t, conn, 2, true)
	assertMigrationCompletion(t, conn, 3, true)
	assertSQLiteChecks(t, conn)

	if _, err := conn.Exec(parts[1]); err != nil {
		t.Fatalf("apply Down migration: %v", err)
	}
	rows, err := conn.Query("PRAGMA table_info(marking_jobs)")
	if err != nil {
		t.Fatalf("inspect schema after Down: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("scan schema after Down: %v", err)
		}
		if name == "completed_at" {
			t.Fatal("completed_at still exists after Down migration")
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("inspect schema rows after Down: %v", err)
	}
	assertSQLiteChecks(t, conn)
}

func assertMigrationCompletion(t *testing.T, conn *sql.DB, jobID int64, wantValid bool) {
	t.Helper()
	var completedAt sql.NullTime
	if err := conn.QueryRow("SELECT completed_at FROM marking_jobs WHERE id = ?", jobID).Scan(&completedAt); err != nil {
		t.Fatalf("read job %d completed_at: %v", jobID, err)
	}
	if completedAt.Valid != wantValid {
		t.Fatalf("job %d completed_at validity = %v, want %v", jobID, completedAt.Valid, wantValid)
	}
}

func assertSQLiteChecks(t *testing.T, conn *sql.DB) {
	t.Helper()
	var integrity string
	if err := conn.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		t.Fatalf("integrity_check: %v", err)
	}
	if integrity != "ok" {
		t.Fatalf("integrity_check = %q, want ok", integrity)
	}
	rows, err := conn.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign_key_check reported a violation")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("foreign_key_check rows: %v", err)
	}
}
