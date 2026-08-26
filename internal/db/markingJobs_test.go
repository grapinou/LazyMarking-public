package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestCompleteMarkingJobWritesFinalStateAtomically(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Exec(`CREATE TABLE marking_jobs (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'running',
		status_pdf TEXT NOT NULL DEFAULT 'running',
		exam_name TEXT,
		mark_table_name TEXT,
		completed_at TIMESTAMP
	)`); err != nil {
		t.Fatalf("create marking_jobs table: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO marking_jobs (id, user_id) VALUES (?, ?)", 42, 7); err != nil {
		t.Fatalf("insert marking job: %v", err)
	}

	queries := New(conn)
	rows, err := queries.CompleteMarkingJob(context.Background(), CompleteMarkingJobParams{
		ExamName:      sql.NullString{String: "corrected.pdf", Valid: true},
		MarkTableName: sql.NullString{String: "marks.pdf", Valid: true},
		ID:            42,
		UserID:        7,
	})
	if err != nil {
		t.Fatalf("complete marking job: %v", err)
	}
	if rows != 1 {
		t.Fatalf("complete marking job rows = %d, want 1", rows)
	}

	var status, statusPDF, examName, markTableName string
	var completedAt sql.NullTime
	err = conn.QueryRow(
		"SELECT status, status_pdf, exam_name, mark_table_name, completed_at FROM marking_jobs WHERE id = ? AND user_id = ?",
		42,
		7,
	).Scan(&status, &statusPDF, &examName, &markTableName, &completedAt)
	if err != nil {
		t.Fatalf("read completed marking job: %v", err)
	}
	if status != "success" || statusPDF != "success" || examName != "corrected.pdf" || markTableName != "marks.pdf" {
		t.Fatalf(
			"completed job = status %q, status_pdf %q, exam_name %q, mark_table_name %q",
			status,
			statusPDF,
			examName,
			markTableName,
		)
	}
	if !completedAt.Valid {
		t.Fatal("completed_at is NULL, want completion timestamp")
	}
}
