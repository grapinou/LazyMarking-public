package tools

import (
	"errors"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestValidateQrCodeForMarkingJobScopesGenerationAndUser(t *testing.T) {
	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`
		CREATE TABLE marking_jobs(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, exam_generated_id INTEGER);
		CREATE TABLE student_exam(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, exam_generated_id INTEGER NOT NULL);
		INSERT INTO marking_jobs VALUES (100, 1, 10);
		INSERT INTO student_exam VALUES
			(101, 1, 10),
			(102, 1, 20),
			(201, 2, 30);
	`); err != nil {
		t.Fatal(err)
	}
	queries := db.New(conn)

	if err := ValidateQrCodeForMarkingJob(t.Context(), queries, 100, 1, 101); err != nil {
		t.Fatalf("same-generation QR refused: %v", err)
	}
	for _, studentExamID := range []int64{102, 201, 999} {
		if err := ValidateQrCodeForMarkingJob(t.Context(), queries, 100, 1, studentExamID); !errors.Is(err, ErrQrOutsideMarkingGeneration) {
			t.Fatalf("student_exam_id=%d error=%v", studentExamID, err)
		}
	}
}
