package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

type transactionalReviewFixture struct {
	conn      *sql.DB
	queries   *Queries
	copyID    int64
	questions []MarkingQuestionResult
	answers   map[string]MarkingAnswerDetection
}

func newTransactionalReviewFixture(t *testing.T) transactionalReviewFixture {
	t.Helper()
	conn := markingReviewTestDB(t)
	queries := New(conn)
	qcm := config.QCM{Questions: []config.Question{
		{Tags: config.Tags{Point: config.Point{PointValue: 2}}, Answers: []config.Answer{{State: 1}, {State: 1}, {State: 1}}},
		{Tags: config.Tags{Point: config.Point{PointValue: 2}}, Answers: []config.Answer{{State: 1}, {State: 0}}},
		{Tags: config.Tags{Point: config.Point{PointValue: 1}}, Answers: []config.Answer{{State: 1}}},
	}}
	snapshot, err := json.Marshal(qcm)
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	if _, err := conn.Exec(`
		CREATE TABLE student_exam_content(student_exam_id INTEGER PRIMARY KEY,page_tot INTEGER NOT NULL,content TEXT NOT NULL,user_id INTEGER NOT NULL);
		INSERT INTO student_exam_content(student_exam_id,page_tot,content,user_id) VALUES(1000,2,?,1);
		INSERT INTO marking_jobs(id,user_id,exam_generated_id,status,result_schema_version,marking_algorithm_version,detection_threshold,ambiguity_delta)
		VALUES(300,1,10,'success',1,'1',150,5);
	`, string(snapshot)); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	input := PersistedMarkingCopyInput{
		UserID: 1, MarkingJobID: 300, StudentExamID: 1000, ExpectedPages: 2, DetectedPages: 2,
		ScoreHalfUnits: 6, TotalPoints: 5,
		Questions: []PersistedQuestionInput{
			{QuestionIndex: 0, State: "partial", ScoreHalfUnits: 2, TotalPoints: 2, Answers: []PersistedAnswerDetectionInput{
				{AnswerIndex: 0, DetectedState: 1, MeanGray: 40},
				{AnswerIndex: 1, DetectedState: 0, MeanGray: 210},
				{AnswerIndex: 2, DetectedState: 0, MeanGray: 220},
			}},
			{QuestionIndex: 1, State: "correct", ScoreHalfUnits: 4, TotalPoints: 2, Answers: []PersistedAnswerDetectionInput{
				{AnswerIndex: 0, DetectedState: 1, MeanGray: 40},
				{AnswerIndex: 1, DetectedState: 0, MeanGray: 220},
			}},
			{QuestionIndex: 2, State: "incorrect", ScoreHalfUnits: 0, TotalPoints: 1, Answers: []PersistedAnswerDetectionInput{
				{AnswerIndex: 0, DetectedState: 0, MeanGray: 154.01},
			}},
		},
	}
	copyID, err := PersistCorrectedMarkingCopy(t.Context(), conn, input)
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	questions, err := queries.ListMarkingQuestionResults(t.Context(), ListMarkingQuestionResultsParams{CopyResultID: copyID, UserID: 1})
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	answers := make(map[string]MarkingAnswerDetection)
	for _, question := range questions {
		rows, err := queries.ListMarkingAnswerDetections(t.Context(), ListMarkingAnswerDetectionsParams{QuestionResultID: question.ID, UserID: 1})
		if err != nil {
			conn.Close()
			t.Fatal(err)
		}
		for _, answer := range rows {
			answers[reviewAnswerKey(question.QuestionIndex, answer.AnswerIndex)] = answer
		}
	}
	return transactionalReviewFixture{conn: conn, queries: queries, copyID: copyID, questions: questions, answers: answers}
}

func reviewAnswerKey(question, answer int64) string {
	return string(rune('0'+question)) + ":" + string(rune('0'+answer))
}

func (fixture transactionalReviewFixture) apply(t *testing.T, question, answer, state, answerRevision, jobRevision int64) ApplyMarkingAnswerReviewResult {
	t.Helper()
	result, err := ApplyMarkingAnswerReview(t.Context(), fixture.queries, ApplyMarkingAnswerReviewInput{
		UserID: 1, MarkingJobID: 300, AnswerDetectionID: fixture.answers[reviewAnswerKey(question, answer)].ID,
		ReviewedState: state, ExpectedAnswerReviewRevision: answerRevision, ExpectedJobReviewRevision: jobRevision,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestApplyMarkingAnswerReviewReal154Regression(t *testing.T) {
	fixture := newTransactionalReviewFixture(t)
	defer fixture.conn.Close()
	result := fixture.apply(t, 2, 0, 1, 0, 0)
	if result.DetectedState != 0 || result.EffectiveState != 1 || result.QuestionState != "correct" ||
		result.QuestionScoreHalfUnits != 2 || result.CopyScoreHalfUnits != 8 ||
		result.JobReviewRevision != 1 || result.ArtifactsRevision != 0 {
		t.Fatalf("result=%+v", result)
	}
	detection := fixture.answers[reviewAnswerKey(2, 0)]
	effective, err := fixture.queries.GetEffectiveAnswerDetection(t.Context(), GetEffectiveAnswerDetectionParams{AnswerDetectionID: detection.ID, UserID: 1})
	if err != nil || effective.DetectedState != 0 || effective.ReviewedState.Int64 != 1 || effective.EffectiveState != 1 {
		t.Fatalf("effective=%+v err=%v", effective, err)
	}
	summary, err := fixture.queries.GetMarkingReviewSummary(t.Context(), GetMarkingReviewSummaryParams{MarkingJobID: 300, UserID: 1})
	if err != nil || summary.TotalCandidates != 1 || summary.ReviewedCandidates != 1 || summary.PendingCandidates != 0 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
}

func TestApplyMarkingAnswerReviewConfirmationAndIdempotence(t *testing.T) {
	fixture := newTransactionalReviewFixture(t)
	defer fixture.conn.Close()
	before := markingCopyScore(t, fixture)
	result := fixture.apply(t, 1, 0, 1, 0, 0)
	if result.EffectiveState != 1 || result.CopyScoreHalfUnits != before || result.JobReviewRevision != 1 || result.ArtifactsRevision != 1 {
		t.Fatalf("confirmation=%+v before=%d", result, before)
	}
	noOp := fixture.apply(t, 1, 0, 1, 1, 1)
	if !noOp.NoOp || noOp.JobReviewRevision != 1 || noOp.ArtifactsRevision != 1 || noOp.AnswerReviewRevision != 1 {
		t.Fatalf("idempotent result=%+v", noOp)
	}
}

func TestApplyMarkingAnswerReviewNonAmbiguousAndMultipleQuestions(t *testing.T) {
	fixture := newTransactionalReviewFixture(t)
	defer fixture.conn.Close()
	q0Before := fixture.questions[0]
	result := fixture.apply(t, 1, 0, 0, 0, 0)
	if result.QuestionState != "incorrect" || result.QuestionScoreHalfUnits != 0 || result.CopyScoreHalfUnits != 2 || result.ArtifactsRevision != 0 {
		t.Fatalf("non-ambiguous override=%+v", result)
	}
	questions, err := fixture.queries.ListMarkingQuestionResults(t.Context(), ListMarkingQuestionResultsParams{CopyResultID: fixture.copyID, UserID: 1})
	if err != nil || questions[0].State != q0Before.State || questions[0].ScoreHalfUnits != q0Before.ScoreHalfUnits {
		t.Fatalf("unrelated question changed: %+v err=%v", questions, err)
	}
	summary, err := fixture.queries.GetMarkingReviewSummary(t.Context(), GetMarkingReviewSummaryParams{MarkingJobID: 300, UserID: 1})
	if err != nil || summary.TotalCandidates != 1 || summary.ReviewedCandidates != 0 || summary.PendingCandidates != 1 {
		t.Fatalf("non-ambiguous summary=%+v err=%v", summary, err)
	}
}

func TestApplyMarkingAnswerReviewEffectiveVectorAndArtifactMonotonicity(t *testing.T) {
	fixture := newTransactionalReviewFixture(t)
	defer fixture.conn.Close()
	// Q0 remains partial although an effective answer changes: artifacts still
	// become stale because the correction marks, not only the score, changed.
	first := fixture.apply(t, 0, 1, 1, 0, 0)
	if first.QuestionState != "partial" || first.QuestionScoreHalfUnits != 2 || first.CopyScoreHalfUnits != 6 || first.ArtifactsRevision != 0 {
		t.Fatalf("same-score override=%+v", first)
	}
	// A confirmation while already stale must never catch artifacts up.
	confirmation := fixture.apply(t, 1, 0, 1, 0, 1)
	if confirmation.JobReviewRevision != 2 || confirmation.ArtifactsRevision != 0 {
		t.Fatalf("stale confirmation=%+v", confirmation)
	}
	// The previous override remains effective when the next answer is reviewed.
	second := fixture.apply(t, 0, 2, 1, 0, 2)
	if second.QuestionState != "correct" || second.QuestionScoreHalfUnits != 4 || second.CopyScoreHalfUnits != 8 || second.ArtifactsRevision != 0 {
		t.Fatalf("multi-review=%+v", second)
	}
}

func TestApplyMarkingAnswerReviewOwnershipConcurrencyAndRollback(t *testing.T) {
	fixture := newTransactionalReviewFixture(t)
	defer fixture.conn.Close()
	target := fixture.answers[reviewAnswerKey(0, 1)]
	for name, input := range map[string]ApplyMarkingAnswerReviewInput{
		"Bob":       {UserID: 2, MarkingJobID: 300, AnswerDetectionID: target.ID, ReviewedState: 1},
		"wrong job": {UserID: 1, MarkingJobID: 200, AnswerDetectionID: target.ID, ReviewedState: 1},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ApplyMarkingAnswerReview(t.Context(), fixture.queries, input); !errors.Is(err, ErrMarkingReviewUnavailable) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	// The stale global revision fails only at publication, after the transaction
	// has tentatively inserted/recalculated; rollback must remove every change.
	_, err := ApplyMarkingAnswerReview(t.Context(), fixture.queries, ApplyMarkingAnswerReviewInput{
		UserID: 1, MarkingJobID: 300, AnswerDetectionID: target.ID, ReviewedState: 1, ExpectedJobReviewRevision: 99,
	})
	if !errors.Is(err, ErrMarkingReviewConflict) {
		t.Fatalf("stale job revision error=%v", err)
	}
	assertNoReviewAndOriginalScore(t, fixture, target.ID, 6)

	first := fixture.apply(t, 0, 1, 1, 0, 0)
	if first.JobReviewRevision != 1 {
		t.Fatalf("first=%+v", first)
	}
	if _, err := ApplyMarkingAnswerReview(t.Context(), fixture.queries, ApplyMarkingAnswerReviewInput{
		UserID: 1, MarkingJobID: 300, AnswerDetectionID: target.ID, ReviewedState: 0,
		ExpectedAnswerReviewRevision: 0, ExpectedJobReviewRevision: 1,
	}); !errors.Is(err, ErrMarkingReviewConflict) {
		t.Fatalf("same-answer stale error=%v", err)
	}
	other := fixture.answers[reviewAnswerKey(0, 2)]
	if _, err := ApplyMarkingAnswerReview(t.Context(), fixture.queries, ApplyMarkingAnswerReviewInput{
		UserID: 1, MarkingJobID: 300, AnswerDetectionID: other.ID, ReviewedState: 1, ExpectedJobReviewRevision: 0,
	}); !errors.Is(err, ErrMarkingReviewConflict) {
		t.Fatalf("different-answer stale job error=%v", err)
	}
	if _, err := fixture.queries.GetMarkingAnswerReview(t.Context(), GetMarkingAnswerReviewParams{AnswerDetectionID: other.ID, UserID: 1}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("stale different-answer review survived: %v", err)
	}
	retry := fixture.apply(t, 0, 2, 1, 0, 1)
	if retry.JobReviewRevision != 2 {
		t.Fatalf("retry=%+v", retry)
	}
}

func TestApplyMarkingAnswerReviewRejectsNonSuccessfulWorkflow(t *testing.T) {
	for _, status := range []string{"running", "failed"} {
		t.Run(status, func(t *testing.T) {
			fixture := newTransactionalReviewFixture(t)
			defer fixture.conn.Close()
			if _, err := fixture.conn.Exec(`UPDATE marking_jobs SET status=? WHERE id=300`, status); err != nil {
				t.Fatal(err)
			}
			target := fixture.answers[reviewAnswerKey(2, 0)]
			if _, err := ApplyMarkingAnswerReview(t.Context(), fixture.queries, ApplyMarkingAnswerReviewInput{
				UserID: 1, MarkingJobID: 300, AnswerDetectionID: target.ID, ReviewedState: 1,
			}); !errors.Is(err, ErrMarkingReviewUnavailable) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	for _, outcome := range []string{"incomplete", "not_seen", "error"} {
		t.Run(outcome, func(t *testing.T) {
			fixture := newTransactionalReviewFixture(t)
			defer fixture.conn.Close()
			detectedPages := 2
			if outcome == "incomplete" {
				detectedPages = 1
			} else if outcome == "not_seen" {
				detectedPages = 0
			}
			if _, err := fixture.conn.Exec(`UPDATE marking_copy_results SET outcome=?,detected_pages=?,score_half_units=NULL,total_points=NULL,failure_code='synthetic' WHERE id=?`, outcome, detectedPages, fixture.copyID); err != nil {
				t.Fatal(err)
			}
			target := fixture.answers[reviewAnswerKey(2, 0)]
			if _, err := ApplyMarkingAnswerReview(t.Context(), fixture.queries, ApplyMarkingAnswerReviewInput{
				UserID: 1, MarkingJobID: 300, AnswerDetectionID: target.ID, ReviewedState: 1,
			}); !errors.Is(err, ErrMarkingReviewUnavailable) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func markingCopyScore(t *testing.T, fixture transactionalReviewFixture) int64 {
	t.Helper()
	copyResult, err := fixture.queries.GetMarkingCopyResult(t.Context(), GetMarkingCopyResultParams{ID: fixture.copyID, UserID: 1})
	if err != nil || !copyResult.ScoreHalfUnits.Valid {
		t.Fatalf("copy=%+v err=%v", copyResult, err)
	}
	return copyResult.ScoreHalfUnits.Int64
}

func assertNoReviewAndOriginalScore(t *testing.T, fixture transactionalReviewFixture, answerDetectionID, wantScore int64) {
	t.Helper()
	if _, err := fixture.queries.GetMarkingAnswerReview(t.Context(), GetMarkingAnswerReviewParams{AnswerDetectionID: answerDetectionID, UserID: 1}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("review survived rollback: %v", err)
	}
	if got := markingCopyScore(t, fixture); got != wantScore {
		t.Fatalf("score after rollback=%d, want %d", got, wantScore)
	}
	var reviewRevision, artifactsRevision int64
	if err := fixture.conn.QueryRow(`SELECT review_revision,artifacts_revision FROM marking_jobs WHERE id=300`).Scan(&reviewRevision, &artifactsRevision); err != nil || reviewRevision != 0 || artifactsRevision != 0 {
		t.Fatalf("revisions after rollback=(%d,%d) err=%v", reviewRevision, artifactsRevision, err)
	}
}
