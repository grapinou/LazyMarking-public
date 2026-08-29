package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func setupExamIntegrityTest(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	_, err = conn.Exec(`
CREATE TABLE qcm(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
CREATE TABLE class_codes(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
CREATE TABLE periods(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
CREATE TABLE years(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
CREATE TABLE exams(
 id INTEGER PRIMARY KEY, name TEXT NOT NULL CHECK(length(trim(name)) > 0),
 qcm_id INTEGER NOT NULL REFERENCES qcm(id) ON DELETE RESTRICT,
 class_code_id INTEGER NOT NULL REFERENCES class_codes(id) ON DELETE RESTRICT,
 period_id INTEGER NOT NULL REFERENCES periods(id) ON DELETE RESTRICT,
 year_id INTEGER NOT NULL REFERENCES years(id) ON DELETE RESTRICT,
 user_id INTEGER NOT NULL, UNIQUE(name,qcm_id,class_code_id,user_id));
CREATE TABLE exams_generated(id INTEGER PRIMARY KEY, exam_id INTEGER NOT NULL REFERENCES exams(id) ON DELETE CASCADE, user_id INTEGER NOT NULL);
INSERT INTO qcm VALUES (1,'q1',1),(2,'q2',2);
INSERT INTO class_codes VALUES (1,'c1',1),(2,'c2',2);
INSERT INTO periods VALUES (1,'p1',1),(2,'p2',2);
INSERT INTO years VALUES (1,'y1',1),(2,'y2',2);
INSERT INTO exams VALUES (1,'owned',1,1,1,1,1),(2,'foreign',2,2,2,2,2),(3,'referenced',1,1,1,1,1);
INSERT INTO exams_generated VALUES (1,3,1),(2,2,2);`)
	if err != nil {
		t.Fatal(err)
	}
	return conn, New(conn)
}

func TestCreateExamRequiresEveryParentToBeOwned(t *testing.T) {
	_, queries := setupExamIntegrityTest(t)
	ctx := context.Background()
	base := CreateExamParams{Name: "new", QcmID: 1, ClassCodeID: 1, PeriodID: 1, YearID: 1, UserID: 1}
	rows, err := queries.CreateExam(ctx, base)
	if err != nil || rows != 1 {
		t.Fatalf("owned create: rows=%d err=%v", rows, err)
	}

	tests := []struct {
		name string
		edit func(*CreateExamParams)
	}{
		{"foreign QCM", func(p *CreateExamParams) { p.QcmID = 2 }},
		{"foreign class", func(p *CreateExamParams) { p.ClassCodeID = 2 }},
		{"foreign period", func(p *CreateExamParams) { p.PeriodID = 2 }},
		{"foreign year", func(p *CreateExamParams) { p.YearID = 2 }},
		{"forged user", func(p *CreateExamParams) { p.UserID = 2; p.QcmID, p.ClassCodeID, p.PeriodID, p.YearID = 1, 1, 1, 1 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := base
			params.Name = tc.name
			tc.edit(&params)
			rows, err := queries.CreateExam(ctx, params)
			if err != nil || rows != 0 {
				t.Fatalf("rows=%d err=%v, want zero rows and no constraint error", rows, err)
			}
		})
	}
	if rows, err := queries.CreateExam(ctx, base); err == nil || rows != 0 {
		t.Fatalf("duplicate create: rows=%d err=%v, want constraint error", rows, err)
	}
}

func TestUpdateExamRequiresOwnedTargetAndParents(t *testing.T) {
	conn, queries := setupExamIntegrityTest(t)
	ctx := context.Background()
	base := UpdateExamParams{Name: "changed", QcmID: 1, ClassCodeID: 1, PeriodID: 1, YearID: 1, ID: 1, UserID: 1}
	rows, err := queries.UpdateExam(ctx, base)
	if err != nil || rows != 1 {
		t.Fatalf("owned update: rows=%d err=%v", rows, err)
	}

	tests := []struct {
		name string
		edit func(*UpdateExamParams)
	}{
		{"missing exam", func(p *UpdateExamParams) { p.ID = 999 }},
		{"foreign exam", func(p *UpdateExamParams) { p.ID = 2 }},
		{"foreign QCM", func(p *UpdateExamParams) { p.QcmID = 2 }},
		{"foreign class", func(p *UpdateExamParams) { p.ClassCodeID = 2 }},
		{"foreign period", func(p *UpdateExamParams) { p.PeriodID = 2 }},
		{"foreign year", func(p *UpdateExamParams) { p.YearID = 2 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := base
			params.Name = "rejected"
			tc.edit(&params)
			rows, err := queries.UpdateExam(ctx, params)
			if err != nil || rows != 0 {
				t.Fatalf("rows=%d err=%v", rows, err)
			}
			var name string
			if err := conn.QueryRow("SELECT name FROM exams WHERE id=1").Scan(&name); err != nil || name != "changed" {
				t.Fatalf("owned exam changed after rejection: name=%q err=%v", name, err)
			}
		})
	}
}

func TestUpdateExamIsAtomicallyBlockedWhileGenerationExists(t *testing.T) {
	conn, queries := setupExamIntegrityTest(t)
	ctx := context.Background()
	if _, err := conn.Exec(`
		INSERT INTO qcm VALUES(4,'q4',1);
		INSERT INTO class_codes VALUES(4,'c4',1);
		INSERT INTO periods VALUES(4,'p4',1);
		INSERT INTO years VALUES(4,'y4',1);
	`); err != nil {
		t.Fatal(err)
	}

	base := UpdateExamParams{Name: "referenced", QcmID: 1, ClassCodeID: 1, PeriodID: 1, YearID: 1, ID: 3, UserID: 1}
	tests := []struct {
		name string
		edit func(*UpdateExamParams)
	}{
		{"name", func(p *UpdateExamParams) { p.Name = "must-not-change" }},
		{"QCM", func(p *UpdateExamParams) { p.QcmID = 4 }},
		{"class", func(p *UpdateExamParams) { p.ClassCodeID = 4 }},
		{"period", func(p *UpdateExamParams) { p.PeriodID = 4 }},
		{"year", func(p *UpdateExamParams) { p.YearID = 4 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := base
			tc.edit(&params)
			rows, err := queries.UpdateExam(ctx, params)
			if err != nil || rows != 0 {
				t.Fatalf("generated Exam update: rows=%d err=%v, want zero rows", rows, err)
			}
		})
	}
	var name string
	var qcmID, classID, periodID, yearID int64
	if err := conn.QueryRow("SELECT name,qcm_id,class_code_id,period_id,year_id FROM exams WHERE id=3").Scan(&name, &qcmID, &classID, &periodID, &yearID); err != nil || name != "referenced" || qcmID != 1 || classID != 1 || periodID != 1 || yearID != 1 {
		t.Fatalf("generated Exam changed: (%q,%d,%d,%d,%d) err=%v", name, qcmID, classID, periodID, yearID, err)
	}

	if _, err := conn.Exec("DELETE FROM exams_generated WHERE exam_id=3 AND user_id=1"); err != nil {
		t.Fatal(err)
	}
	params := base
	params.Name, params.QcmID, params.ClassCodeID, params.PeriodID, params.YearID = "allowed", 4, 4, 4, 4
	rows, err := queries.UpdateExam(ctx, params)
	if err != nil || rows != 1 {
		t.Fatalf("Exam update after generation cleanup: rows=%d err=%v, want one row", rows, err)
	}
}

func TestDeleteExamOwnershipAndExistingCascadeSemantics(t *testing.T) {
	conn, queries := setupExamIntegrityTest(t)
	ctx := context.Background()
	for name, params := range map[string]DeleteExamParams{
		"missing": {ID: 999, UserID: 1},
		"foreign": {ID: 2, UserID: 1},
	} {
		t.Run(name, func(t *testing.T) {
			rows, err := queries.DeleteExam(ctx, params)
			if err != nil || rows != 0 {
				t.Fatalf("rows=%d err=%v", rows, err)
			}
		})
	}
	if rows, err := queries.DeleteExam(ctx, DeleteExamParams{ID: 1, UserID: 1}); err != nil || rows != 1 {
		t.Fatalf("unused delete: rows=%d err=%v", rows, err)
	}
	if rows, err := queries.DeleteExam(ctx, DeleteExamParams{ID: 3, UserID: 1}); err != nil || rows != 1 {
		t.Fatalf("referenced delete under existing cascade: rows=%d err=%v", rows, err)
	}
	var count int
	if err := conn.QueryRow("SELECT count(*) FROM exams_generated WHERE exam_id=3").Scan(&count); err != nil || count != 0 {
		t.Fatalf("dependent generation rows after cascade=%d err=%v", count, err)
	}
	if err := conn.QueryRow("SELECT count(*) FROM exams WHERE id=2 AND user_id=2").Scan(&count); err != nil || count != 1 {
		t.Fatalf("foreign exam count=%d err=%v", count, err)
	}
}

func TestExamReadsDoNotExposeForeignExam(t *testing.T) {
	conn, queries := setupExamIntegrityTest(t)
	_, err := queries.GetExamByID(context.Background(), GetExamByIDParams{ID: 2, UserID: 1})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("foreign GetExamByID error=%v, want sql.ErrNoRows", err)
	}
	// Simulate a legacy/inconsistent row to verify reads do not rely solely on
	// migration triggers having always been present.
	if _, err := conn.Exec("INSERT INTO exams VALUES (4,'inconsistent',2,1,1,1,1)"); err != nil {
		t.Fatal(err)
	}
	_, err = queries.GetExamByID(context.Background(), GetExamByIDParams{ID: 4, UserID: 1})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("inconsistent GetExamByID error=%v, want sql.ErrNoRows", err)
	}
	exams, err := queries.GetExamsAllInfos(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, exam := range exams {
		if exam.ID == 2 || exam.ID == 4 {
			t.Fatalf("foreign parent data exposed through exam %d", exam.ID)
		}
	}
	_, err = queries.GetExamNameAndClassCodeName(context.Background(), GetExamNameAndClassCodeNameParams{ID: 4, UserID: 1})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("inconsistent GetExamNameAndClassCodeName error=%v, want sql.ErrNoRows", err)
	}
}

func TestGeneratedExamNameLookupUsesGenerationID(t *testing.T) {
	_, queries := setupExamIntegrityTest(t)
	ctx := context.Background()

	names, err := queries.GetExamNameAndClassCodeName(ctx, GetExamNameAndClassCodeNameParams{ID: 1, UserID: 1})
	if err != nil {
		t.Fatalf("owned generated exam lookup: %v", err)
	}
	if names.ExamName != "referenced" || names.ClassName != "c1" {
		t.Fatalf("names=%+v, want referenced exam and c1", names)
	}

	_, err = queries.GetExamNameAndClassCodeName(ctx, GetExamNameAndClassCodeNameParams{ID: 2, UserID: 1})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("foreign generated exam error=%v, want sql.ErrNoRows", err)
	}
}
