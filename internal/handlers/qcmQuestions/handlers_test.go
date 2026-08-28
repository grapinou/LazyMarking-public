package qcmquestions

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/questionfamilies"
)

func TestQCMQuestionManagementPagesRequireOwnedQCM(t *testing.T) {
	conn, queries := newQCMQuestionHandlerTestDB(t)
	_ = conn
	tests := []struct {
		name   string
		target string
		serve  http.HandlerFunc
	}{
		{name: "table missing", target: "/?qcm_id=999", serve: func(w http.ResponseWriter, r *http.Request) { TableQCMQuestionsHandler(w, r, queries) }},
		{name: "table foreign", target: "/?qcm_id=2", serve: func(w http.ResponseWriter, r *http.Request) { TableQCMQuestionsHandler(w, r, queries) }},
		{name: "add form missing", target: "/?qcm_id=999", serve: func(w http.ResponseWriter, r *http.Request) { AddFormQCMQuestionHandler(w, r, queries) }},
		{name: "add form foreign", target: "/?qcm_id=2", serve: func(w http.ResponseWriter, r *http.Request) { AddFormQCMQuestionHandler(w, r, queries) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedQCMQuestionRequest(t, http.MethodGet, tc.target, nil, tc.serve)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
}

func TestAddQCMQuestionsRollsBackMixedOwnershipSelection(t *testing.T) {
	conn, queries := newQCMQuestionHandlerTestDB(t)
	form := url.Values{"qcm_id": {"3"}, "question_ids": {"10", "20"}}
	response := serveAuthenticatedQCMQuestionRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
		AddQCMQuestionHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM qcm_questions WHERE qcm_id = 3").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partially inserted QCM relations = %d, want 0", count)
	}
}

func TestDeleteQCMQuestionDoesNotReportMissingOrForeignAsSuccess(t *testing.T) {
	conn, queries := newQCMQuestionHandlerTestDB(t)
	tests := []struct {
		name string
		form url.Values
	}{
		{name: "missing relation", form: url.Values{"qcm_id": {"1"}, "qcm_question_id": {"999"}}},
		{name: "foreign relation", form: url.Values{"qcm_id": {"2"}, "qcm_question_id": {"200"}}},
		{name: "mismatched parent", form: url.Values{"qcm_id": {"3"}, "qcm_question_id": {"100"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedQCMQuestionRequest(t, http.MethodPost, "/", tc.form, func(w http.ResponseWriter, r *http.Request) {
				DeleteQCMQuestionHandler(w, r, queries)
			})
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM qcm_questions WHERE id = 200 AND user_id = 2").Scan(&count); err != nil || count != 1 {
		t.Fatalf("foreign relation count = %d, err = %v; want 1", count, err)
	}
}

func TestLoadQCMQuestionFamiliesEnforcesCandidatesOwnershipAndFamilySelection(t *testing.T) {
	_, queries := newQCMQuestionHandlerTestDB(t)
	families, err := loadQCMQuestionFamilies(context.Background(), queries, 1, db.GetFilteredQuestionsParams{UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(families) != 2 {
		t.Fatalf("families count = %d, want 2 candidates", len(families))
	}
	for _, family := range families {
		if family.Main.ID == 10 {
			t.Fatal("family already in QCM was exposed")
		}
		if family.Main.ID == 20 {
			t.Fatal("foreign main question was exposed")
		}
		if !family.Main.Selectable {
			t.Fatalf("main question %d is not selectable", family.Main.ID)
		}
		for _, variant := range family.Variants {
			if variant.Content == "foreign variant" || variant.Content == "inconsistent parent variant" {
				t.Fatalf("foreign variant exposed: %#v", variant)
			}
		}
	}
	if families[0].Main.ID != 11 || len(families[0].Variants) != 2 {
		t.Fatalf("family with variants = %#v, want main 11 with two owned variants", families[0])
	}
	if families[1].Main.ID != 12 || families[1].Variants == nil || len(families[1].Variants) != 0 {
		t.Fatalf("empty family = %#v, want main 12 with non-nil empty variants", families[1])
	}
}

func TestLoadQCMQuestionFamiliesFiltersOnMainMetadataAndKeepsAllVariants(t *testing.T) {
	_, queries := newQCMQuestionHandlerTestDB(t)
	tests := []struct {
		name  string
		apply func(*db.GetFilteredQuestionsParams)
	}{
		{"subject", func(p *db.GetFilteredQuestionsParams) { p.SubjectID = sql.NullInt64{Int64: 1, Valid: true} }},
		{"theme", func(p *db.GetFilteredQuestionsParams) { p.ThemeID = sql.NullInt64{Int64: 1, Valid: true} }},
		{"year level", func(p *db.GetFilteredQuestionsParams) { p.YearLevelID = sql.NullInt64{Int64: 1, Valid: true} }},
		{"skill", func(p *db.GetFilteredQuestionsParams) { p.SkillID = sql.NullInt64{Int64: 1, Valid: true} }},
		{"difficulty", func(p *db.GetFilteredQuestionsParams) { p.DifficultyID = sql.NullInt64{Int64: 1, Valid: true} }},
		{"point", func(p *db.GetFilteredQuestionsParams) { p.PointID = sql.NullInt64{Int64: 1, Valid: true} }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filters := db.GetFilteredQuestionsParams{UserID: 1}
			tc.apply(&filters)
			families, err := loadQCMQuestionFamilies(context.Background(), queries, 3, filters)
			if err != nil {
				t.Fatal(err)
			}
			if len(families) != 2 || families[0].Main.ID != 10 || families[1].Main.ID != 11 {
				t.Fatalf("filtered main IDs = %v, want [10 11]", mainQuestionIDs(families))
			}
			if len(families[1].Variants) != 2 {
				t.Fatalf("matching family variants = %#v, want all two variants", families[1].Variants)
			}
		})
	}
}

func mainQuestionIDs(families []questionfamilies.QuestionFamily) []int64 {
	ids := make([]int64, 0, len(families))
	for _, family := range families {
		ids = append(ids, family.Main.ID)
	}
	return ids
}

func newQCMQuestionHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE qcm (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE subjects(id INTEGER PRIMARY KEY,name TEXT,user_id INTEGER); CREATE TABLE themes(id INTEGER PRIMARY KEY,name TEXT,user_id INTEGER);
		CREATE TABLE year_levels(id INTEGER PRIMARY KEY,name TEXT,user_id INTEGER); CREATE TABLE skills(id INTEGER PRIMARY KEY,name TEXT,user_id INTEGER);
		CREATE TABLE difficulties(id INTEGER PRIMARY KEY,name TEXT,user_id INTEGER); CREATE TABLE points(id INTEGER PRIMARY KEY,point_value INTEGER,user_id INTEGER);
		CREATE TABLE questions (id INTEGER PRIMARY KEY, subject_id INTEGER, theme_id INTEGER, year_level_id INTEGER, skill_id INTEGER, difficulty_id INTEGER, point_id INTEGER, content TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE alt_questions(id INTEGER PRIMARY KEY,question_id INTEGER,content TEXT,user_id INTEGER);
		CREATE TABLE qcm_questions (id INTEGER PRIMARY KEY, qcm_id INTEGER NOT NULL, question_id INTEGER NOT NULL, user_id INTEGER NOT NULL, UNIQUE(qcm_id, question_id));
		INSERT INTO qcm VALUES (1, 'owned', 1), (2, 'foreign', 2), (3, 'owned empty', 1);
		INSERT INTO subjects VALUES(1,'subject one',1),(2,'foreign subject',2),(3,'subject three',1);
		INSERT INTO themes VALUES(1,'theme one',1),(2,'foreign theme',2),(3,'theme three',1);
		INSERT INTO year_levels VALUES(1,'year one',1),(2,'foreign year',2),(3,'year three',1);
		INSERT INTO skills VALUES(1,'skill one',1),(2,'foreign skill',2),(3,'skill three',1);
		INSERT INTO difficulties VALUES(1,'difficulty one',1),(2,'foreign difficulty',2),(3,'difficulty three',1);
		INSERT INTO points VALUES(1,1,1),(2,2,2),(3,3,1);
		INSERT INTO questions VALUES
			(10,1,1,1,1,1,1,'owned selected',1),
			(11,1,1,1,1,1,1,'owned candidate with variants',1),
			(12,3,3,3,3,3,3,'owned other metadata',1),
			(20,2,2,2,2,2,2,'foreign content',2);
		INSERT INTO alt_questions VALUES
			(110,11,'owned variant A',1),(111,11,'owned variant B',1),
			(112,11,'foreign variant',2),(200,20,'inconsistent parent variant',1);
		INSERT INTO qcm_questions VALUES (100, 1, 10, 1), (200, 2, 20, 2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func serveAuthenticatedQCMQuestionRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "qcm-question-handler-test-key-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
	if form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	session, err := login.GetStore().Get(request, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	session.Values["username"] = "test-user"
	cookies := httptest.NewRecorder()
	if err := session.Save(request, cookies); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range cookies.Result().Cookies() {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	login.CheckAuth(handler).ServeHTTP(response, request)
	return response
}
