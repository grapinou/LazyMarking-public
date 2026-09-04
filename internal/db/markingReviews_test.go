package db

import (
	"database/sql"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestMarkingReviewMigrationLegacyAndRevisionChecks(t *testing.T) {
	conn := markingReviewTestDB(t)
	defer conn.Close()

	var delta sql.NullFloat64
	var reviewRevision, artifactsRevision int64
	if err := conn.QueryRow(`SELECT ambiguity_delta, review_revision, artifacts_revision FROM marking_jobs WHERE id=90`).Scan(&delta, &reviewRevision, &artifactsRevision); err != nil {
		t.Fatal(err)
	}
	if delta.Valid || reviewRevision != 0 || artifactsRevision != 0 {
		t.Fatalf("legacy job delta=%v review=%d artifacts=%d", delta, reviewRevision, artifactsRevision)
	}
	for _, table := range []string{"marking_answer_reviews", "marking_aligned_pages"} {
		var count int64
		if err := conn.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("legacy migration table %s count=%d err=%v", table, count, err)
		}
	}
	for name, statement := range map[string]string{
		"negative delta":     `UPDATE marking_jobs SET ambiguity_delta=-0.1 WHERE id=100`,
		"negative review":    `UPDATE marking_jobs SET review_revision=-1 WHERE id=100`,
		"negative artifacts": `UPDATE marking_jobs SET artifacts_revision=-1 WHERE id=100`,
		"artifacts ahead":    `UPDATE marking_jobs SET artifacts_revision=1 WHERE id=100`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := conn.Exec(statement); err == nil {
				t.Fatal("invalid revision metadata succeeded")
			}
		})
	}
	if _, err := conn.Exec(`UPDATE marking_jobs SET ambiguity_delta=10 WHERE id=100`); err != nil {
		t.Fatalf("activate ambiguity delta: %v", err)
	}
	if _, err := conn.Exec(`UPDATE marking_jobs SET ambiguity_delta=11 WHERE id=100`); err == nil {
		t.Fatal("initialized ambiguity delta changed")
	}
	if _, err := conn.Exec(`UPDATE marking_jobs SET review_revision=2, artifacts_revision=2 WHERE id=100`); err != nil {
		t.Fatalf("coherent revisions: %v", err)
	}

	_, down := markingReviewMigration(t)
	if _, err := conn.Exec(down); err != nil {
		t.Fatalf("migration Down: %v", err)
	}
	for _, table := range []string{"marking_answer_reviews", "marking_aligned_pages"} {
		if _, err := conn.Exec(`SELECT * FROM ` + table); err == nil {
			t.Fatalf("table %s remains after Down", table)
		}
	}
	if _, err := conn.Exec(`SELECT ambiguity_delta FROM marking_jobs`); err == nil {
		t.Fatal("marking_jobs review columns remain after Down")
	}
	var legacyUser int64
	if err := conn.QueryRow(`SELECT user_id FROM marking_jobs WHERE id=90`).Scan(&legacyUser); err != nil || legacyUser != 1 {
		t.Fatalf("legacy job after Down user=%d err=%v", legacyUser, err)
	}
}

func TestMarkingArtifactsRevisionAdvanceIsOwnershipAndRevisionAware(t *testing.T) {
	conn := markingReviewTestDB(t)
	defer conn.Close()
	queries := New(conn)
	if _, err := conn.Exec(`UPDATE marking_jobs SET status='success', ambiguity_delta=5, review_revision=3, artifacts_revision=1, exam_name='corrected.pdf', mark_table_name='mark-table.pdf' WHERE id=100`); err != nil {
		t.Fatal(err)
	}
	target, err := queries.GetMarkingArtifactsRegenerationTarget(t.Context(), GetMarkingArtifactsRegenerationTargetParams{MarkingJobID: 100, UserID: 1})
	if err != nil || target.ReviewRevision != 3 || target.ArtifactsRevision != 1 || !target.AmbiguityDelta.Valid {
		t.Fatalf("target=%+v err=%v", target, err)
	}
	if _, err := queries.GetMarkingArtifactsRegenerationTarget(t.Context(), GetMarkingArtifactsRegenerationTargetParams{MarkingJobID: 100, UserID: 2}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-user target err=%v", err)
	}
	if _, err := queries.GetMarkingArtifactsRegenerationTarget(t.Context(), GetMarkingArtifactsRegenerationTargetParams{MarkingJobID: 101, UserID: 1}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("running target err=%v", err)
	}
	for name, params := range map[string]AdvanceMarkingArtifactsRevisionParams{
		"cross-user":      {MarkingJobID: 100, UserID: 2, ExpectedReviewRevision: 3, ExpectedArtifactsRevision: 1},
		"stale-review":    {MarkingJobID: 100, UserID: 1, ExpectedReviewRevision: 2, ExpectedArtifactsRevision: 1},
		"stale-artifacts": {MarkingJobID: 100, UserID: 1, ExpectedReviewRevision: 3, ExpectedArtifactsRevision: 0},
	} {
		t.Run(name, func(t *testing.T) {
			rows, err := queries.AdvanceMarkingArtifactsRevision(t.Context(), params)
			if err != nil || rows != 0 {
				t.Fatalf("rows=%d err=%v", rows, err)
			}
		})
	}
	rows, err := queries.AdvanceMarkingArtifactsRevision(t.Context(), AdvanceMarkingArtifactsRevisionParams{MarkingJobID: 100, UserID: 1, ExpectedReviewRevision: 3, ExpectedArtifactsRevision: 1})
	if err != nil || rows != 1 {
		t.Fatalf("rows=%d err=%v", rows, err)
	}
	if rows, err := queries.AdvanceMarkingArtifactsRevision(t.Context(), AdvanceMarkingArtifactsRevisionParams{MarkingJobID: 100, UserID: 1, ExpectedReviewRevision: 3, ExpectedArtifactsRevision: 1}); err != nil || rows != 0 {
		t.Fatalf("second advance rows=%d err=%v", rows, err)
	}
}

func TestMarkingAnswerReviewEffectiveStateOwnershipAndConcurrency(t *testing.T) {
	conn := markingReviewTestDB(t)
	defer conn.Close()
	queries := New(conn)
	copyID := createCorrectedCopyResult(t, queries, 100, 1000)
	questionID, err := queries.CreateMarkingQuestionResult(t.Context(), CreateMarkingQuestionResultParams{
		CopyResultID: copyID, QuestionIndex: 0, State: "correct", ScoreHalfUnits: 4, TotalPoints: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	uncheckedID, err := queries.CreateMarkingAnswerDetection(t.Context(), CreateMarkingAnswerDetectionParams{
		QuestionResultID: questionID, AnswerIndex: 0, DetectedState: 0, MeanGray: 210,
	})
	if err != nil {
		t.Fatal(err)
	}
	clearCheckedID, err := queries.CreateMarkingAnswerDetection(t.Context(), CreateMarkingAnswerDetectionParams{
		QuestionResultID: questionID, AnswerIndex: 1, DetectedState: 1, MeanGray: 40,
	})
	if err != nil {
		t.Fatal(err)
	}
	confirmedID, err := queries.CreateMarkingAnswerDetection(t.Context(), CreateMarkingAnswerDetectionParams{
		QuestionResultID: questionID, AnswerIndex: 2, DetectedState: 1, MeanGray: 149,
	})
	if err != nil {
		t.Fatal(err)
	}

	before, err := queries.GetEffectiveAnswerDetection(t.Context(), GetEffectiveAnswerDetectionParams{AnswerDetectionID: uncheckedID, UserID: 1})
	if err != nil || before.DetectedState != 0 || before.ReviewedState.Valid || before.EffectiveState != 0 {
		t.Fatalf("before review=%+v err=%v", before, err)
	}
	if _, err := queries.CreateMarkingAnswerReview(t.Context(), CreateMarkingAnswerReviewParams{
		ReviewerUserID: 1, ReviewedState: 1, AnswerDetectionID: uncheckedID,
	}); err != nil {
		t.Fatal(err)
	}
	after, err := queries.GetEffectiveAnswerDetection(t.Context(), GetEffectiveAnswerDetectionParams{AnswerDetectionID: uncheckedID, UserID: 1})
	if err != nil || after.DetectedState != 0 || after.ReviewedState.Int64 != 1 || after.EffectiveState != 1 || after.Revision.Int64 != 1 {
		t.Fatalf("after review=%+v err=%v", after, err)
	}

	// Mean 40 is intentionally far from any plausible narrow band: review is
	// authorized by ownership, never by ambiguity.
	if _, err := queries.CreateMarkingAnswerReview(t.Context(), CreateMarkingAnswerReviewParams{
		ReviewerUserID: 1, ReviewedState: 0, AnswerDetectionID: clearCheckedID,
	}); err != nil {
		t.Fatalf("review non-ambiguous detection: %v", err)
	}
	clearAfter, err := queries.GetEffectiveAnswerDetection(t.Context(), GetEffectiveAnswerDetectionParams{AnswerDetectionID: clearCheckedID, UserID: 1})
	if err != nil || clearAfter.DetectedState != 1 || clearAfter.EffectiveState != 0 {
		t.Fatalf("non-ambiguous review=%+v err=%v", clearAfter, err)
	}
	if _, err := conn.Exec(`INSERT INTO marking_answer_reviews(answer_detection_id,reviewer_user_id,reviewed_state) VALUES(?,2,1)`, confirmedID); err == nil {
		t.Fatal("direct Bob review of Alice detection succeeded")
	}
	if _, err := queries.CreateMarkingAnswerReview(t.Context(), CreateMarkingAnswerReviewParams{
		ReviewerUserID: 1, ReviewedState: 1, AnswerDetectionID: confirmedID,
	}); err != nil {
		t.Fatal(err)
	}
	confirmed, err := queries.GetEffectiveAnswerDetection(t.Context(), GetEffectiveAnswerDetectionParams{AnswerDetectionID: confirmedID, UserID: 1})
	if err != nil || !confirmed.ReviewedState.Valid || confirmed.DetectedState != 1 || confirmed.EffectiveState != 1 {
		t.Fatalf("confirmed review=%+v err=%v", confirmed, err)
	}

	if _, err := queries.CreateMarkingAnswerReview(t.Context(), CreateMarkingAnswerReviewParams{
		ReviewerUserID: 1, ReviewedState: 0, AnswerDetectionID: uncheckedID,
	}); err == nil {
		t.Fatal("duplicate first review succeeded")
	}
	if _, err := queries.CreateMarkingAnswerReview(t.Context(), CreateMarkingAnswerReviewParams{
		ReviewerUserID: 2, ReviewedState: 1, AnswerDetectionID: uncheckedID,
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Bob SQLC review error=%v, want sql.ErrNoRows", err)
	}
	if _, err := conn.Exec(`INSERT INTO marking_answer_reviews(answer_detection_id,reviewer_user_id,reviewed_state) VALUES(?,2,1)`, clearCheckedID+1000); err == nil {
		t.Fatal("direct incoherent review succeeded")
	}
	if _, err := conn.Exec(`UPDATE marking_answer_reviews SET reviewer_user_id=2 WHERE answer_detection_id=?`, uncheckedID); err == nil {
		t.Fatal("review owner changed to Bob")
	}
	if _, err := conn.Exec(`UPDATE marking_answer_reviews SET reviewed_state=0, revision=3 WHERE answer_detection_id=?`, uncheckedID); err == nil {
		t.Fatal("review revision skipped directly")
	}

	rows, err := queries.UpdateMarkingAnswerReview(t.Context(), UpdateMarkingAnswerReviewParams{
		ReviewedState: 0, AnswerDetectionID: uncheckedID, ReviewerUserID: 1, ExpectedRevision: 1,
	})
	if err != nil || rows != 1 {
		t.Fatalf("optimistic update rows=%d err=%v", rows, err)
	}
	rows, err = queries.UpdateMarkingAnswerReview(t.Context(), UpdateMarkingAnswerReviewParams{
		ReviewedState: 1, AnswerDetectionID: uncheckedID, ReviewerUserID: 1, ExpectedRevision: 1,
	})
	if err != nil || rows != 0 {
		t.Fatalf("stale update rows=%d err=%v", rows, err)
	}
	final, err := queries.GetMarkingAnswerReview(t.Context(), GetMarkingAnswerReviewParams{AnswerDetectionID: uncheckedID, UserID: 1})
	if err != nil || final.Revision != 2 || final.ReviewedState != 0 {
		t.Fatalf("final review=%+v err=%v", final, err)
	}
	if _, err := queries.GetMarkingAnswerReview(t.Context(), GetMarkingAnswerReviewParams{AnswerDetectionID: uncheckedID, UserID: 2}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Bob read error=%v", err)
	}

	var detectedState int64
	if err := conn.QueryRow(`SELECT detected_state FROM marking_answer_detections WHERE id=?`, uncheckedID).Scan(&detectedState); err != nil || detectedState != 0 {
		t.Fatalf("automatic detection changed state=%d err=%v", detectedState, err)
	}
}

func TestMarkingReviewCandidateReadModel(t *testing.T) {
	conn := markingReviewTestDB(t)
	defer conn.Close()
	queries := New(conn)
	if _, err := conn.Exec(`INSERT INTO marking_jobs(id,user_id,exam_generated_id,status,result_schema_version,marking_algorithm_version,detection_threshold,ambiguity_delta) VALUES(300,1,10,'success',1,'1',150,5)`); err != nil {
		t.Fatal(err)
	}
	copyID := createCorrectedCopyResult(t, queries, 300, 1000)
	questionID, err := queries.CreateMarkingQuestionResult(t.Context(), CreateMarkingQuestionResultParams{
		CopyResultID: copyID, QuestionIndex: 0, State: "correct", ScoreHalfUnits: 2, TotalPoints: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	type detectionFixture struct {
		mean  float64
		state int64
	}
	fixtures := []detectionFixture{
		{mean: 144.99, state: 1}, {mean: 145, state: 1}, {mean: 149.99, state: 1},
		{mean: 150, state: 0}, {mean: 154.01, state: 0}, {mean: 155, state: 0},
		{mean: 155.01, state: 0}, {mean: 40, state: 1},
	}
	ids := make(map[float64]int64, len(fixtures))
	for index, fixture := range fixtures {
		id, err := queries.CreateMarkingAnswerDetection(t.Context(), CreateMarkingAnswerDetectionParams{
			QuestionResultID: questionID, AnswerIndex: int64(index), DetectedState: fixture.state, MeanGray: fixture.mean,
		})
		if err != nil {
			t.Fatal(err)
		}
		ids[fixture.mean] = id
	}
	if _, err := queries.CreateMarkingAlignedPage(t.Context(), CreateMarkingAlignedPageParams{
		UserID: 1, CopyResultID: copyID, PageExam: 1, StorageKey: "aligned/student-exam-1000/page-1.png",
		Width: 100, Height: 100, Sha256: strings.Repeat("a", 64),
	}); err != nil {
		t.Fatal(err)
	}
	candidates, err := queries.ListMarkingReviewCandidates(t.Context(), ListMarkingReviewCandidatesParams{MarkingJobID: 300, UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	wantMeans := []float64{145, 149.99, 150, 154.01, 155}
	if len(candidates) != len(wantMeans) {
		t.Fatalf("candidate count=%d, want %d", len(candidates), len(wantMeans))
	}
	for index, candidate := range candidates {
		if candidate.MeanGray != wantMeans[index] || candidate.Threshold.Float64 != 150 || candidate.AmbiguityDelta.Float64 != 5 ||
			candidate.PageExam != 1 || candidate.StorageKey != "aligned/student-exam-1000/page-1.png" {
			t.Fatalf("candidate[%d]=%+v", index, candidate)
		}
		if candidate.MeanGray == 154.01 && candidate.DetectedState != 0 {
			t.Fatal("154.01 historical detected state changed")
		}
	}
	if foreign, err := queries.ListMarkingReviewCandidates(t.Context(), ListMarkingReviewCandidatesParams{MarkingJobID: 300, UserID: 2}); err != nil || len(foreign) != 0 {
		t.Fatalf("Bob candidates=%d err=%v", len(foreign), err)
	}

	// A confirming review and an override are both completed decisions.
	for _, review := range []struct {
		mean  float64
		state int64
	}{{mean: 145, state: 1}, {mean: 154.01, state: 1}, {mean: 40, state: 0}} {
		if _, err := queries.CreateMarkingAnswerReview(t.Context(), CreateMarkingAnswerReviewParams{
			AnswerDetectionID: ids[review.mean], ReviewerUserID: 1, ReviewedState: review.state,
		}); err != nil {
			t.Fatal(err)
		}
	}
	reviewedCandidates, err := queries.ListMarkingReviewCandidates(t.Context(), ListMarkingReviewCandidatesParams{MarkingJobID: 300, UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range reviewedCandidates {
		switch candidate.MeanGray {
		case 145:
			if candidate.ReviewedState.Int64 != 1 || candidate.EffectiveState != 1 {
				t.Fatalf("confirmed candidate=%+v", candidate)
			}
		case 154.01:
			if candidate.DetectedState != 0 || candidate.ReviewedState.Int64 != 1 || candidate.EffectiveState != 1 {
				t.Fatalf("override candidate=%+v", candidate)
			}
		}
	}
	summary, err := queries.GetMarkingReviewSummary(t.Context(), GetMarkingReviewSummaryParams{MarkingJobID: 300, UserID: 1})
	if err != nil || summary.TotalCandidates != 5 || summary.ReviewedCandidates != 2 || summary.PendingCandidates != 3 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	status, err := DeriveMarkingReviewStatus(summary.AmbiguityDelta, summary.TotalCandidates, summary.PendingCandidates)
	if err != nil || status != MarkingReviewPending {
		t.Fatalf("status=%q err=%v", status, err)
	}
	pending, err := queries.ListPendingMarkingReviewCandidates(t.Context(), ListPendingMarkingReviewCandidatesParams{MarkingJobID: 300, UserID: 1})
	if err != nil || len(pending) != 3 {
		t.Fatalf("pending=%d err=%v", len(pending), err)
	}
	for _, candidate := range candidates {
		if !candidate.ReviewedState.Valid && candidate.MeanGray != 145 && candidate.MeanGray != 154.01 {
			if _, err := queries.CreateMarkingAnswerReview(t.Context(), CreateMarkingAnswerReviewParams{
				AnswerDetectionID: candidate.AnswerDetectionID, ReviewerUserID: 1, ReviewedState: candidate.DetectedState,
			}); err != nil {
				t.Fatal(err)
			}
		}
	}
	summary, err = queries.GetMarkingReviewSummary(t.Context(), GetMarkingReviewSummaryParams{MarkingJobID: 300, UserID: 1})
	if err != nil || summary.TotalCandidates != 5 || summary.ReviewedCandidates != 5 || summary.PendingCandidates != 0 {
		t.Fatalf("completed summary=%+v err=%v", summary, err)
	}
	status, err = DeriveMarkingReviewStatus(summary.AmbiguityDelta, summary.TotalCandidates, summary.PendingCandidates)
	if err != nil || status != MarkingReviewCompleted {
		t.Fatalf("completed status=%q err=%v", status, err)
	}
	var reviewRevision, artifactsRevision int64
	if err := conn.QueryRow(`SELECT review_revision, artifacts_revision FROM marking_jobs WHERE id=300`).Scan(&reviewRevision, &artifactsRevision); err != nil || reviewRevision != 0 || artifactsRevision != 0 {
		t.Fatalf("revisions=(%d,%d) err=%v", reviewRevision, artifactsRevision, err)
	}
	if _, err := queries.GetMarkingReviewSummary(t.Context(), GetMarkingReviewSummaryParams{MarkingJobID: 300, UserID: 2}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Bob summary error=%v", err)
	}
}

func TestMarkingReviewLegacyAndSuccessfulJobBoundary(t *testing.T) {
	conn := markingReviewTestDB(t)
	defer conn.Close()
	queries := New(conn)
	if _, err := conn.Exec(`UPDATE marking_jobs SET status='success' WHERE id=90`); err != nil {
		t.Fatal(err)
	}
	legacy, err := queries.GetMarkingReviewSummary(t.Context(), GetMarkingReviewSummaryParams{MarkingJobID: 90, UserID: 1})
	if err != nil || legacy.AmbiguityDelta.Valid || legacy.TotalCandidates != 0 || legacy.PendingCandidates != 0 {
		t.Fatalf("legacy summary=%+v err=%v", legacy, err)
	}
	status, err := DeriveMarkingReviewStatus(legacy.AmbiguityDelta, legacy.TotalCandidates, legacy.PendingCandidates)
	if err != nil || status != MarkingReviewUnavailable {
		t.Fatalf("legacy status=%q err=%v", status, err)
	}
	if candidates, err := queries.ListMarkingReviewCandidates(t.Context(), ListMarkingReviewCandidatesParams{MarkingJobID: 90, UserID: 1}); err != nil || len(candidates) != 0 {
		t.Fatalf("legacy candidates=%d err=%v", len(candidates), err)
	}
	if _, err := conn.Exec(`UPDATE marking_jobs SET ambiguity_delta=5 WHERE id=101`); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.GetMarkingReviewSummary(t.Context(), GetMarkingReviewSummaryParams{MarkingJobID: 101, UserID: 1}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("running job summary error=%v", err)
	}
}

func TestHybridReviewCandidatesAreDetectorDisagreements(t *testing.T) {
	conn := markingReviewTestDB(t)
	defer conn.Close()
	queries := New(conn)
	result, err := conn.Exec(`INSERT INTO marking_jobs(user_id,exam_generated_id,status,result_schema_version,marking_algorithm_version,detection_threshold,ambiguity_delta,review_policy_version,v2_roi_radius_ratio,v2_dark_pixel_threshold,v2_dark_ratio_threshold,v2_chroma_pixel_threshold,v2_chroma_ratio_threshold) VALUES(1,10,'success',2,'hybrid-historical-v2-frozen-1',150,0,'detector-agreement-v1',.4,220,.1,12,.05)`)
	if err != nil {
		t.Fatal(err)
	}
	jobID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	copyID := createCorrectedCopyResult(t, queries, jobID, 1000)
	questionID, err := queries.CreateMarkingQuestionResult(t.Context(), CreateMarkingQuestionResultParams{CopyResultID: copyID, QuestionIndex: 0, State: "incorrect", ScoreHalfUnits: 0, TotalPoints: 1})
	if err != nil {
		t.Fatal(err)
	}
	var disagreementIDs []int64
	for index, states := range [][2]int64{{0, 0}, {1, 1}, {0, 1}, {1, 0}} {
		review := states[0] != states[1]
		reason := sql.NullString{String: "detector_disagreement", Valid: review}
		colorSignal := int64(0)
		grayscaleSignal := states[1]
		if index == 2 {
			colorSignal = 1
			grayscaleSignal = 0
		}
		id, err := queries.CreateHybridMarkingAnswerDetection(t.Context(), CreateHybridMarkingAnswerDetectionParams{
			QuestionResultID: questionID, AnswerIndex: int64(index), DetectedState: states[0], MeanGray: map[bool]float64{true: 100, false: 200}[states[0] == 1],
			HistoricalState: sql.NullInt64{Int64: states[0], Valid: true}, V2State: sql.NullInt64{Int64: states[1], Valid: true},
			DarkRatio: sql.NullFloat64{Float64: float64(states[1]), Valid: true}, ChromaRatio: sql.NullFloat64{Float64: 0, Valid: true},
			GrayscaleSignal: sql.NullInt64{Int64: grayscaleSignal, Valid: true}, ColorSignal: sql.NullInt64{Int64: colorSignal, Valid: true}, ReviewReason: reason,
			AutomaticState: sql.NullInt64{Int64: states[0], Valid: !review},
		})
		if err != nil {
			t.Fatal(err)
		}
		if review {
			disagreementIDs = append(disagreementIDs, id)
		}
	}
	if _, err := queries.CreateMarkingAlignedPage(t.Context(), CreateMarkingAlignedPageParams{UserID: 1, CopyResultID: copyID, PageExam: 1, StorageKey: "aligned/student-exam-1000/page-1.png", Width: 100, Height: 100, Sha256: strings.Repeat("b", 64)}); err != nil {
		t.Fatal(err)
	}
	candidates, err := queries.ListPendingMarkingReviewCandidates(t.Context(), ListPendingMarkingReviewCandidatesParams{MarkingJobID: jobID, UserID: 1})
	if err != nil || len(candidates) != 2 || candidates[0].AnswerDetectionID != disagreementIDs[0] || candidates[1].AnswerDetectionID != disagreementIDs[1] {
		t.Fatalf("hybrid candidates=%+v err=%v", candidates, err)
	}
	if _, err := queries.CreateMarkingAnswerReview(t.Context(), CreateMarkingAnswerReviewParams{AnswerDetectionID: disagreementIDs[0], ReviewerUserID: 1, ReviewedState: 1}); err != nil {
		t.Fatal(err)
	}
	effective, err := queries.GetEffectiveAnswerDetection(t.Context(), GetEffectiveAnswerDetectionParams{AnswerDetectionID: disagreementIDs[0], UserID: 1})
	if err != nil || effective.DetectedState != 0 || effective.EffectiveState != 1 {
		t.Fatalf("human override=%+v err=%v", effective, err)
	}
}

func TestColorConfidenceReviewCandidates(t *testing.T) {
	conn := markingReviewTestDB(t)
	defer conn.Close()
	queries := New(conn)
	result, err := conn.Exec(`INSERT INTO marking_jobs(user_id,exam_generated_id,status,result_schema_version,marking_algorithm_version,detection_threshold,ambiguity_delta,review_policy_version,v2_roi_radius_ratio,v2_dark_pixel_threshold,v2_dark_ratio_threshold,v2_chroma_pixel_threshold,v2_chroma_ratio_threshold) VALUES(1,10,'success',2,'hybrid-historical-v2-frozen-1',150,0,'detector-color-confidence-v1',.4,220,.1,12,.05)`)
	if err != nil {
		t.Fatal(err)
	}
	jobID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	copyID := createCorrectedCopyResult(t, queries, jobID, 1000)
	questionID, err := queries.CreateMarkingQuestionResult(t.Context(), CreateMarkingQuestionResultParams{CopyResultID: copyID, QuestionIndex: 0, State: "incorrect", ScoreHalfUnits: 0, TotalPoints: 1})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		historical, v2, grayscale, color int64
		automatic                        sql.NullInt64
		review                           bool
	}{
		{historical: 0, v2: 0, automatic: sql.NullInt64{Int64: 0, Valid: true}},
		{historical: 1, v2: 1, grayscale: 1, automatic: sql.NullInt64{Int64: 1, Valid: true}},
		{historical: 1, v2: 1, color: 1, automatic: sql.NullInt64{Int64: 1, Valid: true}},
		{historical: 0, v2: 1, color: 1, automatic: sql.NullInt64{Int64: 1, Valid: true}},
		{historical: 0, v2: 1, grayscale: 1, review: true},
		{historical: 1, v2: 0, review: true},
	}
	var wantCandidates []int64
	var colorConfidenceID int64
	for index, test := range tests {
		reason := sql.NullString{String: "detector_disagreement", Valid: test.review}
		id, createErr := queries.CreateHybridMarkingAnswerDetection(t.Context(), CreateHybridMarkingAnswerDetectionParams{
			QuestionResultID: questionID, AnswerIndex: int64(index), DetectedState: test.historical,
			MeanGray:        map[bool]float64{true: 100, false: 200}[test.historical == 1],
			HistoricalState: sql.NullInt64{Int64: test.historical, Valid: true}, V2State: sql.NullInt64{Int64: test.v2, Valid: true},
			DarkRatio: sql.NullFloat64{Float64: float64(test.grayscale), Valid: true}, ChromaRatio: sql.NullFloat64{Float64: float64(test.color), Valid: true},
			GrayscaleSignal: sql.NullInt64{Int64: test.grayscale, Valid: true}, ColorSignal: sql.NullInt64{Int64: test.color, Valid: true},
			AutomaticState: test.automatic, ReviewReason: reason,
		})
		if createErr != nil {
			t.Fatalf("case %d: %v", index, createErr)
		}
		if test.review {
			wantCandidates = append(wantCandidates, id)
		}
		if index == 3 {
			colorConfidenceID = id
		}
	}
	if _, err := queries.CreateMarkingAlignedPage(t.Context(), CreateMarkingAlignedPageParams{UserID: 1, CopyResultID: copyID, PageExam: 1, StorageKey: "aligned/student-exam-1000/page-1.png", Width: 100, Height: 100, Sha256: strings.Repeat("c", 64)}); err != nil {
		t.Fatal(err)
	}
	candidates, err := queries.ListPendingMarkingReviewCandidates(t.Context(), ListPendingMarkingReviewCandidatesParams{MarkingJobID: jobID, UserID: 1})
	if err != nil || len(candidates) != len(wantCandidates) {
		t.Fatalf("candidates=%+v err=%v", candidates, err)
	}
	for index, candidate := range candidates {
		if candidate.AnswerDetectionID != wantCandidates[index] {
			t.Fatalf("candidate %d id=%d, want %d", index, candidate.AnswerDetectionID, wantCandidates[index])
		}
	}
	summary, err := queries.GetMarkingReviewSummary(t.Context(), GetMarkingReviewSummaryParams{MarkingJobID: jobID, UserID: 1})
	if err != nil || summary.TotalCandidates != 2 || summary.PendingCandidates != 2 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	effective, err := queries.GetEffectiveAnswerDetection(t.Context(), GetEffectiveAnswerDetectionParams{AnswerDetectionID: colorConfidenceID, UserID: 1})
	if err != nil || effective.DetectedState != 0 || effective.EffectiveState != 1 || effective.ReviewedState.Valid {
		t.Fatalf("color confidence effective state=%+v err=%v", effective, err)
	}
	answers, err := queries.ListEffectiveQuestionAnswersForReview(t.Context(), ListEffectiveQuestionAnswersForReviewParams{
		QuestionResultID: questionID, CopyResultID: copyID, MarkingJobID: jobID, UserID: 1,
	})
	if err != nil || len(answers) != len(tests) || answers[3].DetectedState != 0 || answers[3].EffectiveState != 1 {
		t.Fatalf("effective question answers=%+v err=%v", answers, err)
	}
}

func TestMarkingAlignedPageMetadataIntegrityAndOwnership(t *testing.T) {
	conn := markingReviewTestDB(t)
	defer conn.Close()
	queries := New(conn)
	copyID := createCorrectedCopyResult(t, queries, 100, 1000)
	valid := CreateMarkingAlignedPageParams{
		UserID: 1, CopyResultID: copyID, PageExam: 1,
		StorageKey: "aligned/student-exam-1000/page-1.png",
		Width:      2480, Height: 3508, Sha256: strings.Repeat("a", 64),
	}
	alignedID, err := queries.CreateMarkingAlignedPage(t.Context(), valid)
	if err != nil {
		t.Fatal(err)
	}
	got, err := queries.GetMarkingAlignedPage(t.Context(), GetMarkingAlignedPageParams{CopyResultID: copyID, PageExam: 1, UserID: 1})
	if err != nil || got.ID != alignedID || got.StorageKey != valid.StorageKey || got.Width != 2480 || got.Height != 3508 || got.Sha256 != valid.Sha256 || got.CreatedAt.IsZero() {
		t.Fatalf("aligned page=%+v err=%v", got, err)
	}
	if _, err := queries.GetMarkingAlignedPage(t.Context(), GetMarkingAlignedPageParams{CopyResultID: copyID, PageExam: 1, UserID: 2}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Bob read error=%v", err)
	}
	if _, err := queries.CreateMarkingAlignedPage(t.Context(), valid); err == nil {
		t.Fatal("duplicate copy/page succeeded")
	}
	for _, tc := range []struct {
		name   string
		params CreateMarkingAlignedPageParams
	}{
		{name: "cross user", params: CreateMarkingAlignedPageParams{UserID: 2, CopyResultID: copyID, PageExam: 2, StorageKey: "aligned/student-exam-1000/page-2.png", Width: 1, Height: 1, Sha256: strings.Repeat("b", 64)}},
		{name: "missing copy", params: CreateMarkingAlignedPageParams{UserID: 1, CopyResultID: 9999, PageExam: 1, StorageKey: "aligned/student-exam-1000/page-1.png", Width: 1, Height: 1, Sha256: strings.Repeat("b", 64)}},
		{name: "absent page", params: CreateMarkingAlignedPageParams{UserID: 1, CopyResultID: copyID, PageExam: 3, StorageKey: "aligned/student-exam-1000/page-3.png", Width: 1, Height: 1, Sha256: strings.Repeat("b", 64)}},
		{name: "wrong key", params: CreateMarkingAlignedPageParams{UserID: 1, CopyResultID: copyID, PageExam: 2, StorageKey: "aligned/student-exam-999/page-2.png", Width: 1, Height: 1, Sha256: strings.Repeat("b", 64)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := queries.CreateMarkingAlignedPage(t.Context(), tc.params); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("error=%v, want sql.ErrNoRows", err)
			}
		})
	}
	for name, statement := range map[string]string{
		"traversal":  `INSERT INTO marking_aligned_pages(user_id,copy_result_id,page_exam,storage_key,width,height,sha256) VALUES(1,` + intString(copyID) + `,2,'aligned/../outside.png',1,1,'` + strings.Repeat("b", 64) + `')`,
		"bad hash":   `INSERT INTO marking_aligned_pages(user_id,copy_result_id,page_exam,storage_key,width,height,sha256) VALUES(1,` + intString(copyID) + `,2,'aligned/student-exam-1000/page-2.png',1,1,'ABC')`,
		"zero width": `INSERT INTO marking_aligned_pages(user_id,copy_result_id,page_exam,storage_key,width,height,sha256) VALUES(1,` + intString(copyID) + `,2,'aligned/student-exam-1000/page-2.png',0,1,'` + strings.Repeat("b", 64) + `')`,
		"Bob direct": `INSERT INTO marking_aligned_pages(user_id,copy_result_id,page_exam,storage_key,width,height,sha256) VALUES(2,` + intString(copyID) + `,2,'aligned/student-exam-1000/page-2.png',1,1,'` + strings.Repeat("b", 64) + `')`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := conn.Exec(statement); err == nil {
				t.Fatal("invalid direct aligned page succeeded")
			}
		})
	}
	if _, err := conn.Exec(`UPDATE marking_aligned_pages SET width=1 WHERE id=?`, alignedID); err == nil {
		t.Fatal("aligned page metadata changed")
	}
}

func markingReviewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn := markingResultsTestDB(t)
	if _, err := conn.Exec(`ALTER TABLE marking_jobs ADD COLUMN status TEXT NOT NULL DEFAULT 'running'`); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	if _, err := conn.Exec(`ALTER TABLE marking_jobs ADD COLUMN exam_name TEXT; ALTER TABLE marking_jobs ADD COLUMN mark_table_name TEXT`); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	up38, _ := resultMetadataMigration(t)
	if _, err := conn.Exec(up38); err != nil {
		conn.Close()
		t.Fatalf("result metadata migration Up: %v", err)
	}
	if _, err := conn.Exec(`
		CREATE TABLE student_exam_page_content(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			student_exam_id INTEGER NOT NULL,
			page INTEGER NOT NULL,
			content TEXT NOT NULL,
			user_id INTEGER NOT NULL
		);
		INSERT INTO student_exam_page_content(student_exam_id,page,content,user_id)
		VALUES(1000,1,'{"questions":[{}]}',1),(1000,2,'{"questions":[{}]}',1),(2000,1,'{"questions":[{}]}',2);
	`); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	up, _ := markingReviewMigration(t)
	if _, err := conn.Exec(up); err != nil {
		conn.Close()
		t.Fatalf("migration Up: %v", err)
	}
	return conn
}

func hybridDetectionMigration(t *testing.T) (string, string) {
	t.Helper()
	migration, err := os.ReadFile("../../db/migrations/0042_add_hybrid_answer_detection.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(migration), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("migration has no Down section")
	}
	return strings.Replace(parts[0], "-- +goose Up", "", 1), parts[1]
}

func markingReviewMigration(t *testing.T) (string, string) {
	t.Helper()
	migration, err := os.ReadFile("../../db/migrations/0040_add_marking_review_model.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(migration), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("migration has no Down section")
	}
	return strings.Replace(parts[0], "-- +goose Up", "", 1), parts[1]
}

func intString(value int64) string {
	// Test SQL contains only IDs returned by the local synthetic database.
	return strconv.FormatInt(value, 10)
}
