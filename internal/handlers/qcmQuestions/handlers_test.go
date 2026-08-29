package qcmquestions

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/questionfamilies"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestQCMQuestionPagesProvideOwnedQCMContext(t *testing.T) {
	_, queries := newQCMQuestionHandlerTestDB(t)
	tests := []struct {
		name        string
		target      string
		wantContext data.QCMContext
		install     func(func(http.ResponseWriter, data.QCMQuestionPageData)) func()
		serve       http.HandlerFunc
	}{
		{
			name:        "composition",
			target:      "/?qcm_id=1",
			wantContext: data.QCMContext{ID: 1, Name: "owned"},
			install: func(capture func(http.ResponseWriter, data.QCMQuestionPageData)) func() {
				previous := renderTableQCMQuestionPage
				renderTableQCMQuestionPage = capture
				return func() { renderTableQCMQuestionPage = previous }
			},
			serve: func(w http.ResponseWriter, r *http.Request) { TableQCMQuestionsHandler(w, r, queries) },
		},
		{
			name:        "selector",
			target:      "/?qcm_id=3",
			wantContext: data.QCMContext{ID: 3, Name: "owned empty"},
			install: func(capture func(http.ResponseWriter, data.QCMQuestionPageData)) func() {
				previous := renderAddFormQCMQuestionPage
				renderAddFormQCMQuestionPage = capture
				return func() { renderAddFormQCMQuestionPage = previous }
			},
			serve: func(w http.ResponseWriter, r *http.Request) { AddFormQCMQuestionHandler(w, r, queries) },
		},
		{
			name:        "removal confirmation",
			target:      "/?qcm_id=1&qcm_question_id=100",
			wantContext: data.QCMContext{ID: 1, Name: "owned"},
			install: func(capture func(http.ResponseWriter, data.QCMQuestionPageData)) func() {
				previous := renderDeleteFormQCMQuestionPage
				renderDeleteFormQCMQuestionPage = capture
				return func() { renderDeleteFormQCMQuestionPage = previous }
			},
			serve: func(w http.ResponseWriter, r *http.Request) { DeleteFormQCMQuestionHandler(w, r, queries) },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got data.QCMContext
			restore := tc.install(func(_ http.ResponseWriter, page data.QCMQuestionPageData) {
				got = page.QCMContext
			})
			defer restore()
			response := serveAuthenticatedQCMQuestionRequest(t, http.MethodGet, tc.target, nil, tc.serve)
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			if got != tc.wantContext {
				t.Fatalf("QCMContext = %#v, want %#v", got, tc.wantContext)
			}
		})
	}
}

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

func TestAddQCMQuestionsSortsBatchBeforeAssigningPositions(t *testing.T) {
	conn, queries := newQCMQuestionHandlerTestDB(t)
	form := url.Values{"qcm_id": {"3"}, "question_ids": {"12", "10", "11"}}
	response := serveAuthenticatedQCMQuestionRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
		AddQCMQuestionHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", response.Code)
	}
	assertHandlerQCMPositions(t, conn, 3, [][2]int64{{10, 1}, {11, 2}, {12, 3}})
}

func TestDeleteQCMQuestionCompactsPositions(t *testing.T) {
	tests := []struct {
		name     string
		deleteID int64
		want     [][2]int64
	}{
		{name: "last", deleteID: 12, want: [][2]int64{{10, 1}, {11, 2}}},
		{name: "middle", deleteID: 11, want: [][2]int64{{10, 1}, {12, 2}}},
		{name: "first", deleteID: 10, want: [][2]int64{{11, 1}, {12, 2}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, queries := newQCMQuestionHandlerTestDB(t)
			insertHandlerQCMQuestions(t, queries, 3, 10, 11, 12)
			relationID := handlerQCMQuestionRelationID(t, conn, 3, tc.deleteID)
			form := url.Values{"qcm_id": {"3"}, "qcm_question_id": {strconv.FormatInt(relationID, 10)}}
			response := serveAuthenticatedQCMQuestionRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
				DeleteQCMQuestionHandler(w, r, queries, conn)
			})
			if response.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want 303", response.Code)
			}
			assertHandlerQCMPositions(t, conn, 3, tc.want)
		})
	}
}

func TestDeleteQCMQuestionRollsBackWhenCompactionFails(t *testing.T) {
	conn, queries := newQCMQuestionHandlerTestDB(t)
	insertHandlerQCMQuestions(t, queries, 3, 10, 11, 12)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_qcm_position_update BEFORE UPDATE OF position ON qcm_questions
		BEGIN SELECT RAISE(ABORT, 'simulated compaction failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	relationID := handlerQCMQuestionRelationID(t, conn, 3, 11)
	form := url.Values{"qcm_id": {"3"}, "qcm_question_id": {strconv.FormatInt(relationID, 10)}}
	response := serveAuthenticatedQCMQuestionRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
		DeleteQCMQuestionHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	assertHandlerQCMPositions(t, conn, 3, [][2]int64{{10, 1}, {11, 2}, {12, 3}})
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
				DeleteQCMQuestionHandler(w, r, queries, conn)
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

func TestMoveQCMQuestionSwapsAdjacentPositions(t *testing.T) {
	tests := []struct {
		name      string
		question  int64
		direction qcmQuestionMoveDirection
		want      [][2]int64
	}{
		{name: "up from middle", question: 12, direction: moveQCMQuestionUp, want: [][2]int64{{10, 1}, {12, 2}, {11, 3}, {13, 4}}},
		{name: "down from middle", question: 11, direction: moveQCMQuestionDown, want: [][2]int64{{10, 1}, {12, 2}, {11, 3}, {13, 4}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, queries := newQCMQuestionHandlerTestDB(t)
			if _, err := conn.Exec("INSERT INTO questions VALUES (13,3,3,3,3,3,3,'owned fourth question',1)"); err != nil {
				t.Fatal(err)
			}
			insertHandlerQCMQuestions(t, queries, 3, 10, 11, 12, 13)
			before := handlerQCMQuestionIdentity(t, conn, 3)
			relationID := handlerQCMQuestionRelationID(t, conn, 3, tc.question)
			moved, err := moveQCMQuestion(context.Background(), queries, conn, 1, 3, relationID, tc.direction)
			if err != nil {
				t.Fatal(err)
			}
			if !moved {
				t.Fatal("moved = false, want true")
			}
			assertHandlerQCMPositions(t, conn, 3, tc.want)
			after := handlerQCMQuestionIdentity(t, conn, 3)
			if !reflect.DeepEqual(after, before) {
				t.Fatalf("relation identity changed: after=%v before=%v", after, before)
			}
		})
	}
}

func TestMoveQCMQuestionHandlesBoundsAndTwoElements(t *testing.T) {
	tests := []struct {
		name      string
		question  int64
		direction qcmQuestionMoveDirection
		wantMoved bool
		want      [][2]int64
	}{
		{name: "first up is no-op", question: 10, direction: moveQCMQuestionUp, want: [][2]int64{{10, 1}, {11, 2}}},
		{name: "last down is no-op", question: 11, direction: moveQCMQuestionDown, want: [][2]int64{{10, 1}, {11, 2}}},
		{name: "second moves up", question: 11, direction: moveQCMQuestionUp, wantMoved: true, want: [][2]int64{{11, 1}, {10, 2}}},
		{name: "first moves down", question: 10, direction: moveQCMQuestionDown, wantMoved: true, want: [][2]int64{{11, 1}, {10, 2}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, queries := newQCMQuestionHandlerTestDB(t)
			insertHandlerQCMQuestions(t, queries, 3, 10, 11)
			relationID := handlerQCMQuestionRelationID(t, conn, 3, tc.question)
			moved, err := moveQCMQuestion(context.Background(), queries, conn, 1, 3, relationID, tc.direction)
			if err != nil {
				t.Fatal(err)
			}
			if moved != tc.wantMoved {
				t.Fatalf("moved = %t, want %t", moved, tc.wantMoved)
			}
			assertHandlerQCMPositions(t, conn, 3, tc.want)
		})
	}
}

func TestMoveQCMQuestionRollsBackFailedSwap(t *testing.T) {
	conn, queries := newQCMQuestionHandlerTestDB(t)
	insertHandlerQCMQuestions(t, queries, 3, 10, 11, 12)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_adjacent_qcm_move BEFORE UPDATE OF position ON qcm_questions
		WHEN OLD.question_id = 11
		BEGIN SELECT RAISE(ABORT, 'simulated adjacent move failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	relationID := handlerQCMQuestionRelationID(t, conn, 3, 12)
	moved, err := moveQCMQuestion(context.Background(), queries, conn, 1, 3, relationID, moveQCMQuestionUp)
	if err == nil {
		t.Fatal("error = nil, want simulated swap failure")
	}
	if moved {
		t.Fatal("moved = true after failed swap")
	}
	assertHandlerQCMPositions(t, conn, 3, [][2]int64{{10, 1}, {11, 2}, {12, 3}})
}

func TestMoveQCMQuestionHandlersRedirectAfterMoveAndBoundary(t *testing.T) {
	tests := []struct {
		name      string
		question  int64
		direction qcmQuestionMoveDirection
		serve     func(http.ResponseWriter, *http.Request, *db.Queries, *sql.DB)
		want      [][2]int64
	}{
		{name: "move up", question: 12, direction: moveQCMQuestionUp, serve: MoveQCMQuestionUpHandler, want: [][2]int64{{10, 1}, {12, 2}, {11, 3}}},
		{name: "move down", question: 11, direction: moveQCMQuestionDown, serve: MoveQCMQuestionDownHandler, want: [][2]int64{{10, 1}, {12, 2}, {11, 3}}},
		{name: "upper boundary", question: 10, direction: moveQCMQuestionUp, serve: MoveQCMQuestionUpHandler, want: [][2]int64{{10, 1}, {11, 2}, {12, 3}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, queries := newQCMQuestionHandlerTestDB(t)
			insertHandlerQCMQuestions(t, queries, 3, 10, 11, 12)
			relationID := handlerQCMQuestionRelationID(t, conn, 3, tc.question)
			form := url.Values{"qcm_id": {"3"}, "qcm_question_id": {strconv.FormatInt(relationID, 10)}}
			response := serveAuthenticatedQCMQuestionRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
				tc.serve(w, r, queries, conn)
			})
			if response.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want 303", response.Code)
			}
			wantLocation := data.QCMURL(data.DefaultQCMRoutes.AddQuestionURL, 3)
			if location := response.Header().Get("Location"); location != wantLocation {
				t.Fatalf("Location = %q, want %q", location, wantLocation)
			}
			assertHandlerQCMPositions(t, conn, 3, tc.want)
		})
	}
}

func TestMoveQCMQuestionHandlersRejectUnownedOrMismatchedRelations(t *testing.T) {
	conn, queries := newQCMQuestionHandlerTestDB(t)
	insertHandlerQCMQuestions(t, queries, 3, 10, 11)
	ownedRelation := handlerQCMQuestionRelationID(t, conn, 3, 11)
	tests := []struct {
		name string
		form url.Values
	}{
		{name: "missing relation", form: url.Values{"qcm_id": {"3"}, "qcm_question_id": {"999"}}},
		{name: "wrong parent", form: url.Values{"qcm_id": {"1"}, "qcm_question_id": {strconv.FormatInt(ownedRelation, 10)}}},
		{name: "foreign QCM and relation", form: url.Values{"qcm_id": {"2"}, "qcm_question_id": {"200"}}},
		{name: "foreign relation under owned QCM", form: url.Values{"qcm_id": {"1"}, "qcm_question_id": {"200"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedQCMQuestionRequest(t, http.MethodPost, "/", tc.form, func(w http.ResponseWriter, r *http.Request) {
				MoveQCMQuestionUpHandler(w, r, queries, conn)
			})
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
	assertHandlerQCMPositions(t, conn, 3, [][2]int64{{10, 1}, {11, 2}})
	assertHandlerQCMPositions(t, conn, 1, [][2]int64{{10, 1}})
	assertHandlerQCMPositions(t, conn, 2, [][2]int64{{20, 1}})
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

func insertHandlerQCMQuestions(t *testing.T, queries *db.Queries, qcmID int64, questionIDs ...int64) {
	t.Helper()
	for _, questionID := range questionIDs {
		rows, err := queries.CreateQCMQuestion(context.Background(), db.CreateQCMQuestionParams{QcmID: qcmID, QuestionID: questionID, UserID: 1})
		if err != nil || rows != 1 {
			t.Fatalf("insert question %d rows=%d err=%v", questionID, rows, err)
		}
	}
}

func handlerQCMQuestionRelationID(t *testing.T, conn *sql.DB, qcmID, questionID int64) int64 {
	t.Helper()
	var id int64
	if err := conn.QueryRow("SELECT id FROM qcm_questions WHERE qcm_id=? AND question_id=?", qcmID, questionID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func handlerQCMQuestionIdentity(t *testing.T, conn *sql.DB, qcmID int64) [][2]int64 {
	t.Helper()
	rows, err := conn.Query("SELECT id, question_id FROM qcm_questions WHERE qcm_id=? ORDER BY id", qcmID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var identity [][2]int64
	for rows.Next() {
		var item [2]int64
		if err := rows.Scan(&item[0], &item[1]); err != nil {
			t.Fatal(err)
		}
		identity = append(identity, item)
	}
	return identity
}

func assertHandlerQCMPositions(t *testing.T, conn *sql.DB, qcmID int64, want [][2]int64) {
	t.Helper()
	rows, err := conn.Query("SELECT question_id,position FROM qcm_questions WHERE qcm_id=? ORDER BY position", qcmID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got [][2]int64
	for rows.Next() {
		var item [2]int64
		if err := rows.Scan(&item[0], &item[1]); err != nil {
			t.Fatal(err)
		}
		got = append(got, item)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("positions = %v, want %v", got, want)
	}
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
		CREATE TABLE qcm_questions (id INTEGER PRIMARY KEY, qcm_id INTEGER NOT NULL, question_id INTEGER NOT NULL, user_id INTEGER NOT NULL, position INTEGER NOT NULL CHECK(position >= 1), UNIQUE(qcm_id, question_id), UNIQUE(qcm_id, position));
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
		INSERT INTO qcm_questions VALUES (100, 1, 10, 1, 1), (200, 2, 20, 2, 1);
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
