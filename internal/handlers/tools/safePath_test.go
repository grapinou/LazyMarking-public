package tools

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCreateUserTempDirRejectsTraversal(t *testing.T) {
	if path, err := operationTempDir("../escape", "exam-1"); err == nil || path != "" {
		t.Fatalf("operationTempDir accepted traversal: %q", path)
	}
	if path, err := operationTempDir("alex", "../escape"); err == nil || path != "" {
		t.Fatalf("operationTempDir accepted operation traversal: %q", path)
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

func TestCreateOperationTempDirRejectsSymlinkedParents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires additional privileges on Windows")
	}
	t.Run("user directory", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := os.MkdirAll(filepath.Join("assets", "tmp"), 0o750); err != nil {
			t.Fatal(err)
		}
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join("assets", "tmp", "alex")); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, ok := CreateOperationTempDir("alex", "exam-1"); ok {
			t.Fatal("created workspace through symlinked user directory")
		}
	})
	t.Run("operation directory", func(t *testing.T) {
		t.Chdir(t.TempDir())
		userDir := filepath.Join("assets", "tmp", "alex")
		if err := os.MkdirAll(userDir, 0o750); err != nil {
			t.Fatal(err)
		}
		outside := t.TempDir()
		link := filepath.Join(userDir, "exam-1")
		if err := os.Symlink(outside, link); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, ok := CreateOperationTempDir("alex", "exam-1"); ok {
			t.Fatal("accepted symlinked operation directory")
		}
		if err := RemoveOperationTempDir("alex", "exam-1"); err == nil {
			t.Fatal("removed symlinked operation directory")
		}
		if err := os.WriteFile(filepath.Join(outside, "marker"), []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
		if contents, err := os.ReadFile(filepath.Join(outside, "marker")); err != nil || string(contents) != "keep" {
			t.Fatalf("outside workspace changed: contents=%q err=%v", contents, err)
		}
	})
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
