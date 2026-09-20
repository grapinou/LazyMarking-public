package db

import (
	"database/sql"
	"testing"
)

func TestCurrentExamResultsPreserveEarlierCorrectionsAcrossSuccessiveImports(t *testing.T) {
	conn := markingWorkflowTestDB(t)
	defer conn.Close()

	if _, err := conn.Exec(`
		INSERT INTO marking_jobs(id,user_id,exam_generated_id,status,status_pdf,completed_at,review_policy_version,detection_threshold,ambiguity_delta,source_pdf_filename) VALUES
			(100,1,10,'success','success','2026-09-19 10:00:00','detector-agreement-v1',150,0,'principal.pdf'),
			(101,1,10,'success','success','2026-09-20 10:00:00','detector-agreement-v1',150,0,'rattrapage.pdf'),
			(102,1,10,'success','success','2026-09-20 11:00:00','detector-agreement-v1',150,0,'controle.pdf'),
			(103,1,10,'failed','running','2026-09-20 12:00:00','detector-agreement-v1',150,0,'interrompu.pdf');

		INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,score_half_units,total_points) VALUES
			(1000,1,100,100,'corrected',14,10),
			(1001,1,100,101,'corrected',12,10),
			(1002,1,100,102,'not_seen',NULL,NULL),
			(1010,1,101,100,'not_seen',NULL,NULL),
			(1011,1,101,101,'not_seen',NULL,NULL),
			(1012,1,101,102,'corrected',16,10),
			(1020,1,102,100,'incomplete',NULL,NULL),
			(1021,1,102,101,'not_seen',NULL,NULL),
			(1022,1,102,102,'not_seen',NULL,NULL),
			(1030,1,103,100,'corrected',2,10);

		INSERT INTO marking_question_results(id,copy_result_id) VALUES(2001,1001);
		INSERT INTO marking_answer_detections(id,question_result_id,review_reason) VALUES(3001,2001,'detector_disagreement');
	`); err != nil {
		t.Fatal(err)
	}

	queries := New(conn)
	rows, err := queries.ListCurrentExamResultsForMarkingJob(t.Context(), ListCurrentExamResultsForMarkingJobParams{UserID: 1, MarkingJobID: 102})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("current result count=%d, want 3", len(rows))
	}
	byStudent := make(map[int64]ListCurrentExamResultsForMarkingJobRow, len(rows))
	for _, row := range rows {
		byStudent[row.StudentExamID] = row
	}
	assertCurrentWorkflowResult(t, byStudent[100], 100, 14, 0)
	assertCurrentWorkflowResult(t, byStudent[101], 100, 12, 1)
	assertCurrentWorkflowResult(t, byStudent[102], 101, 16, 0)

	// A human decision updates the same authoritative copy row. The cumulative
	// read model immediately reflects it without touching either import.
	if _, err := conn.Exec(`
		INSERT INTO marking_answer_reviews(answer_detection_id,reviewer_user_id,reviewed_state) VALUES(3001,1,1);
		UPDATE marking_copy_results SET score_half_units=13 WHERE id=1001;
	`); err != nil {
		t.Fatal(err)
	}
	rows, err = queries.ListCurrentExamResultsForMarkingJob(t.Context(), ListCurrentExamResultsForMarkingJobParams{UserID: 1, MarkingJobID: 101})
	if err != nil {
		t.Fatal(err)
	}
	byStudent = make(map[int64]ListCurrentExamResultsForMarkingJobRow, len(rows))
	for _, row := range rows {
		byStudent[row.StudentExamID] = row
	}
	assertCurrentWorkflowResult(t, byStudent[101], 100, 13, 0)
}

func TestRecentMarkingJobsExposeResumableLifecycle(t *testing.T) {
	conn := markingWorkflowTestDB(t)
	defer conn.Close()
	if _, err := conn.Exec(`
		INSERT INTO marking_jobs(id,user_id,exam_generated_id,status,status_pdf,completed_at,review_policy_version,detection_threshold,ambiguity_delta,source_pdf_filename) VALUES
			(200,1,10,'running','running',NULL,'detector-agreement-v1',150,0,'en-cours.pdf'),
			(201,1,10,'success','success',CURRENT_TIMESTAMP,'detector-agreement-v1',150,0,'a-verifier.pdf'),
			(202,1,10,'failed','running',CURRENT_TIMESTAMP,'detector-agreement-v1',150,0,'interrompu.pdf');
		INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,score_half_units,total_points) VALUES
			(2010,1,201,100,'corrected',14,10),
			(2011,1,201,101,'incomplete',NULL,NULL),
			(2012,1,201,102,'not_seen',NULL,NULL);
		INSERT INTO marking_question_results(id,copy_result_id) VALUES(2100,2010);
		INSERT INTO marking_answer_detections(id,question_result_id,review_reason) VALUES(2200,2100,'detector_disagreement');
	`); err != nil {
		t.Fatal(err)
	}
	rows, err := New(conn).ListRecentMarkingJobs(t.Context(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].ID != 202 || rows[1].ID != 201 || rows[2].ID != 200 {
		t.Fatalf("recent job order=%v", []int64{rows[0].ID, rows[1].ID, rows[2].ID})
	}
	if rows[1].CorrectedCopies != 1 || rows[1].IssueCopies != 1 || rows[1].NotSeenCopies != 1 || rows[1].PendingReviews != 1 {
		t.Fatalf("successful job summary=%+v", rows[1])
	}
}

func TestGenerationImportHistoryIsCompleteAndScoped(t *testing.T) {
	conn := markingWorkflowTestDB(t)
	defer conn.Close()
	for id := 1; id <= 25; id++ {
		if _, err := conn.Exec(`INSERT INTO marking_jobs(id,user_id,exam_generated_id,status,status_pdf) VALUES(?,1,10,'success','success')`, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := conn.Exec(`
		INSERT INTO exams_generated VALUES(11,1,1);
		INSERT INTO class_codes VALUES(2,'Autre classe',2);
		INSERT INTO exams VALUES(2,'Autre évaluation',2,2);
		INSERT INTO exams_generated VALUES(20,2,2);
		INSERT INTO marking_jobs(id,user_id,exam_generated_id,status,status_pdf) VALUES(26,1,11,'success','success'),(27,2,20,'success','success');
	`); err != nil {
		t.Fatal(err)
	}
	queries := New(conn)
	history, err := queries.ListMarkingJobHistory(t.Context(), ListMarkingJobHistoryParams{UserID: 1, GenerationID: int64(10)})
	if err != nil || len(history) != 25 {
		t.Fatalf("generation history: count=%d err=%v", len(history), err)
	}
	if history[0].ID != 25 || history[24].ID != 1 {
		t.Fatal("generation history order")
	}
	foreign, err := queries.ListMarkingJobHistory(t.Context(), ListMarkingJobHistoryParams{UserID: 1, GenerationID: int64(20)})
	if err != nil || len(foreign) != 0 {
		t.Fatalf("foreign history: count=%d err=%v", len(foreign), err)
	}
	recent, err := queries.ListRecentMarkingJobs(t.Context(), 1)
	if err != nil || len(recent) != 20 || recent[0].ID != 26 {
		t.Fatalf("recent history: count=%d err=%v", len(recent), err)
	}
}

func assertCurrentWorkflowResult(t *testing.T, row ListCurrentExamResultsForMarkingJobRow, wantJob, wantHalfScore, wantPending int64) {
	t.Helper()
	if row.MarkingJobID != wantJob || row.Outcome != "corrected" || !row.ScoreHalfUnits.Valid || row.ScoreHalfUnits.Int64 != wantHalfScore || row.PendingReviews != wantPending {
		t.Fatalf("current result=%+v, want job=%d score=%d pending=%d", row, wantJob, wantHalfScore, wantPending)
	}
}

func markingWorkflowTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY,username TEXT NOT NULL);
		CREATE TABLE class_codes(id INTEGER PRIMARY KEY,name TEXT NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE exams(id INTEGER PRIMARY KEY,name TEXT NOT NULL,class_code_id INTEGER NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE exams_generated(id INTEGER PRIMARY KEY,exam_id INTEGER NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE students(id INTEGER PRIMARY KEY,first_name TEXT NOT NULL,last_name TEXT NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE student_exam(id INTEGER PRIMARY KEY,exam_generated_id INTEGER NOT NULL,student_id INTEGER NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE marking_jobs(
			id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,exam_generated_id INTEGER,
			status TEXT NOT NULL,status_pdf TEXT NOT NULL,completed_at TIMESTAMP,
			review_policy_version TEXT,detection_threshold REAL,ambiguity_delta REAL,
			source_pdf_filename TEXT
		);
		CREATE TABLE marking_copy_results(
			id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,marking_job_id INTEGER NOT NULL,
			student_exam_id INTEGER NOT NULL,outcome TEXT NOT NULL,score_half_units INTEGER,
			total_points INTEGER,completed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE student_exam_content(student_exam_id INTEGER NOT NULL,user_id INTEGER NOT NULL,content TEXT NOT NULL);
		CREATE TABLE marking_question_results(id INTEGER PRIMARY KEY,copy_result_id INTEGER NOT NULL,question_index INTEGER NOT NULL DEFAULT 0,state TEXT NOT NULL DEFAULT 'incorrect',score_half_units INTEGER NOT NULL DEFAULT 0,total_points INTEGER NOT NULL DEFAULT 1);
		CREATE TABLE marking_answer_detections(
			id INTEGER PRIMARY KEY,question_result_id INTEGER NOT NULL,mean_gray REAL NOT NULL DEFAULT 0,
			review_reason TEXT
		);
		CREATE TABLE marking_answer_reviews(
			id INTEGER PRIMARY KEY,answer_detection_id INTEGER NOT NULL UNIQUE,
			reviewer_user_id INTEGER NOT NULL,reviewed_state INTEGER NOT NULL
		);
		INSERT INTO users VALUES(1,'alice'),(2,'bob');
		INSERT INTO class_codes VALUES(1,'2de A',1);
		INSERT INTO exams VALUES(1,'Devoir réel',1,1);
		INSERT INTO exams_generated VALUES(10,1,1);
		INSERT INTO students VALUES(1,'Alice','Alpha',1),(2,'Basile','Beta',1),(3,'Chloé','Gamma',1);
		INSERT INTO student_exam VALUES(100,10,1,1),(101,10,2,1),(102,10,3,1);
	`); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	return conn
}
