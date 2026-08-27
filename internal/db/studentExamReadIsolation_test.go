package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestStudentExamContentReadsRequireOwnedParent(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
		CREATE TABLE student_exam(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam_content(
			id INTEGER PRIMARY KEY, student_exam_id INTEGER NOT NULL,
			page_tot INTEGER NOT NULL, content TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam_page_content(
			id INTEGER PRIMARY KEY, student_exam_id INTEGER NOT NULL,
			page INTEGER NOT NULL, content TEXT NOT NULL, user_id INTEGER NOT NULL);
		INSERT INTO student_exam VALUES (1,1),(2,2);
		INSERT INTO student_exam_content VALUES
			(1,1,1,'owned',1),(2,2,1,'legacy-inconsistent',1);
		INSERT INTO student_exam_page_content VALUES
			(1,1,1,'owned-page',1),(2,2,1,'legacy-inconsistent-page',1);
	`); err != nil {
		t.Fatal(err)
	}
	queries := New(conn)
	ctx := context.Background()

	if _, err := queries.GetStudentContentExam(ctx, GetStudentContentExamParams{StudentExamID: 1, UserID: 1}); err != nil {
		t.Fatalf("owned exam content: %v", err)
	}
	if _, err := queries.GetPageContent(ctx, GetPageContentParams{StudentExamID: 1, Page: 1, UserID: 1}); err != nil {
		t.Fatalf("owned page content: %v", err)
	}
	if _, err := queries.GetStudentContentExam(ctx, GetStudentContentExamParams{StudentExamID: 2, UserID: 1}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("inconsistent exam content error=%v, want sql.ErrNoRows", err)
	}
	if _, err := queries.GetPageContent(ctx, GetPageContentParams{StudentExamID: 2, Page: 1, UserID: 1}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("inconsistent page content error=%v, want sql.ErrNoRows", err)
	}
}
