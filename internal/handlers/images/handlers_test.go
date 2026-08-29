package images

import (
	"database/sql"
	"errors"
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

func TestEditImageHandlerReturnsNotFoundForMissingImage(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`CREATE TABLE questions(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE images(id INTEGER PRIMARY KEY,question_id INTEGER,image_name TEXT,resize_percentage INTEGER,user_id INTEGER); INSERT INTO questions VALUES(1,1);`); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SESSION_KEY", "image-handler-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"question_id": {"1"}, "width": {"50"}}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
	login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		EditImageHandler(w, r, db.New(conn))
	})).ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", response.Code)
	}
}

func TestDeleteImageHandlerKeepsDatabaseDeletionWhenFilesystemRemovalFails(t *testing.T) {
	t.Chdir(t.TempDir())
	conn := setupDeleteImageHandlerTest(t)
	imagePath := writeMainImageFile(t)

	originalRemove := removeStoredImageFile
	removeStoredImageFile = func(name string) error {
		if name != "main.png" {
			t.Fatalf("remove called with %q, want main.png", name)
		}
		return errors.New("injected remove failure")
	}
	t.Cleanup(func() { removeStoredImageFile = originalRemove })

	response := authenticatedImageRequest(t, url.Values{"question_id": {"42"}}, conn)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusSeeOther)
	}
	assertMainImageRows(t, conn, 0)
	if _, err := os.Stat(imagePath); err != nil {
		t.Fatalf("orphaned file was not preserved: %v", err)
	}
}

func TestDeleteImageHandlerAcceptsAlreadyAbsentFile(t *testing.T) {
	t.Chdir(t.TempDir())
	conn := setupDeleteImageHandlerTest(t)

	response := authenticatedImageRequest(t, url.Values{"question_id": {"42"}}, conn)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusSeeOther)
	}
	assertMainImageRows(t, conn, 0)
}

func TestDeleteImageHandlerDoesNotRemoveFileWhenDatabaseDeleteFails(t *testing.T) {
	t.Chdir(t.TempDir())
	conn := setupDeleteImageHandlerTest(t)
	imagePath := writeMainImageFile(t)
	if _, err := conn.Exec(`CREATE TRIGGER fail_image_delete BEFORE DELETE ON images BEGIN SELECT RAISE(ABORT, 'injected delete failure'); END;`); err != nil {
		t.Fatal(err)
	}

	removeCalled := false
	originalRemove := removeStoredImageFile
	removeStoredImageFile = func(string) error {
		removeCalled = true
		return nil
	}
	t.Cleanup(func() { removeStoredImageFile = originalRemove })

	response := authenticatedImageRequest(t, url.Values{"question_id": {"42"}}, conn)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusInternalServerError)
	}
	if removeCalled {
		t.Fatal("filesystem removal was called after database delete failure")
	}
	assertMainImageRows(t, conn, 1)
	if _, err := os.Stat(imagePath); err != nil {
		t.Fatalf("image file changed after database failure: %v", err)
	}
}

func setupDeleteImageHandlerTest(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := conn.Exec(`
CREATE TABLE questions(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE images(id INTEGER PRIMARY KEY,question_id INTEGER,image_name TEXT,resize_percentage INTEGER,user_id INTEGER);
INSERT INTO questions VALUES(42,1);
INSERT INTO images VALUES(70,42,'main.png',50,1);`); err != nil {
		t.Fatal(err)
	}
	return conn
}

func writeMainImageFile(t *testing.T) string {
	t.Helper()
	if err := os.MkdirAll(config.ImageSavePath, 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(config.ImageSavePath, "main.png")
	if err := os.WriteFile(path, []byte("image"), 0o640); err != nil {
		t.Fatal(err)
	}
	return path
}

func authenticatedImageRequest(t *testing.T, form url.Values, conn *sql.DB) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "image-delete-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
	login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		DeleteImageHandler(w, r, db.New(conn))
	})).ServeHTTP(response, request)
	return response
}

func assertMainImageRows(t *testing.T, conn *sql.DB, want int) {
	t.Helper()
	var got int
	if err := conn.QueryRow("SELECT COUNT(*) FROM images WHERE question_id=42").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("image rows=%d, want %d", got, want)
	}
}
