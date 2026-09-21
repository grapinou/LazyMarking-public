package db

import (
	"os"
	"strings"
	"testing"
)

func TestQuestionInstructionMigrationPreservesExistingData(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err = conn.Exec(`CREATE TABLE questions(id INTEGER PRIMARY KEY,content TEXT,user_id INTEGER); INSERT INTO questions VALUES(1,'Énoncé historique',1); CREATE TABLE student_exam_content(content TEXT); INSERT INTO student_exam_content VALUES('{"questions":[{"content":"Ancienne copie"}]}');`); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../db/migrations/0045_add_question_instruction.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(source), "-- +goose Down", 2)
	if _, err = conn.Exec(parts[0]); err != nil {
		t.Fatal(err)
	}
	var instruction, content, snapshot string
	if err = conn.QueryRow("SELECT content,instruction FROM questions WHERE id=1").Scan(&content, &instruction); err != nil {
		t.Fatal(err)
	}
	if content != "Énoncé historique" || instruction != "" {
		t.Fatalf("migrated data: %q %q", content, instruction)
	}
	if err = conn.QueryRow("SELECT content FROM student_exam_content").Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot != `{"questions":[{"content":"Ancienne copie"}]}` {
		t.Fatal("historical snapshot changed")
	}
	if _, err = conn.Exec(parts[1]); err != nil {
		t.Fatal(err)
	}
	if err = conn.QueryRow("SELECT content FROM questions WHERE id=1").Scan(&content); err != nil || content != "Énoncé historique" {
		t.Fatalf("down: %q %v", content, err)
	}
}
