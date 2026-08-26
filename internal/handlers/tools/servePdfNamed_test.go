package tools

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServePdfNamedServesExistingPDF(t *testing.T) {
	t.Chdir(t.TempDir())
	dir := filepath.Join("assets", "tmp", "alex", "exam-42")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	const content = "%PDF-1.4\n%%EOF\n"
	if err := os.WriteFile(filepath.Join(dir, "result.pdf"), []byte(content), 0o640); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/pdf?file=result.pdf", nil)
	recorder := httptest.NewRecorder()
	ServePdfNamed("alex", "exam-42", "result.pdf", recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %q", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != content {
		t.Fatalf("body = %q, want %q", recorder.Body.String(), content)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/pdf" {
		t.Fatalf("Content-Type = %q, want application/pdf", got)
	}
}

func TestServePdfNamedRejectsSymlinkOutsideWorkspace(t *testing.T) {
	t.Chdir(t.TempDir())
	workspace := createServePDFTestWorkspace(t)
	outsidePath, err := filepath.Abs("outside.pdf")
	if err != nil {
		t.Fatal(err)
	}
	const secret = "%PDF-1.4\noutside secret\n%%EOF\n"
	if err := os.WriteFile(outsidePath, []byte(secret), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, filepath.Join(workspace, "result.pdf")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	recorder := servePDFTestRequest("result.pdf")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if recorder.Body.String() == secret {
		t.Fatal("symlink target content was served")
	}
}

func TestServePdfNamedRejectsSymlinkInsideWorkspace(t *testing.T) {
	t.Chdir(t.TempDir())
	workspace := createServePDFTestWorkspace(t)
	if err := os.WriteFile(filepath.Join(workspace, "target.pdf"), []byte("%PDF target"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.pdf", filepath.Join(workspace, "result.pdf")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	recorder := servePDFTestRequest("result.pdf")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestServePdfNamedRejectsDirectoryWithPDFExtension(t *testing.T) {
	t.Chdir(t.TempDir())
	workspace := createServePDFTestWorkspace(t)
	if err := os.Mkdir(filepath.Join(workspace, "result.pdf"), 0o750); err != nil {
		t.Fatal(err)
	}

	recorder := servePDFTestRequest("result.pdf")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestServePdfNamedReturnsNotFoundForMissingFile(t *testing.T) {
	t.Chdir(t.TempDir())
	createServePDFTestWorkspace(t)

	recorder := servePDFTestRequest("missing.pdf")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func createServePDFTestWorkspace(t *testing.T) string {
	t.Helper()
	workspace, ok := CreateOperationTempDir("alex", "exam-42")
	if !ok {
		t.Fatal("create PDF test workspace")
	}
	return workspace
}

func servePDFTestRequest(filename string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/pdf?file="+filename, nil)
	recorder := httptest.NewRecorder()
	ServePdfNamed("alex", "exam-42", filename, recorder, request)
	return recorder
}
