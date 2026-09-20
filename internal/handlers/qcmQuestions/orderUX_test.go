package qcmquestions

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestReorderLongComposition(t *testing.T) {
	t.Chdir("../../..")
	for _, count := range []int{10, 20, 30} {
		page := data.QCMQuestionPageData{ReorderURL: "/?reorder=1", CompositionURL: "/", QCMContext: data.QCMContext{ID: 3}}
		for i := 0; i < count; i++ {
			page.QCMQuestions = append(page.QCMQuestions, data.QCMQuestionItem{
				QCMQuestionID: int64(i + 1), Position: int64(i + 1), Content: strings.Repeat("Énoncé long ", 50),
				IsFirst: i == 0, IsLast: i == count-1, MoveUpURL: "/move-up", MoveDownURL: "/move-down",
			})
		}
		for _, mode := range []bool{false, true} {
			page.Reordering = mode
			rec := httptest.NewRecorder()
			RenderTableQCMQuestionPage(rec, page)
			body := rec.Body.String()
			wantForms := 0
			if mode {
				wantForms = 2 * count
			}
			if rec.Code != 200 || strings.Count(body, `<article class="card shadow-sm">`) != count || strings.Count(body, `name="qcm_question_id"`) != wantForms {
				t.Fatalf("count=%d mode=%t: missing questions or unexpected controls", count, mode)
			}
		}
	}
}

func TestReorderModeAndPersistence(t *testing.T) {
	t.Chdir("../../..")
	conn, queries := newQCMQuestionHandlerTestDB(t)
	insertHandlerQCMQuestions(t, queries, 3, 10, 11, 12)
	base := data.QCMURL(data.DefaultQCMRoutes.AddQuestionURL, 3)
	get := func(path string) string {
		t.Helper()
		response := serveAuthenticatedQCMQuestionRequest(t, http.MethodGet, path, nil, func(w http.ResponseWriter, r *http.Request) { TableQCMQuestionsHandler(w, r, queries) })
		if response.Code != 200 {
			t.Fatalf("GET %s: %d %s", path, response.Code, response.Body.String())
		}
		return response.Body.String()
	}
	normal := get(base)
	if !strings.Contains(normal, ">Réorganiser</a>") || strings.Contains(normal, `name="qcm_question_id"`) {
		t.Fatal("normal composition exposes move forms or misses mode link")
	}
	assertHandlerQCMPositions(t, conn, 3, [][2]int64{{10, 1}, {11, 2}, {12, 3}})
	mode := get(base + "&reorder=1")
	if !strings.Contains(mode, "Terminer la réorganisation") || strings.Count(mode, `name="reorder" value="1"`) != 6 || strings.Count(mode, " disabled") != 2 {
		t.Fatal("reorder mode, boundaries or form flags incorrect")
	}
	for _, path := range []string{data.QCMURL(data.DefaultQCMRoutes.AddQuestionURL, 1), data.QCMURL(data.DefaultQCMRoutes.AddQuestionURL, 1) + "&reorder=1"} {
		single := get(path)
		if strings.Contains(single, ">Réorganiser</a>") || strings.Contains(single, `name="qcm_question_id"`) {
			t.Fatal("single question exposes reorder controls")
		}
	}
	relation := handlerQCMQuestionRelationID(t, conn, 3, 12)
	form := url.Values{"qcm_id": {"3"}, "qcm_question_id": {strconv.FormatInt(relation, 10)}, "reorder": {"1"}}
	for _, direction := range []string{"up", "down"} {
		response := serveAuthenticatedQCMQuestionRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
			if direction == "up" {
				MoveQCMQuestionUpHandler(w, r, queries, conn)
			} else {
				MoveQCMQuestionDownHandler(w, r, queries, conn)
			}
		})
		if response.Code != 303 || response.Header().Get("Location") != base+"&reorder=1" {
			t.Fatalf("mode lost after move: %v", response)
		}
		body := get(response.Header().Get("Location"))
		a, b, c := strings.Index(body, "owned selected"), strings.Index(body, "owned candidate with variants"), strings.Index(body, "owned other metadata")
		if direction == "up" {
			assertHandlerQCMPositions(t, conn, 3, [][2]int64{{10, 1}, {12, 2}, {11, 3}})
			if !(a < c && c < b) {
				t.Fatal("reload did not show A C B")
			}
		} else {
			assertHandlerQCMPositions(t, conn, 3, [][2]int64{{10, 1}, {11, 2}, {12, 3}})
			if !(a < b && b < c) {
				t.Fatal("reload did not show A B C")
			}
		}
	}
	if strings.Contains(get(base), `name="qcm_question_id"`) {
		t.Fatal("leaving mode still exposes move forms")
	}
}
