package questions

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestQuestionInstructionCreateEditClearAndOwnership(t *testing.T) {
	conn := setupQuestionMutationHandlerTest(t)
	queries := db.New(conn)
	form := url.Values{"content": {"Énoncé distinct"}, "instruction": {"  Choisissez toutes les réponses « exactes ».\nJustifiez mentalement.  "}, "subjectID": {"1"}, "themeID": {"1"}, "yearLevelID": {"1"}, "skillID": {"1"}, "difficultyID": {"1"}, "pointID": {"1"}}
	res := authenticatedQuestionRequest(t, http.MethodPost, "/questions/add", form, func(w http.ResponseWriter, r *http.Request) { AddQuestionsHandler(w, r, queries) })
	if res.Code != http.StatusSeeOther {
		t.Fatalf("create: %d", res.Code)
	}
	assertStored := func(want string) {
		t.Helper()
		var instruction, content string
		if err := conn.QueryRow("SELECT instruction,content FROM questions WHERE id=1").Scan(&instruction, &content); err != nil {
			t.Fatal(err)
		}
		if instruction != want || content != "Énoncé distinct" {
			t.Fatalf("stored instruction=%q content=%q", instruction, content)
		}
	}
	assertStored(strings.TrimSpace(form.Get("instruction")))
	form.Set("question_id", "1")
	for _, instruction := range []string{`Consigne modifiée "sans injection" #panic("x")`, ""} {
		form.Set("instruction", instruction)
		res = authenticatedQuestionRequest(t, http.MethodPost, "/questions/edit", form, func(w http.ResponseWriter, r *http.Request) { EditQuestionHandler(w, r, queries) })
		if res.Code != http.StatusSeeOther {
			t.Fatalf("edit: %d", res.Code)
		}
		assertStored(instruction)
	}
	// Change the owner out of band: the same authenticated POST must now be refused.
	if _, err := conn.Exec("UPDATE questions SET user_id=2 WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	form.Set("instruction", "Vol de consigne")
	res = authenticatedQuestionRequest(t, http.MethodPost, "/questions/edit", form, func(w http.ResponseWriter, r *http.Request) { EditQuestionHandler(w, r, queries) })
	if res.Code != http.StatusNotFound {
		t.Fatalf("foreign edit: %d", res.Code)
	}
	assertStored("")
	// Creation with a foreign classification must not create a question or instruction.
	form.Set("subjectID", "999")
	res = authenticatedQuestionRequest(t, http.MethodPost, "/questions/add", form, func(w http.ResponseWriter, r *http.Request) { AddQuestionsHandler(w, r, queries) })
	if res.Code != http.StatusNotFound {
		t.Fatalf("foreign create: %d", res.Code)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM questions").Scan(&count); err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
