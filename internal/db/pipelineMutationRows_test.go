package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestExamGenerationMutationRowsAffected(t *testing.T) {
	conn, queries := newPipelineMutationTestDB(t)
	ctx := context.Background()

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.UpdateExamGeneratedProcessedStudent(ctx, UpdateExamGeneratedProcessedStudentParams{ID: 10, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateExamGeneratedProcessedStudent(ctx, UpdateExamGeneratedProcessedStudentParams{ID: 999, UserID: 1})
	})
	assertMutationRows(t, 1, func() (int64, error) {
		return queries.UpdateExamGenerated(ctx, UpdateExamGeneratedParams{Status: "success", ID: 10, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateExamGenerated(ctx, UpdateExamGeneratedParams{Status: "failed", ID: 999, UserID: 1})
	})

	var processed int64
	var status string
	if err := conn.QueryRow("SELECT processed_students, status FROM exams_generated WHERE id = 10").Scan(&processed, &status); err != nil {
		t.Fatal(err)
	}
	if processed != 1 || status != "success" {
		t.Fatalf("exam generation = processed_students %d, status %q; want 1, success", processed, status)
	}
}

func TestMarkingProgressMutationRowsAffected(t *testing.T) {
	conn, queries := newPipelineMutationTestDB(t)
	ctx := context.Background()

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.UpdateMarkingJobTotalPages(ctx, UpdateMarkingJobTotalPagesParams{
			TotalPages: sql.NullInt64{Int64: 3, Valid: true}, ID: 20, UserID: 1,
		})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateMarkingJobTotalPages(ctx, UpdateMarkingJobTotalPagesParams{
			TotalPages: sql.NullInt64{Int64: 3, Valid: true}, ID: 999, UserID: 1,
		})
	})
	assertMutationRows(t, 1, func() (int64, error) {
		return queries.UpdateMarkingJobTotalExam(ctx, UpdateMarkingJobTotalExamParams{
			TotalExams: sql.NullInt64{Int64: 2, Valid: true}, ID: 20, UserID: 1,
		})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateMarkingJobTotalExam(ctx, UpdateMarkingJobTotalExamParams{
			TotalExams: sql.NullInt64{Int64: 2, Valid: true}, ID: 999, UserID: 1,
		})
	})
	assertMutationRows(t, 1, func() (int64, error) {
		return queries.UpdateMarkingJobPageDone(ctx, UpdateMarkingJobPageDoneParams{ID: 20, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateMarkingJobPageDone(ctx, UpdateMarkingJobPageDoneParams{ID: 999, UserID: 1})
	})
	assertMutationRows(t, 1, func() (int64, error) {
		return queries.UpdateMarkingJobExamDone(ctx, UpdateMarkingJobExamDoneParams{ID: 20, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateMarkingJobExamDone(ctx, UpdateMarkingJobExamDoneParams{ID: 999, UserID: 1})
	})

	var totalPages, totalExams, donePages, doneExams int64
	if err := conn.QueryRow(`
		SELECT total_pages, total_exams, done_pages, done_exams
		FROM marking_jobs WHERE id = 20
	`).Scan(&totalPages, &totalExams, &donePages, &doneExams); err != nil {
		t.Fatal(err)
	}
	if totalPages != 3 || totalExams != 2 || donePages != 1 || doneExams != 1 {
		t.Fatalf("marking progress = totals %d/%d, done %d/%d; want totals 3/2, done 1/1", totalPages, totalExams, donePages, doneExams)
	}
}

func TestMarkingTerminalMutationRowsAffected(t *testing.T) {
	conn, queries := newPipelineMutationTestDB(t)
	ctx := context.Background()

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.FailMarkingJob(ctx, FailMarkingJobParams{ID: 20, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.FailMarkingJob(ctx, FailMarkingJobParams{ID: 999, UserID: 1})
	})
	assertMutationRows(t, 1, func() (int64, error) {
		return queries.CompleteMarkingJob(ctx, CompleteMarkingJobParams{
			ExamName:      sql.NullString{String: "corrected.pdf", Valid: true},
			MarkTableName: sql.NullString{String: "mark-table.pdf", Valid: true},
			ID:            21,
			UserID:        1,
		})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.CompleteMarkingJob(ctx, CompleteMarkingJobParams{ID: 999, UserID: 1})
	})

	var failedStatus, successStatus, successPDFStatus string
	var failedCompletedAt, successCompletedAt sql.NullTime
	if err := conn.QueryRow("SELECT status, completed_at FROM marking_jobs WHERE id = 20").Scan(&failedStatus, &failedCompletedAt); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow("SELECT status, status_pdf, completed_at FROM marking_jobs WHERE id = 21").Scan(&successStatus, &successPDFStatus, &successCompletedAt); err != nil {
		t.Fatal(err)
	}
	if failedStatus != "failed" || !failedCompletedAt.Valid {
		t.Fatalf("failed job = status %q, completed_at valid %v", failedStatus, failedCompletedAt.Valid)
	}
	if successStatus != "success" || successPDFStatus != "success" || !successCompletedAt.Valid {
		t.Fatalf("completed job = status %q, status_pdf %q, completed_at valid %v", successStatus, successPDFStatus, successCompletedAt.Valid)
	}
}

func newPipelineMutationTestDB(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec(`
		CREATE TABLE exams_generated (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			processed_students INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'running'
		);
		CREATE TABLE marking_jobs (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			total_pages INTEGER DEFAULT 0,
			done_pages INTEGER DEFAULT 0,
			total_exams INTEGER DEFAULT 0,
			done_exams INTEGER DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'running',
			status_pdf TEXT NOT NULL DEFAULT 'running',
			exam_name TEXT,
			mark_table_name TEXT,
			completed_at TIMESTAMP
		);
		INSERT INTO exams_generated (id, user_id) VALUES (10, 1);
		INSERT INTO marking_jobs (id, user_id) VALUES (20, 1), (21, 1);
	`); err != nil {
		t.Fatalf("create pipeline mutation test schema: %v", err)
	}

	return conn, New(conn)
}
