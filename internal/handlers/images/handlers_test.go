package images

import (
	"bytes"
	"database/sql"
	"errors"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
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

func TestAddImageHandlerPrevalidatesOwnedQuestion(t *testing.T) {
	tests := []struct {
		name       string
		questionID string
	}{
		{name: "missing question", questionID: "999"},
		{name: "foreign question", questionID: "99"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			conn := setupAddImageHandlerTest(t)
			circleChecks := 0
			originalCheck := checkUploadedImageCircles
			checkUploadedImageCircles = func(string, string, float64) bool {
				circleChecks++
				return true
			}
			t.Cleanup(func() { checkUploadedImageCircles = originalCheck })

			response := authenticatedAddImageRequest(t, conn, test.questionID, "upload.png", "25")
			if response.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusNotFound)
			}
			if circleChecks != 0 {
				t.Fatalf("OpenCV checks=%d, want 0", circleChecks)
			}
			assertImageTableCount(t, conn, 0)
			assertNoPermanentImages(t)
		})
	}
}

func TestAddImageHandlerCreatesOwnedImageAndFile(t *testing.T) {
	t.Chdir(t.TempDir())
	conn := setupAddImageHandlerTest(t)
	originalCheck := checkUploadedImageCircles
	checkUploadedImageCircles = func(string, string, float64) bool { return true }
	t.Cleanup(func() { checkUploadedImageCircles = originalCheck })

	response := authenticatedAddImageRequest(t, conn, "42", "upload.png", "25")
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusSeeOther)
	}
	var questionID, width int64
	var filename string
	if err := conn.QueryRow("SELECT question_id,image_name,resize_percentage FROM images").Scan(&questionID, &filename, &width); err != nil {
		t.Fatal(err)
	}
	if questionID != 42 || width != 25 || filename != "1_test-user_mainQuestion_42_upload.png" {
		t.Fatalf("stored image=(question=%d name=%q width=%d)", questionID, filename, width)
	}
	if _, err := os.Stat(filepath.Join(config.ImageSavePath, filename)); err != nil {
		t.Fatalf("stored file missing: %v", err)
	}
}

func TestAddImageHandlerCleansFileWhenInsertFails(t *testing.T) {
	t.Chdir(t.TempDir())
	conn := setupAddImageHandlerTest(t)
	if _, err := conn.Exec("INSERT INTO images VALUES(70,42,'existing.png',50,1)"); err != nil {
		t.Fatal(err)
	}
	originalCheck := checkUploadedImageCircles
	checkUploadedImageCircles = func(string, string, float64) bool { return true }
	t.Cleanup(func() { checkUploadedImageCircles = originalCheck })

	response := authenticatedAddImageRequest(t, conn, "42", "second.png", "25")
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusSeeOther)
	}
	assertImageTableCount(t, conn, 1)
	newPath := filepath.Join(config.ImageSavePath, "1_test-user_mainQuestion_42_second.png")
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatalf("file from rejected INSERT was not cleaned up: %v", err)
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

func setupAddImageHandlerTest(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := conn.Exec(`
CREATE TABLE subjects(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE themes(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE year_levels(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE skills(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE difficulties(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE points(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE questions(id INTEGER PRIMARY KEY,subject_id INTEGER,theme_id INTEGER,year_level_id INTEGER,skill_id INTEGER,difficulty_id INTEGER,point_id INTEGER,content TEXT,user_id INTEGER);
CREATE TABLE images(id INTEGER PRIMARY KEY,question_id INTEGER UNIQUE,image_name TEXT,resize_percentage INTEGER,user_id INTEGER);
INSERT INTO subjects VALUES(1,1),(2,2); INSERT INTO themes VALUES(1,1),(2,2); INSERT INTO year_levels VALUES(1,1),(2,2);
INSERT INTO skills VALUES(1,1),(2,2); INSERT INTO difficulties VALUES(1,1),(2,2); INSERT INTO points VALUES(1,1),(2,2);
INSERT INTO questions VALUES(42,1,1,1,1,1,1,'owned',1),(99,2,2,2,2,2,2,'foreign',2);`); err != nil {
		t.Fatal(err)
	}
	return conn
}

func authenticatedAddImageRequest(t *testing.T, conn *sql.DB, questionID, filename, width string) *httptest.ResponseRecorder {
	t.Helper()
	request := imageUploadRequest(t, filename, map[string]string{
		"question_id": questionID,
		"width":       width,
	})
	t.Setenv("SESSION_KEY", "image-add-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
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
	login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddImageHandler(w, r, db.New(conn))
	})).ServeHTTP(response, request)
	return response
}

func imageUploadRequest(t *testing.T, filename string, fields map[string]string) *http.Request {
	t.Helper()
	var encoded bytes.Buffer
	pixel := image.NewRGBA(image.Rect(0, 0, 30, 20))
	pixel.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&encoded, pixel); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(encoded.Bytes()); err != nil {
		t.Fatal(err)
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/images/add", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func assertImageTableCount(t *testing.T, conn *sql.DB, want int) {
	t.Helper()
	var got int
	if err := conn.QueryRow("SELECT COUNT(*) FROM images").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("image rows=%d, want %d", got, want)
	}
}

func assertNoPermanentImages(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(config.ImageSavePath)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("permanent image entries=%d, want 0", len(entries))
	}
}
