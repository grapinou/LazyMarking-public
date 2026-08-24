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
