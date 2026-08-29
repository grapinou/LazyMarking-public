package altimages

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
)

func TestTableAltImageHandlerUsesQuestionParentToReadExistingImage(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	conn := setupAltImageHandlerTest(t)

	response := authenticatedAltImageRequest(t, http.MethodGet,
		"/dashboard/questions/altquestions/altimages?question_id=42&alt_question_id=7", nil,
		func(w http.ResponseWriter, r *http.Request) { TableAltImageHandler(w, r, db.New(conn)) })

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	if !strings.Contains(body, "variant-7.png") {
		t.Fatalf("existing variant image was not rendered: %q", body)
	}
	if strings.Contains(body, "Pas d'image alternative") {
		t.Fatal("existing variant image was incorrectly treated as absent")
	}
}

func TestTableAltImageHandlerRejectsMismatchedAndForeignVariants(t *testing.T) {
	conn := setupAltImageHandlerTest(t)
	for _, target := range []string{
		"/?question_id=42&alt_question_id=8",
		"/?question_id=99&alt_question_id=9",
	} {
		response := authenticatedAltImageRequest(t, http.MethodGet, target, nil,
			func(w http.ResponseWriter, r *http.Request) { TableAltImageHandler(w, r, db.New(conn)) })
		if response.Code != http.StatusNotFound {
			t.Fatalf("target=%q status=%d, want %d", target, response.Code, http.StatusNotFound)
		}
	}
}

func TestDeleteFormAltImageHandlerRejectsMismatchedParent(t *testing.T) {
	conn := setupAltImageHandlerTest(t)
	response := authenticatedAltImageRequest(t, http.MethodGet,
		"/?question_id=42&alt_question_id=8", nil,
		func(w http.ResponseWriter, r *http.Request) { DeleteFormAltImageHandler(w, r, db.New(conn)) })
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestDeleteAltImageHandlerRemovesOwnedVariantImageAndFile(t *testing.T) {
	t.Chdir(t.TempDir())
	conn := setupAltImageHandlerTest(t)
	if err := os.MkdirAll(config.ImageSavePath, 0o750); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(config.ImageSavePath, "variant-7.png")
	if err := os.WriteFile(imagePath, []byte("image"), 0o640); err != nil {
		t.Fatal(err)
	}

	response := authenticatedAltImageRequest(t, http.MethodPost, "/delete", url.Values{
		"question_id":     {"42"},
		"alt_question_id": {"7"},
	}, func(w http.ResponseWriter, r *http.Request) { DeleteAltImageHandler(w, r, db.New(conn)) })

	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusSeeOther)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM alt_images WHERE alt_question_id=7").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("remaining image rows=%d, want 0", count)
	}
	if _, err := os.Stat(imagePath); !os.IsNotExist(err) {
		t.Fatalf("stored image was not removed: %v", err)
	}
}

func TestDeleteAltImageHandlerReturnsNotFoundWithoutMutation(t *testing.T) {
	tests := []struct {
		name          string
		questionID    string
		altQuestionID string
	}{
		{name: "missing image", questionID: "42", altQuestionID: "999"},
		{name: "variant bound to another question", questionID: "43", altQuestionID: "7"},
		{name: "foreign variant image", questionID: "99", altQuestionID: "9"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			conn := setupAltImageHandlerTest(t)
			if err := os.MkdirAll(config.ImageSavePath, 0o750); err != nil {
				t.Fatal(err)
			}
			imageNames := []string{"variant-7.png", "variant-8.png", "variant-9.png"}
			for _, name := range imageNames {
				if err := os.WriteFile(filepath.Join(config.ImageSavePath, name), []byte(name), 0o640); err != nil {
					t.Fatal(err)
				}
			}

			response := authenticatedAltImageRequest(t, http.MethodPost, "/delete", url.Values{
				"question_id":     {test.questionID},
				"alt_question_id": {test.altQuestionID},
			}, func(w http.ResponseWriter, r *http.Request) {
				DeleteAltImageHandler(w, r, db.New(conn))
			})

			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusNotFound)
			}
			var count int
			if err := conn.QueryRow("SELECT COUNT(*) FROM alt_images").Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 3 {
				t.Fatalf("remaining image rows=%d, want 3", count)
			}
			for _, name := range imageNames {
				if _, err := os.Stat(filepath.Join(config.ImageSavePath, name)); err != nil {
					t.Fatalf("stored image %q changed: %v", name, err)
				}
			}
		})
	}
}

func TestDeleteAltImageHandlerKeepsDatabaseErrorsAsInternalServerError(t *testing.T) {
	conn := setupAltImageHandlerTest(t)
	if _, err := conn.Exec("DROP TABLE alt_images"); err != nil {
		t.Fatal(err)
	}

	response := authenticatedAltImageRequest(t, http.MethodPost, "/delete", url.Values{
		"question_id":     {"42"},
		"alt_question_id": {"7"},
	}, func(w http.ResponseWriter, r *http.Request) {
		DeleteAltImageHandler(w, r, db.New(conn))
	})

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func setupAltImageHandlerTest(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if _, err := conn.Exec(`
CREATE TABLE subjects(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE themes(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE year_levels(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE skills(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE difficulties(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE points(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE questions(id INTEGER PRIMARY KEY,subject_id INTEGER,theme_id INTEGER,year_level_id INTEGER,skill_id INTEGER,difficulty_id INTEGER,point_id INTEGER,content TEXT,user_id INTEGER);
CREATE TABLE alt_questions(id INTEGER PRIMARY KEY,question_id INTEGER,content TEXT,user_id INTEGER);
CREATE TABLE alt_images(id INTEGER PRIMARY KEY,alt_question_id INTEGER UNIQUE,image_name TEXT,resize_percentage INTEGER,user_id INTEGER);
INSERT INTO subjects VALUES(1,1),(2,2); INSERT INTO themes VALUES(1,1),(2,2); INSERT INTO year_levels VALUES(1,1),(2,2);
INSERT INTO skills VALUES(1,1),(2,2); INSERT INTO difficulties VALUES(1,1),(2,2); INSERT INTO points VALUES(1,1),(2,2);
INSERT INTO questions VALUES(42,1,1,1,1,1,1,'owned question',1),(43,1,1,1,1,1,1,'other owned question',1),(99,2,2,2,2,2,2,'foreign question',2);
INSERT INTO alt_questions VALUES(7,42,'owned variant',1),(8,43,'other parent variant',1),(9,99,'foreign variant',2);
INSERT INTO alt_images VALUES(70,7,'variant-7.png',50,1),(80,8,'variant-8.png',50,1),(90,9,'variant-9.png',50,2);`); err != nil {
		t.Fatal(err)
	}
	return conn
}

func authenticatedAltImageRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "alt-image-handler-test-key-32-bytes")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	var body string
	if form != nil {
		body = form.Encode()
	}
	request := httptest.NewRequest(method, target, strings.NewReader(body))
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
