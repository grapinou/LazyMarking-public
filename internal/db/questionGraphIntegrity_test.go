package db

import (
	"context"
	"database/sql"
	"testing"
)

func setupQuestionGraphTest(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	_, err = conn.Exec(`
CREATE TABLE subjects(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE themes(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE year_levels(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE skills(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE difficulties(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE points(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE questions(id INTEGER PRIMARY KEY,subject_id INTEGER,theme_id INTEGER,year_level_id INTEGER,skill_id INTEGER,difficulty_id INTEGER,point_id INTEGER,content TEXT NOT NULL,user_id INTEGER,UNIQUE(content,user_id));
CREATE TABLE answers(id INTEGER PRIMARY KEY,question_id INTEGER,state INTEGER,content TEXT,user_id INTEGER);
CREATE TABLE images(id INTEGER PRIMARY KEY,question_id INTEGER UNIQUE,image_name TEXT,resize_percentage INTEGER,user_id INTEGER);
CREATE TABLE alt_questions(id INTEGER PRIMARY KEY,question_id INTEGER,content TEXT,user_id INTEGER,UNIQUE(content,user_id));
CREATE TABLE alt_answers(id INTEGER PRIMARY KEY,alt_question_id INTEGER,state INTEGER,content TEXT,user_id INTEGER);
CREATE TABLE alt_images(id INTEGER PRIMARY KEY,alt_question_id INTEGER UNIQUE,image_name TEXT,resize_percentage INTEGER,user_id INTEGER);
INSERT INTO subjects VALUES(1,1),(2,2); INSERT INTO themes VALUES(1,1),(2,2);
INSERT INTO year_levels VALUES(1,1),(2,2); INSERT INTO skills VALUES(1,1),(2,2);
INSERT INTO difficulties VALUES(1,1),(2,2); INSERT INTO points VALUES(1,1),(2,2);
INSERT INTO questions VALUES(10,1,1,1,1,1,1,'owned',1),(11,1,1,1,1,1,1,'owned two',1),(20,2,2,2,2,2,2,'foreign',2);
INSERT INTO answers VALUES(100,10,1,'answer',1),(110,11,1,'other answer',1),(200,20,1,'foreign answer',2);
INSERT INTO alt_questions VALUES(30,10,'alternative',1),(31,11,'other alternative',1),(40,20,'foreign alternative',2);
INSERT INTO alt_answers VALUES(300,30,1,'alt answer',1),(310,31,1,'other alt answer',1),(400,40,1,'foreign alt answer',2);
INSERT INTO images VALUES(500,10,'owned.png',50,1),(600,20,'foreign.png',50,2);
INSERT INTO alt_images VALUES(700,30,'owned-alt.png',50,1),(800,40,'foreign-alt.png',50,2);`)
	if err != nil {
		t.Fatal(err)
	}
	return conn, New(conn)
}

func ownedQuestionParams(name string) CreateQuestionParams {
	return CreateQuestionParams{SubjectID: 1, ThemeID: 1, YearLevelID: 1, SkillID: 1, DifficultyID: 1, PointID: 1, Content: name, UserID: 1}
}

func TestQuestionMutationsRequireOwnedFeatures(t *testing.T) {
	conn, queries := setupQuestionGraphTest(t)
	ctx := context.Background()
	if rows, err := queries.CreateQuestion(ctx, ownedQuestionParams("created")); err != nil || rows != 1 {
		t.Fatalf("owned create rows=%d err=%v", rows, err)
	}
	for _, tc := range []struct {
		name string
		edit func(*CreateQuestionParams)
	}{
		{"subject", func(p *CreateQuestionParams) { p.SubjectID = 2 }}, {"theme", func(p *CreateQuestionParams) { p.ThemeID = 2 }},
		{"year level", func(p *CreateQuestionParams) { p.YearLevelID = 2 }}, {"skill", func(p *CreateQuestionParams) { p.SkillID = 2 }},
		{"difficulty", func(p *CreateQuestionParams) { p.DifficultyID = 2 }}, {"point", func(p *CreateQuestionParams) { p.PointID = 2 }},
	} {
		t.Run("foreign "+tc.name, func(t *testing.T) {
			p := ownedQuestionParams("bad " + tc.name)
			tc.edit(&p)
			if rows, err := queries.CreateQuestion(ctx, p); err != nil || rows != 0 {
				t.Fatalf("rows=%d err=%v", rows, err)
			}
		})
	}
	base := UpdateQuestionParams{SubjectID: 1, ThemeID: 1, YearLevelID: 1, SkillID: 1, DifficultyID: 1, PointID: 1, Content: "changed", ID: 10, UserID: 1}
	if rows, err := queries.UpdateQuestion(ctx, base); err != nil || rows != 1 {
		t.Fatalf("owned update rows=%d err=%v", rows, err)
	}
	for name, edit := range map[string]func(*UpdateQuestionParams){
		"missing": func(p *UpdateQuestionParams) { p.ID = 999 }, "foreign target": func(p *UpdateQuestionParams) { p.ID = 20 },
		"foreign feature": func(p *UpdateQuestionParams) { p.SubjectID = 2 },
	} {
		t.Run(name, func(t *testing.T) {
			p := base
			p.Content = "rejected"
			edit(&p)
			if rows, err := queries.UpdateQuestion(ctx, p); err != nil || rows != 0 {
				t.Fatalf("rows=%d err=%v", rows, err)
			}
			var content string
			if err := conn.QueryRow("SELECT content FROM questions WHERE id=10").Scan(&content); err != nil || content != "changed" {
				t.Fatalf("content=%q err=%v", content, err)
			}
		})
	}
}

func TestQuestionReadsHideLegacyInconsistentRows(t *testing.T) {
	conn, queries := setupQuestionGraphTest(t)
	if _, err := conn.Exec("INSERT INTO questions VALUES(12,2,1,1,1,1,1,'inconsistent',1)"); err != nil {
		t.Fatal(err)
	}
	if _, err := queries.GetQuestionByID(context.Background(), GetQuestionByIDParams{ID: 12, UserID: 1}); err != sql.ErrNoRows {
		t.Fatalf("GetQuestionByID error=%v, want sql.ErrNoRows", err)
	}
	rows, err := queries.GetAllQuestions(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.ID == 12 {
			t.Fatal("inconsistent question exposed by GetAllQuestions")
		}
	}
}

func TestAnswerAndAlternativeGraphParentBinding(t *testing.T) {
	conn, queries := setupQuestionGraphTest(t)
	ctx := context.Background()
	if rows, err := queries.CreateAnswer(ctx, CreateAnswerParams{QuestionID: 10, State: 0, Content: "new", UserID: 1}); err != nil || rows != 1 {
		t.Fatalf("answer create rows=%d err=%v", rows, err)
	}
	if rows, err := queries.CreateAnswer(ctx, CreateAnswerParams{QuestionID: 20, State: 0, Content: "bad", UserID: 1}); err != nil || rows != 0 {
		t.Fatalf("foreign answer create rows=%d err=%v", rows, err)
	}
	if rows, err := queries.UpdateAnswer(ctx, UpdateAnswerParams{State: 0, Content: "bad", ID: 100, QuestionID: 11, UserID: 1}); err != nil || rows != 0 {
		t.Fatalf("mismatched answer update rows=%d err=%v", rows, err)
	}
	if rows, err := queries.DeleteAnswer(ctx, DeleteAnswerParams{ID: 200, QuestionID: 20, UserID: 1}); err != nil || rows != 0 {
		t.Fatalf("foreign answer delete rows=%d err=%v", rows, err)
	}
	if rows, err := queries.CreateAltQuestion(ctx, CreateAltQuestionParams{QuestionID: 10, Content: "new alt", UserID: 1}); err != nil || rows != 1 {
		t.Fatalf("alt create rows=%d err=%v", rows, err)
	}
	if rows, err := queries.CreateAltQuestion(ctx, CreateAltQuestionParams{QuestionID: 20, Content: "bad alt", UserID: 1}); err != nil || rows != 0 {
		t.Fatalf("foreign alt create rows=%d err=%v", rows, err)
	}
	if rows, err := queries.UpdateAltQuestion(ctx, UpdateAltQuestionParams{Content: "bad", ID: 30, QuestionID: 11, UserID: 1}); err != nil || rows != 0 {
		t.Fatalf("mismatched alt update rows=%d err=%v", rows, err)
	}
	if rows, err := queries.UpdateAltAnswer(ctx, UpdateAltAnswerParams{State: 0, Content: "bad", ID: 300, AltQuestionID: 31, UserID: 1, QuestionID: 11}); err != nil || rows != 0 {
		t.Fatalf("mismatched alt answer update rows=%d err=%v", rows, err)
	}
	var content string
	if err := conn.QueryRow("SELECT content FROM answers WHERE id=100").Scan(&content); err != nil || content != "answer" {
		t.Fatalf("answer changed: %q err=%v", content, err)
	}
	if err := conn.QueryRow("SELECT content FROM alt_answers WHERE id=300").Scan(&content); err != nil || content != "alt answer" {
		t.Fatalf("alt answer changed: %q err=%v", content, err)
	}
}

func TestImageMutationsRequireOwnedParentsAndReportRows(t *testing.T) {
	conn, queries := setupQuestionGraphTest(t)
	ctx := context.Background()
	if rows, err := queries.CreateImage(ctx, CreateImageParams{QuestionID: 11, ImageName: "new.png", ResizePercentage: 50, UserID: 1}); err != nil || rows != 1 {
		t.Fatalf("image create rows=%d err=%v", rows, err)
	}
	if rows, err := queries.CreateImage(ctx, CreateImageParams{QuestionID: 20, ImageName: "bad.png", ResizePercentage: 50, UserID: 1}); err != nil || rows != 0 {
		t.Fatalf("foreign image create rows=%d err=%v", rows, err)
	}
	if rows, err := queries.UpdateSizeImage(ctx, UpdateSizeImageParams{ResizePercentage: 60, QuestionID: 20, UserID: 1}); err != nil || rows != 0 {
		t.Fatalf("foreign resize rows=%d err=%v", rows, err)
	}
	if rows, err := queries.DeleteImage(ctx, DeleteImageParams{QuestionID: 999, UserID: 1}); err != nil || rows != 0 {
		t.Fatalf("missing delete rows=%d err=%v", rows, err)
	}
	if rows, err := queries.UpdateSizeAltImage(ctx, UpdateSizeAltImageParams{ResizePercentage: 60, AltQuestionID: 30, UserID: 1, QuestionID: 11}); err != nil || rows != 0 {
		t.Fatalf("mismatched alt resize rows=%d err=%v", rows, err)
	}
	var size int
	if err := conn.QueryRow("SELECT resize_percentage FROM images WHERE id=600").Scan(&size); err != nil || size != 50 {
		t.Fatalf("foreign image size=%d err=%v", size, err)
	}
}
