package tools

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestMarkingStudentExamFailsWhenTypstWriterFails(t *testing.T) {
	markingStudentExamTestChdir(t)
	queries, closeDB := markingStudentExamTestQueries(t, 1)
	defer closeDB()

	missingDir := filepath.Join(t.TempDir(), "missing")
	markExam, err := MarkingStudentExam(
		1,
		"test",
		missingDir,
		config.Exam{StudentExamID: 42, Pages: []config.Page{{Number: 1, Name: "scan.png"}}},
		context.Background(),
		queries,
	)
	if !errors.Is(err, ErrMarkingStudentExam) {
		t.Fatalf("MarkingStudentExam error = %v, want %v", err, ErrMarkingStudentExam)
	}
	if markExam.Status {
		t.Fatal("MarkExam.Status = true, want false")
	}
}

func TestMarkingStudentExamFailsWhenTypstExportFails(t *testing.T) {
	markingStudentExamTestChdir(t)
	queries, closeDB := markingStudentExamTestQueries(t, 1)
	defer closeDB()

	t.Setenv("PATH", t.TempDir())
	markExam, err := MarkingStudentExam(
		1,
		"test",
		t.TempDir(),
		config.Exam{StudentExamID: 42, Pages: []config.Page{{Number: 1, Name: "scan.png"}}},
		context.Background(),
		queries,
	)
	if !errors.Is(err, ErrMarkingStudentExam) {
		t.Fatalf("MarkingStudentExam error = %v, want %v", err, ErrMarkingStudentExam)
	}
	if markExam.Status {
		t.Fatal("MarkExam.Status = true, want false")
	}
}

func markingStudentExamTestQueries(t *testing.T, pageTotal int64) (*db.Queries, func()) {
	t.Helper()

	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if _, err := conn.Exec(`CREATE TABLE student_exam_content (
		student_exam_id INTEGER NOT NULL,
		page_tot INTEGER NOT NULL,
		content TEXT NOT NULL,
		user_id INTEGER NOT NULL
	)`); err != nil {
		conn.Close()
		t.Fatalf("create student exam content table: %v", err)
	}

	content, err := json.Marshal(config.QCM{
		Name: "test-exam",
		Student: config.StudentQCM{
			FirstName: "Ada",
			LastName:  "Lovelace",
		},
	})
	if err != nil {
		conn.Close()
		t.Fatalf("encode test QCM: %v", err)
	}
	if _, err := conn.Exec(
		"INSERT INTO student_exam_content (student_exam_id, page_tot, content, user_id) VALUES (?, ?, ?, ?)",
		42,
		pageTotal,
		string(content),
		1,
	); err != nil {
		conn.Close()
		t.Fatalf("insert student exam content: %v", err)
	}

	return db.New(conn), func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close test database: %v", err)
		}
	}
}

func markingStudentExamTestChdir(t *testing.T) {
	t.Helper()
	workingDir, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	t.Chdir(workingDir)
}
