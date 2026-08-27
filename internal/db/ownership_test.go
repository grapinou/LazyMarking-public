package db

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestOwnershipMigrationRejectsCrossUserRelations(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	schema := `
CREATE TABLE users(id INTEGER PRIMARY KEY);
CREATE TABLE subjects(id INTEGER PRIMARY KEY, user_id INTEGER); CREATE TABLE themes(id INTEGER PRIMARY KEY, user_id INTEGER);
CREATE TABLE year_levels(id INTEGER PRIMARY KEY, user_id INTEGER); CREATE TABLE skills(id INTEGER PRIMARY KEY, user_id INTEGER);
CREATE TABLE difficulties(id INTEGER PRIMARY KEY, user_id INTEGER); CREATE TABLE points(id INTEGER PRIMARY KEY, user_id INTEGER);
CREATE TABLE questions(id INTEGER PRIMARY KEY, subject_id INTEGER, theme_id INTEGER, year_level_id INTEGER, skill_id INTEGER, difficulty_id INTEGER, point_id INTEGER, user_id INTEGER);
CREATE TABLE alt_questions(id INTEGER PRIMARY KEY, question_id INTEGER, user_id INTEGER);
CREATE TABLE answers(id INTEGER PRIMARY KEY, question_id INTEGER, user_id INTEGER);
CREATE TABLE images(id INTEGER PRIMARY KEY, question_id INTEGER, user_id INTEGER);
CREATE TABLE alt_answers(id INTEGER PRIMARY KEY, alt_question_id INTEGER, user_id INTEGER);
CREATE TABLE alt_images(id INTEGER PRIMARY KEY, alt_question_id INTEGER, user_id INTEGER);
CREATE TABLE students(id INTEGER PRIMARY KEY, user_id INTEGER); CREATE TABLE class_codes(id INTEGER PRIMARY KEY, user_id INTEGER);
CREATE TABLE student_class_codes(id INTEGER PRIMARY KEY, student_id INTEGER, class_code_id INTEGER, user_id INTEGER, UNIQUE(student_id, class_code_id, user_id));
CREATE TABLE qcm(id INTEGER PRIMARY KEY, user_id INTEGER); CREATE TABLE qcm_questions(id INTEGER PRIMARY KEY, qcm_id INTEGER, question_id INTEGER, user_id INTEGER, UNIQUE(qcm_id, question_id));
CREATE TABLE periods(id INTEGER PRIMARY KEY, user_id INTEGER); CREATE TABLE years(id INTEGER PRIMARY KEY, user_id INTEGER);
CREATE TABLE exams(id INTEGER PRIMARY KEY, qcm_id INTEGER, class_code_id INTEGER, period_id INTEGER, year_id INTEGER, user_id INTEGER);
CREATE TABLE exams_generated(id INTEGER PRIMARY KEY, exam_id INTEGER, user_id INTEGER);
CREATE TABLE student_exam(id INTEGER PRIMARY KEY, exam_generated_id INTEGER, student_id INTEGER, user_id INTEGER);
CREATE TABLE student_exam_content(id INTEGER PRIMARY KEY, student_exam_id INTEGER, user_id INTEGER);
CREATE TABLE student_exam_page_content(id INTEGER PRIMARY KEY, student_exam_id INTEGER, user_id INTEGER);`
	if _, err := conn.Exec(schema); err != nil {
		t.Fatal(err)
	}

	migration, err := os.ReadFile("../../db/migrations/0030_enforce_user_relation_ownership.sql")
	if err != nil {
		t.Fatal(err)
	}
	up := strings.SplitN(string(migration), "-- +goose Down", 2)[0]
	up = strings.Replace(up, "-- +goose Up", "", 1)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("apply ownership migration: %v", err)
	}

	if _, err := conn.Exec(`
INSERT INTO users VALUES (1), (2);
INSERT INTO subjects VALUES (1,1),(2,2); INSERT INTO themes VALUES (1,1),(2,2);
INSERT INTO year_levels VALUES (1,1),(2,2); INSERT INTO skills VALUES (1,1),(2,2);
INSERT INTO difficulties VALUES (1,1),(2,2); INSERT INTO points VALUES (1,1),(2,2);
INSERT INTO questions VALUES (1,1,1,1,1,1,1,1);
INSERT INTO questions VALUES (3,2,2,2,2,2,2,2);
INSERT INTO students VALUES (1,1),(2,2); INSERT INTO class_codes VALUES (1,1),(2,2);
INSERT INTO qcm VALUES (1,1),(2,2); INSERT INTO periods VALUES (1,1),(2,2); INSERT INTO years VALUES (1,1),(2,2);`); err != nil {
		t.Fatal(err)
	}

	if _, err := conn.Exec("INSERT INTO qcm_questions(qcm_id, question_id, user_id) VALUES (2, 1, 2)"); err == nil {
		t.Fatal("user B associated user A's question with a QCM")
	}
	if _, err := conn.Exec("INSERT INTO answers(question_id, user_id) VALUES (1, 2)"); err == nil {
		t.Fatal("user B added an answer to user A's question")
	}
	if _, err := conn.Exec("INSERT INTO questions VALUES (2,1,2,2,2,2,2,2)"); err == nil {
		t.Fatal("user B created a question with user A's subject")
	}
	if _, err := conn.Exec("INSERT INTO qcm_questions(qcm_id, question_id, user_id) VALUES (1, 1, 1)"); err != nil {
		t.Fatalf("valid same-user relation was rejected: %v", err)
	}
	for name, statement := range map[string]string{
		"owner QCM to foreign question": "INSERT INTO qcm_questions(qcm_id, question_id, user_id) VALUES (1, 3, 1)",
		"foreign QCM to owner question": "INSERT INTO qcm_questions(qcm_id, question_id, user_id) VALUES (2, 1, 1)",
		"forged QCM relation user":      "INSERT INTO qcm_questions(qcm_id, question_id, user_id) VALUES (1, 1, 2)",
		"duplicate QCM membership":      "INSERT INTO qcm_questions(qcm_id, question_id, user_id) VALUES (1, 1, 1)",
	} {
		if _, err := conn.Exec(statement); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if _, err := conn.Exec("INSERT INTO exams(qcm_id, class_code_id, period_id, year_id, user_id) VALUES (1, 1, 1, 1, 1)"); err != nil {
		t.Fatalf("valid exam relation was rejected: %v", err)
	}
	for name, statement := range map[string]string{
		"foreign QCM":        "INSERT INTO exams(qcm_id, class_code_id, period_id, year_id, user_id) VALUES (2, 1, 1, 1, 1)",
		"foreign class":      "INSERT INTO exams(qcm_id, class_code_id, period_id, year_id, user_id) VALUES (1, 2, 1, 1, 1)",
		"foreign period":     "INSERT INTO exams(qcm_id, class_code_id, period_id, year_id, user_id) VALUES (1, 1, 2, 1, 1)",
		"foreign year":       "INSERT INTO exams(qcm_id, class_code_id, period_id, year_id, user_id) VALUES (1, 1, 1, 2, 1)",
		"forged exam user":   "INSERT INTO exams(qcm_id, class_code_id, period_id, year_id, user_id) VALUES (1, 1, 1, 1, 2)",
		"cross-owner update": "UPDATE exams SET qcm_id = 2 WHERE id = 1",
	} {
		if _, err := conn.Exec(statement); err == nil {
			t.Errorf("exam %s was accepted", name)
		}
	}
	if _, err := conn.Exec(`
INSERT INTO answers(id, question_id, user_id) VALUES (1, 1, 1);
INSERT INTO images(id, question_id, user_id) VALUES (1, 1, 1);
INSERT INTO alt_questions(id, question_id, user_id) VALUES (1, 1, 1);
INSERT INTO alt_answers(id, alt_question_id, user_id) VALUES (1, 1, 1);
INSERT INTO alt_images(id, alt_question_id, user_id) VALUES (1, 1, 1);`); err != nil {
		t.Fatalf("valid question child graph was rejected: %v", err)
	}
	for name, statement := range map[string]string{
		"foreign answer parent":       "INSERT INTO answers(question_id, user_id) VALUES (1, 2)",
		"foreign image parent":        "INSERT INTO images(question_id, user_id) VALUES (1, 2)",
		"foreign alt question parent": "INSERT INTO alt_questions(question_id, user_id) VALUES (1, 2)",
		"foreign alt answer parent":   "INSERT INTO alt_answers(alt_question_id, user_id) VALUES (1, 2)",
		"foreign alt image parent":    "INSERT INTO alt_images(alt_question_id, user_id) VALUES (1, 2)",
		"cross-owner answer update":   "UPDATE answers SET user_id = 2 WHERE id = 1",
		"cross-owner image update":    "UPDATE images SET user_id = 2 WHERE id = 1",
	} {
		if _, err := conn.Exec(statement); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if _, err := conn.Exec("INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES (1, 1, 1)"); err != nil {
		t.Fatalf("valid student/class relation was rejected: %v", err)
	}
	for name, statement := range map[string]string{
		"owner student to foreign class": "INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES (1, 2, 1)",
		"foreign student to owner class": "INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES (2, 1, 1)",
		"forged relation user":           "INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES (1, 1, 2)",
		"duplicate membership":           "INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES (1, 1, 1)",
	} {
		if _, err := conn.Exec(statement); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	queries := New(conn)
	_, err = queries.GetRandomQuestionByQuestionID(context.Background(), GetRandomQuestionByQuestionIDParams{
		QuestionID: 1,
		UserID:     2,
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("user B read user A's question: error = %v, want sql.ErrNoRows", err)
	}
}
