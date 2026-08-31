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
	if _, err := conn.Exec(`
		CREATE TABLE student_exam_page_content(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			student_exam_id INTEGER NOT NULL,
			page INTEGER NOT NULL,
			content TEXT NOT NULL,
			user_id INTEGER NOT NULL
		);
		INSERT INTO student_exam_page_content(student_exam_id,page,content,user_id)
		VALUES(1000,1,'{}',1),(1000,2,'{}',1),(2000,1,'{}',2);
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
