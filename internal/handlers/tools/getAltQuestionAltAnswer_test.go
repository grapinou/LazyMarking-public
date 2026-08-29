package tools

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestAltQuestionBuildersIncludeOwnedVariantImage(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
CREATE TABLE questions(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE alt_questions(id INTEGER PRIMARY KEY,question_id INTEGER,content TEXT,user_id INTEGER);
CREATE TABLE alt_images(id INTEGER PRIMARY KEY,alt_question_id INTEGER,image_name TEXT,resize_percentage INTEGER,user_id INTEGER);
CREATE TABLE alt_answers(id INTEGER PRIMARY KEY,alt_question_id INTEGER,state INTEGER,content TEXT,user_id INTEGER);
INSERT INTO questions VALUES(42,1);
INSERT INTO alt_questions VALUES(7,42,'illustrated variant',1);
INSERT INTO alt_images VALUES(70,7,'variant-7.png',65,1);`); err != nil {
		t.Fatal(err)
	}
	queries := db.New(conn)

	t.Run("preview request helper", func(t *testing.T) {
		request := httptest.NewRequest("GET", "/", nil)
		question, err := GetAltQuestionAltAnswer(1, 7, queries, request)
		if err != nil {
			t.Fatal(err)
		}
		if question.Image.Name != "variant-7.png" || question.Image.Width != "65" {
			t.Fatalf("image=%#v, want variant-7.png at 65%%", question.Image)
		}
	})

	t.Run("generation context helper", func(t *testing.T) {
		question, err := GetAltQuestionAltAnswerCtx(1, 7, queries, context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if question.Image.Name != "variant-7.png" || question.Image.Width != "65" {
			t.Fatalf("image=%#v, want variant-7.png at 65%%", question.Image)
		}
	})
}
