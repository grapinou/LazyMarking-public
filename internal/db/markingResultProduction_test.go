package db

import (
	"database/sql"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestCompleteMarkingJobWithResultsRequiresCompleteTerminalCoverage(t *testing.T) {
	conn := productionResultsTestDB(t)
	defer conn.Close()
	queries := New(conn)

	for _, studentID := range []int64{1000, 1001} {
		input := oneQuestionCorrectedInput(100, studentID)
		copyID, err := PersistCorrectedMarkingCopy(t.Context(), conn, input)
		if err != nil {
			t.Fatalf("persist corrected %d: %v", studentID, err)
		}
		insertProductionAlignedPages(t, conn, copyID, studentID, 2)
	}
	if rows, err := queries.CompleteMarkingJobWithResults(t.Context(), completionParams(100)); err != nil || rows != 0 {
		t.Fatalf("incomplete coverage rows=%d err=%v, want 0", rows, err)
	}
	assertProductionJobStatus(t, conn, 100, "running")

	if _, err := PersistTerminalMarkingCopy(t.Context(), queries, PersistedTerminalMarkingCopyInput{
		UserID: 1, MarkingJobID: 100, StudentExamID: 1002, Outcome: "not_seen",
		ExpectedPages: 2, DetectedPages: 0, FailureCode: "no_qr_pages",
	}); err != nil {
		t.Fatal(err)
	}
	if rows, err := queries.CompleteMarkingJobWithResults(t.Context(), completionParams(100)); err != nil || rows != 1 {
		t.Fatalf("complete coverage rows=%d err=%v, want 1", rows, err)
	}
	assertProductionJobStatus(t, conn, 100, "success")

	var results, questions, detections int
	if err := conn.QueryRow("SELECT COUNT(*) FROM marking_copy_results WHERE marking_job_id=100").Scan(&results); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(`SELECT COUNT(*) FROM marking_question_results q JOIN marking_copy_results c ON c.id=q.copy_result_id WHERE c.marking_job_id=100`).Scan(&questions); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(`SELECT COUNT(*) FROM marking_answer_detections a JOIN marking_question_results q ON q.id=a.question_result_id JOIN marking_copy_results c ON c.id=q.copy_result_id WHERE c.marking_job_id=100`).Scan(&detections); err != nil {
		t.Fatal(err)
	}
	if results != 3 || questions != 2 || detections != 4 {
		t.Fatalf("hierarchy=(%d,%d,%d), want (3,2,4)", results, questions, detections)
	}
}

func TestCompleteMarkingJobWithResultsRejectsPartialCorrectedChildren(t *testing.T) {
	conn := productionResultsTestDB(t)
	defer conn.Close()
	queries := New(conn)
	copyID, err := queries.CreateMarkingCopyResult(t.Context(), CreateMarkingCopyResultParams{
		UserID: 1, MarkingJobID: 101, StudentExamID: 1000, Outcome: "corrected",
		ExpectedPages: 2, DetectedPages: 2,
		ScoreHalfUnits: sql.NullInt64{Int64: 2, Valid: true}, TotalPoints: sql.NullInt64{Int64: 1, Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := queries.CreateMarkingQuestionResult(t.Context(), CreateMarkingQuestionResultParams{
		CopyResultID: copyID, QuestionIndex: 0, State: "correct", ScoreHalfUnits: 2, TotalPoints: 1,
	}); err != nil {
		t.Fatal(err)
	}
	for _, studentID := range []int64{1001, 1002} {
		if _, err := PersistTerminalMarkingCopy(t.Context(), queries, PersistedTerminalMarkingCopyInput{
			UserID: 1, MarkingJobID: 101, StudentExamID: studentID, Outcome: "error", ExpectedPages: 2, DetectedPages: 2,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if rows, err := queries.CompleteMarkingJobWithResults(t.Context(), completionParams(101)); err != nil || rows != 0 {
		t.Fatalf("partial corrected children rows=%d err=%v, want 0", rows, err)
	}
}

func TestCompleteMarkingJobWithResultsRejectsCorrectedCopyWithoutAlignedPages(t *testing.T) {
	conn := productionResultsTestDB(t)
	defer conn.Close()
	queries := New(conn)
	if _, err := PersistCorrectedMarkingCopy(t.Context(), conn, oneQuestionCorrectedInput(101, 1000)); err != nil {
		t.Fatal(err)
	}
	for _, studentID := range []int64{1001, 1002} {
		if _, err := PersistTerminalMarkingCopy(t.Context(), queries, PersistedTerminalMarkingCopyInput{
			UserID: 1, MarkingJobID: 101, StudentExamID: studentID, Outcome: "error", ExpectedPages: 2, DetectedPages: 2,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if rows, err := queries.CompleteMarkingJobWithResults(t.Context(), completionParams(101)); err != nil || rows != 0 {
		t.Fatalf("corrected copy without aligned pages rows=%d err=%v, want 0", rows, err)
	}
	assertProductionJobStatus(t, conn, 101, "running")
}

func TestTerminalResultScopeAndIndependentJobs(t *testing.T) {
	conn := productionResultsTestDB(t)
	defer conn.Close()
	queries := New(conn)
	base := PersistedTerminalMarkingCopyInput{UserID: 1, MarkingJobID: 100, Outcome: "incomplete", ExpectedPages: 2, DetectedPages: 1}
	base.StudentExamID = 1100
	if _, err := PersistTerminalMarkingCopy(t.Context(), queries, base); err == nil {
		t.Fatal("cross-generation terminal result succeeded")
	}
	base.StudentExamID = 1000
	if _, err := PersistTerminalMarkingCopy(t.Context(), queries, base); err != nil {
		t.Fatal(err)
	}
	base.MarkingJobID = 101
	if _, err := PersistTerminalMarkingCopy(t.Context(), queries, base); err != nil {
		t.Fatalf("same copy under independent job: %v", err)
	}
}

func TestMarkingJobResultMetadataMigrationLegacyAndImmutability(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`CREATE TABLE marking_jobs(id INTEGER PRIMARY KEY, exam_generated_id INTEGER); INSERT INTO marking_jobs VALUES(1,NULL);`); err != nil {
		t.Fatal(err)
	}
	up, down := resultMetadataMigration(t)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("Up: %v", err)
	}
	var schema sql.NullInt64
	if err := conn.QueryRow("SELECT result_schema_version FROM marking_jobs WHERE id=1").Scan(&schema); err != nil || schema.Valid {
		t.Fatalf("legacy metadata=%v err=%v", schema, err)
	}
	if _, err := conn.Exec(`INSERT INTO marking_jobs(id, exam_generated_id, result_schema_version, marking_algorithm_version, detection_threshold) VALUES(2,10,1,'1',150)`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec("UPDATE marking_jobs SET detection_threshold=149 WHERE id=2"); err == nil {
		t.Fatal("metadata update succeeded")
	}
	if _, err := conn.Exec("INSERT INTO marking_jobs(id, result_schema_version) VALUES(3,1)"); err == nil {
		t.Fatal("partial metadata succeeded")
	}
	if _, err := conn.Exec(down); err != nil {
		t.Fatalf("Down: %v", err)
	}
	if _, err := conn.Exec("SELECT result_schema_version FROM marking_jobs"); err == nil {
		t.Fatal("metadata column remains after Down")
	}
}

func productionResultsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY, username TEXT NOT NULL);
		CREATE TABLE exams_generated(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE marking_jobs(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), exam_generated_id INTEGER, status TEXT NOT NULL DEFAULT 'running', status_pdf TEXT NOT NULL DEFAULT 'running', exam_name TEXT, mark_table_name TEXT, completed_at TIMESTAMP);
		CREATE TABLE student_exam(id INTEGER PRIMARY KEY, exam_generated_id INTEGER NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam_content(student_exam_id INTEGER PRIMARY KEY, page_tot INTEGER NOT NULL, content TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam_page_content(student_exam_id INTEGER NOT NULL, page INTEGER NOT NULL, content TEXT NOT NULL DEFAULT '{}', user_id INTEGER NOT NULL);
		INSERT INTO users VALUES(1,'alice');
		INSERT INTO exams_generated VALUES(10,1),(11,1);
		INSERT INTO student_exam VALUES(1000,10,1),(1001,10,1),(1002,10,1),(1100,11,1);
		INSERT INTO student_exam_content VALUES
			(1000,2,'{"questions":[{"answers":[{},{}]}]}',1),
			(1001,2,'{"questions":[{"answers":[{},{}]}]}',1),
			(1002,2,'{"questions":[{"answers":[{},{}]}]}',1),
			(1100,2,'{"questions":[{"answers":[{},{}]}]}',1);
		INSERT INTO student_exam_page_content(student_exam_id,page,user_id) VALUES
			(1000,1,1),(1000,2,1),(1001,1,1),(1001,2,1),(1002,1,1),(1002,2,1),(1100,1,1),(1100,2,1);
	`); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	up37, _ := markingResultsMigration(t)
	up38, _ := resultMetadataMigration(t)
	if _, err := conn.Exec(up37); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	if _, err := conn.Exec(up38); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	up40, _ := markingReviewMigration(t)
	if _, err := conn.Exec(up40); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO marking_jobs(id,user_id,exam_generated_id,result_schema_version,marking_algorithm_version,detection_threshold) VALUES(100,1,10,1,'1',150),(101,1,10,1,'1',150)`); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	return conn
}

func insertProductionAlignedPages(t *testing.T, conn *sql.DB, copyID, studentExamID, pages int64) {
	t.Helper()
	for page := int64(1); page <= pages; page++ {
		if _, err := conn.Exec(`INSERT INTO marking_aligned_pages(user_id,copy_result_id,page_exam,storage_key,width,height,sha256) VALUES(1,?,?,?,?,?,?)`,
			copyID, page, "aligned/student-exam-"+strconv.FormatInt(studentExamID, 10)+"/page-"+strconv.FormatInt(page, 10)+".png", 10, 10, strings.Repeat("a", 64)); err != nil {
			t.Fatalf("insert aligned page %d: %v", page, err)
		}
	}
}

func oneQuestionCorrectedInput(jobID, studentID int64) PersistedMarkingCopyInput {
	return PersistedMarkingCopyInput{UserID: 1, MarkingJobID: jobID, StudentExamID: studentID, ExpectedPages: 2, DetectedPages: 2, ScoreHalfUnits: 2, TotalPoints: 1,
		Questions: []PersistedQuestionInput{{QuestionIndex: 0, State: "correct", ScoreHalfUnits: 2, TotalPoints: 1,
			Answers: []PersistedAnswerDetectionInput{{AnswerIndex: 0, DetectedState: 1, MeanGray: 143.25}, {AnswerIndex: 1, DetectedState: 0, MeanGray: 150}}}},
	}
}

func completionParams(jobID int64) CompleteMarkingJobWithResultsParams {
	return CompleteMarkingJobWithResultsParams{ID: jobID, UserID: 1,
		ExamName: sql.NullString{String: "corrected.pdf", Valid: true}, MarkTableName: sql.NullString{String: "mark-table.pdf", Valid: true},
		ResultSchemaVersion: sql.NullInt64{Int64: 1, Valid: true}, MarkingAlgorithmVersion: sql.NullString{String: "1", Valid: true}, DetectionThreshold: sql.NullFloat64{Float64: 150, Valid: true}}
}

func assertProductionJobStatus(t *testing.T, conn *sql.DB, jobID int64, want string) {
	t.Helper()
	var got string
	if err := conn.QueryRow("SELECT status FROM marking_jobs WHERE id=?", jobID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("status=%q, want %q", got, want)
	}
}

func resultMetadataMigration(t *testing.T) (string, string) {
	t.Helper()
	b, err := os.ReadFile("../../db/migrations/0038_add_marking_job_result_metadata.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(b), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("migration has no Down")
	}
	return strings.Replace(parts[0], "-- +goose Up", "", 1), parts[1]
}
