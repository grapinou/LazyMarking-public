package db

import (
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMarkingResultsMigrationUpAndDownPreservesLegacyJobs(t *testing.T) {
	conn := markingResultsTestDB(t)
	defer conn.Close()

	var generation sql.NullInt64
	if err := conn.QueryRow("SELECT exam_generated_id FROM marking_jobs WHERE id = 90").Scan(&generation); err != nil {
		t.Fatal(err)
	}
	if generation.Valid {
		t.Fatalf("legacy generation=%v, want NULL", generation)
	}

	_, down := markingResultsMigration(t)
	if _, err := conn.Exec(down); err != nil {
		t.Fatalf("migration Down: %v", err)
	}
	for _, table := range []string{"marking_copy_results", "marking_question_results", "marking_answer_detections"} {
		var count int
		err := conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
		if err == nil || !strings.Contains(err.Error(), "no such table") {
			t.Fatalf("table %s still queryable after Down: %v", table, err)
		}
	}
	if err := conn.QueryRow("SELECT exam_generated_id FROM marking_jobs WHERE id = 90").Scan(&generation); err != nil {
		t.Fatalf("legacy job missing after Down: %v", err)
	}
}

func TestPersistCorrectedMarkingCopyRoundTrip(t *testing.T) {
	conn := markingResultsTestDB(t)
	defer conn.Close()

	input := validCorrectedCopyInput()
	copyID, err := PersistCorrectedMarkingCopy(t.Context(), conn, input)
	if err != nil {
		t.Fatal(err)
	}
	queries := New(conn)
	copyResult, err := queries.GetMarkingCopyResult(t.Context(), GetMarkingCopyResultParams{ID: copyID, UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if copyResult.Outcome != "corrected" || copyResult.ScoreHalfUnits.Int64 != 7 || copyResult.TotalPoints.Int64 != 5 {
		t.Fatalf("copy result=%+v", copyResult)
	}
	if copyResult.CompletedAt.IsZero() || copyResult.CompletedAt.After(time.Now().Add(time.Second)) {
		t.Fatalf("completed_at=%v", copyResult.CompletedAt)
	}

	questions, err := queries.ListMarkingQuestionResults(t.Context(), ListMarkingQuestionResultsParams{CopyResultID: copyID, UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(questions) != 2 {
		t.Fatalf("question count=%d, want 2", len(questions))
	}
	if questions[0].QuestionIndex != 0 || questions[0].State != "correct" || questions[0].ScoreHalfUnits != 4 || questions[0].TotalPoints != 2 {
		t.Fatalf("question 0=%+v", questions[0])
	}
	if questions[1].QuestionIndex != 1 || questions[1].State != "partial" || questions[1].ScoreHalfUnits != 3 || questions[1].TotalPoints != 3 {
		t.Fatalf("question 1=%+v", questions[1])
	}

	answers, err := queries.ListMarkingAnswerDetections(t.Context(), ListMarkingAnswerDetectionsParams{QuestionResultID: questions[0].ID, UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(answers) != 2 || answers[0].AnswerIndex != 0 || answers[0].DetectedState != 1 || answers[0].MeanGray != 42.25 || answers[1].AnswerIndex != 1 || answers[1].DetectedState != 0 || answers[1].MeanGray != 201.75 {
		t.Fatalf("answers=%+v", answers)
	}
}

func TestMarkingResultChecksAndTerminalOutcomes(t *testing.T) {
	conn := markingResultsTestDB(t)
	defer conn.Close()
	queries := New(conn)

	correctedID := createCorrectedCopyResult(t, queries, 100, 1000)
	validStates := []CreateMarkingQuestionResultParams{
		{CopyResultID: correctedID, QuestionIndex: 0, State: "incorrect", ScoreHalfUnits: 0, TotalPoints: 3},
		{CopyResultID: correctedID, QuestionIndex: 1, State: "partial", ScoreHalfUnits: 3, TotalPoints: 3},
		{CopyResultID: correctedID, QuestionIndex: 2, State: "correct", ScoreHalfUnits: 6, TotalPoints: 3},
	}
	for _, params := range validStates {
		if _, err := queries.CreateMarkingQuestionResult(t.Context(), params); err != nil {
			t.Fatalf("valid state %+v: %v", params, err)
		}
	}
	for i, params := range []CreateMarkingQuestionResultParams{
		{CopyResultID: correctedID, QuestionIndex: 10, State: "incorrect", ScoreHalfUnits: 1, TotalPoints: 3},
		{CopyResultID: correctedID, QuestionIndex: 11, State: "partial", ScoreHalfUnits: 2, TotalPoints: 3},
		{CopyResultID: correctedID, QuestionIndex: 12, State: "correct", ScoreHalfUnits: 3, TotalPoints: 3},
	} {
		if _, err := queries.CreateMarkingQuestionResult(t.Context(), params); err == nil {
			t.Fatalf("incoherent state/score case %d succeeded", i)
		}
	}

	for i, outcome := range []string{"incomplete", "not_seen", "error"} {
		studentExamID := int64(1001 + i)
		if _, err := conn.Exec("INSERT INTO student_exam(id, exam_generated_id, user_id) VALUES (?, 10, 1)", studentExamID); err != nil {
			t.Fatal(err)
		}
		id, err := queries.CreateMarkingCopyResult(t.Context(), CreateMarkingCopyResultParams{
			UserID: 1, MarkingJobID: 100, StudentExamID: studentExamID,
			Outcome: outcome, ExpectedPages: 2, DetectedPages: int64(i),
			FailureCode: sql.NullString{String: "synthetic", Valid: outcome == "error"},
		})
		if err != nil {
			t.Fatalf("outcome %s: %v", outcome, err)
		}
		got, err := queries.GetMarkingCopyResult(t.Context(), GetMarkingCopyResultParams{ID: id, UserID: 1})
		if err != nil || got.ScoreHalfUnits.Valid || got.TotalPoints.Valid || got.CompletedAt.IsZero() {
			t.Fatalf("outcome %s result=%+v err=%v", outcome, got, err)
		}
	}

	if _, err := conn.Exec(`
		INSERT INTO marking_copy_results
		(user_id, marking_job_id, student_exam_id, outcome, expected_pages, detected_pages, score_half_units, total_points)
		VALUES (1, 101, 1001, 'incomplete', 2, 1, 0, 2)
	`); err == nil {
		t.Fatal("non-corrected outcome with score succeeded")
	}
	if _, err := conn.Exec(`
		INSERT INTO marking_copy_results
		(user_id, marking_job_id, student_exam_id, outcome, expected_pages, detected_pages, score_half_units, total_points, failure_detail)
		VALUES (1, 101, 1001, 'corrected', 2, 2, 4, 2, 'must be null')
	`); err == nil {
		t.Fatal("corrected outcome with failure detail succeeded")
	}
	for name, statement := range map[string]string{
		"zero expected pages": `INSERT INTO marking_copy_results
			(user_id, marking_job_id, student_exam_id, outcome, expected_pages, detected_pages)
			VALUES (1, 101, 1001, 'not_seen', 0, 0)`,
		"negative detected pages": `INSERT INTO marking_copy_results
			(user_id, marking_job_id, student_exam_id, outcome, expected_pages, detected_pages)
			VALUES (1, 101, 1001, 'not_seen', 2, -1)`,
		"score above total": `INSERT INTO marking_copy_results
			(user_id, marking_job_id, student_exam_id, outcome, expected_pages, detected_pages, score_half_units, total_points)
			VALUES (1, 101, 1001, 'corrected', 2, 2, 5, 2)`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := conn.Exec(statement); err == nil {
				t.Fatal("invalid copy result succeeded")
			}
		})
	}
}

func TestMarkingResultOwnershipAndUniqueness(t *testing.T) {
	conn := markingResultsTestDB(t)
	defer conn.Close()
	queries := New(conn)

	copyID := createCorrectedCopyResult(t, queries, 100, 1000)
	for _, tc := range []struct {
		name          string
		userID        int64
		jobID         int64
		studentExamID int64
	}{
		{name: "Bob student exam", userID: 1, jobID: 100, studentExamID: 2000},
		{name: "Alice other generation", userID: 1, jobID: 100, studentExamID: 1100},
		{name: "Alice user with Bob job", userID: 1, jobID: 200, studentExamID: 2000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := queries.CreateMarkingCopyResult(t.Context(), correctedCopyParams(tc.userID, tc.jobID, tc.studentExamID))
			if !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("error=%v, want sql.ErrNoRows", err)
			}
		})
	}
	if _, err := conn.Exec(`
		INSERT INTO marking_copy_results
		(user_id, marking_job_id, student_exam_id, outcome, expected_pages, detected_pages, score_half_units, total_points)
		VALUES (1, 100, 2000, 'corrected', 2, 2, 4, 2)
	`); err == nil {
		t.Fatal("direct cross-user SQL insert succeeded")
	}
	if _, err := conn.Exec(`
		INSERT INTO marking_copy_results
		(user_id, marking_job_id, student_exam_id, outcome, expected_pages, detected_pages, score_half_units, total_points)
		VALUES (1, 100, 1100, 'corrected', 2, 2, 4, 2)
	`); err == nil {
		t.Fatal("direct cross-generation SQL insert succeeded")
	}
	if _, err := conn.Exec(`
		INSERT INTO marking_copy_results
		(user_id, marking_job_id, student_exam_id, outcome, expected_pages, detected_pages, score_half_units, total_points)
		VALUES (1, 200, 2000, 'corrected', 2, 2, 4, 2)
	`); err == nil {
		t.Fatal("direct user/job ownership mismatch succeeded")
	}
	if _, err := conn.Exec("UPDATE marking_copy_results SET student_exam_id = 1100 WHERE id = ?", copyID); err == nil {
		t.Fatal("direct cross-generation ownership update succeeded")
	}
	if _, err := queries.CreateMarkingCopyResult(t.Context(), correctedCopyParams(1, 100, 1000)); err == nil {
		t.Fatal("duplicate job/student result succeeded")
	}
	if _, err := queries.CreateMarkingCopyResult(t.Context(), correctedCopyParams(1, 101, 1000)); err != nil {
		t.Fatalf("same student in another job refused: %v", err)
	}

	questionID, err := queries.CreateMarkingQuestionResult(t.Context(), CreateMarkingQuestionResultParams{
		CopyResultID: copyID, QuestionIndex: 0, State: "correct", ScoreHalfUnits: 4, TotalPoints: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queries.CreateMarkingQuestionResult(t.Context(), CreateMarkingQuestionResultParams{
		CopyResultID: copyID, QuestionIndex: 0, State: "correct", ScoreHalfUnits: 4, TotalPoints: 2,
	}); err == nil {
		t.Fatal("duplicate question index succeeded")
	}
	answer := CreateMarkingAnswerDetectionParams{QuestionResultID: questionID, AnswerIndex: 0, DetectedState: 1, MeanGray: 10}
	if _, err := queries.CreateMarkingAnswerDetection(t.Context(), answer); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.CreateMarkingAnswerDetection(t.Context(), answer); err == nil {
		t.Fatal("duplicate answer index succeeded")
	}
	for _, mean := range []float64{-0.01, 255.01} {
		if _, err := queries.CreateMarkingAnswerDetection(t.Context(), CreateMarkingAnswerDetectionParams{
			QuestionResultID: questionID, AnswerIndex: 10, DetectedState: 0, MeanGray: mean,
		}); err == nil {
			t.Fatalf("mean_gray=%v succeeded", mean)
		}
	}
	if _, err := queries.CreateMarkingAnswerDetection(t.Context(), CreateMarkingAnswerDetectionParams{
		QuestionResultID: questionID, AnswerIndex: 11, DetectedState: 2, MeanGray: 100,
	}); err == nil {
		t.Fatal("detected_state=2 succeeded")
	}
}

func TestPersistCorrectedMarkingCopyRollsBackEveryRow(t *testing.T) {
	conn := markingResultsTestDB(t)
	defer conn.Close()
	input := validCorrectedCopyInput()
	input.Questions[1].Answers = append(input.Questions[1].Answers, PersistedAnswerDetectionInput{
		AnswerIndex: 1, DetectedState: 1, MeanGray: 300,
	})
	if _, err := PersistCorrectedMarkingCopy(t.Context(), conn, input); err == nil {
		t.Fatal("invalid final answer succeeded")
	}
	for _, table := range []string{"marking_copy_results", "marking_question_results", "marking_answer_detections"} {
		var count int
		if err := conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s count=%d after rollback", table, count)
		}
	}
}

func TestMarkingResultDeletionContracts(t *testing.T) {
	conn := markingResultsTestDB(t)
	defer conn.Close()
	copyID, err := PersistCorrectedMarkingCopy(t.Context(), conn, validCorrectedCopyInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec("DELETE FROM student_exam WHERE id = 1000"); err == nil {
		t.Fatal("student_exam with historical result was deleted")
	}
	if _, err := conn.Exec("DELETE FROM exams_generated WHERE id = 10"); err == nil {
		t.Fatal("generation with marking job/result was deleted")
	}
	if _, err := conn.Exec("DELETE FROM marking_jobs WHERE id = 100"); err != nil {
		t.Fatalf("explicit job deletion: %v", err)
	}
	for table, predicate := range map[string]string{
		"marking_copy_results":     "id = ?",
		"marking_question_results": "copy_result_id = ?",
	} {
		var count int
		if err := conn.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE "+predicate, copyID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s rows remain after job deletion", table)
		}
	}
	var answerCount int
	if err := conn.QueryRow("SELECT COUNT(*) FROM marking_answer_detections").Scan(&answerCount); err != nil || answerCount != 0 {
		t.Fatalf("answer count=%d err=%v after job deletion", answerCount, err)
	}
}

func markingResultsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY, username TEXT NOT NULL);
		CREATE TABLE exams_generated(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE marking_jobs(
			id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			exam_generated_id INTEGER REFERENCES exams_generated(id) ON DELETE RESTRICT
		);
		CREATE TABLE student_exam(
			id INTEGER PRIMARY KEY, exam_generated_id INTEGER NOT NULL REFERENCES exams_generated(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id)
		);
		INSERT INTO users VALUES (1, 'alice'), (2, 'bob');
		INSERT INTO exams_generated VALUES (10, 1), (11, 1), (20, 2);
		INSERT INTO marking_jobs VALUES (90, 1, NULL), (100, 1, 10), (101, 1, 10), (110, 1, 11), (200, 2, 20);
		INSERT INTO student_exam VALUES (1000, 10, 1), (1100, 11, 1), (2000, 20, 2);
	`); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	up, _ := markingResultsMigration(t)
	if _, err := conn.Exec(up); err != nil {
		conn.Close()
		t.Fatalf("migration Up: %v", err)
	}
	return conn
}

func markingResultsMigration(t *testing.T) (string, string) {
	t.Helper()
	migration, err := os.ReadFile("../../db/migrations/0037_create_marking_results.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(migration), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("migration has no Down section")
	}
	return strings.Replace(parts[0], "-- +goose Up", "", 1), parts[1]
}

func correctedCopyParams(userID, jobID, studentExamID int64) CreateMarkingCopyResultParams {
	return CreateMarkingCopyResultParams{
		UserID: userID, MarkingJobID: jobID, StudentExamID: studentExamID,
		Outcome: "corrected", ExpectedPages: 2, DetectedPages: 2,
		ScoreHalfUnits: sql.NullInt64{Int64: 4, Valid: true},
		TotalPoints:    sql.NullInt64{Int64: 2, Valid: true},
	}
}

func createCorrectedCopyResult(t *testing.T, queries *Queries, jobID, studentExamID int64) int64 {
	t.Helper()
	id, err := queries.CreateMarkingCopyResult(t.Context(), correctedCopyParams(1, jobID, studentExamID))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func validCorrectedCopyInput() PersistedMarkingCopyInput {
	return PersistedMarkingCopyInput{
		UserID: 1, MarkingJobID: 100, StudentExamID: 1000,
		ExpectedPages: 2, DetectedPages: 2, ScoreHalfUnits: 7, TotalPoints: 5,
		Questions: []PersistedQuestionInput{
			{
				QuestionIndex: 0, State: "correct", ScoreHalfUnits: 4, TotalPoints: 2,
				Answers: []PersistedAnswerDetectionInput{
					{AnswerIndex: 0, DetectedState: 1, MeanGray: 42.25},
					{AnswerIndex: 1, DetectedState: 0, MeanGray: 201.75},
				},
			},
			{
				QuestionIndex: 1, State: "partial", ScoreHalfUnits: 3, TotalPoints: 3,
				Answers: []PersistedAnswerDetectionInput{
					{AnswerIndex: 0, DetectedState: 1, MeanGray: 149.5},
				},
			},
		},
	}
}
