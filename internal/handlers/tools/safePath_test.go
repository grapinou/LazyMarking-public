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
	if path, err := operationTempDir("../escape", "exam-1"); err == nil || path != "" {
		t.Fatalf("operationTempDir accepted traversal: %q", path)
	}
}

func TestSafePathComponentAcceptsOrdinaryUsernames(t *testing.T) {
	for _, username := range []string{"alex", "prof2026"} {
		if err := safePathComponent(username); err != nil {
			t.Errorf("safePathComponent(%q) returned %v", username, err)
		}
	}
}

func TestOperationWorkspacesAreDistinct(t *testing.T) {
	first, err := operationTempDir("alex", "exam-101")
	if err != nil {
		t.Fatal("failed to create first workspace")
	}
	second, err := operationTempDir("alex", "exam-102")
	if err != nil {
		t.Fatal("failed to create second workspace")
	}
	if first == second {
		t.Fatalf("operation workspaces collide: %q", first)
	}
}

func TestServePdfNamedRejectsTraversal(t *testing.T) {
	for _, test := range []struct{ username, filename string }{
		{"../other-user", "exam.pdf"},
		{"alice", "../exam.pdf"},
		{"alice", "not-a-pdf.txt"},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/pdf", nil)
		ServePdfNamed(test.username, "exam-1", test.filename, recorder, request)
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
