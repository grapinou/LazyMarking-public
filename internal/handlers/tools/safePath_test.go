package tools

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateUserTempDirRejectsTraversal(t *testing.T) {
	if path, ok := CreateUserTempDir("../escape"); ok || path != "" {
		t.Fatalf("CreateUserTempDir accepted traversal: %q", path)
	}
}

func TestServePdfNamedRejectsTraversal(t *testing.T) {
	for _, test := range []struct{ username, filename string }{
		{"../other-user", "exam.pdf"},
		{"alice", "../exam.pdf"},
		{"alice", "not-a-pdf.txt"},
	} {
		recorder := httptest.NewRecorder()
		ServePdfNamed(test.username, test.filename, recorder)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("ServePdfNamed(%q, %q) status = %d, want 400", test.username, test.filename, recorder.Code)
		}
	}
}

func TestSaveUploadedFileRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	file := multipart.File(&memoryFile{Reader: bytes.NewReader([]byte("content"))})
	if err := SaveUploadedFile(file, dir, "../escape"); err == nil {
		t.Fatal("SaveUploadedFile accepted traversal")
	}
	if _, err := os.Stat(filepath.Join(dir, "..", "escape")); !os.IsNotExist(err) {
		t.Fatalf("unexpected escaped file: %v", err)
	}
}

type memoryFile struct{ *bytes.Reader }

func (m *memoryFile) Close() error { return nil }
