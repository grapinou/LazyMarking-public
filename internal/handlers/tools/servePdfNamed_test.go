package tools

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

func TestServePdfNamedRejectsSymlinkedWorkspaceParents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires additional privileges on Windows")
	}
	for _, component := range []string{"user", "operation"} {
		t.Run(component, func(t *testing.T) {
			t.Chdir(t.TempDir())
			outside := t.TempDir()
			if err := os.WriteFile(filepath.Join(outside, "result.pdf"), []byte("secret"), 0o640); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join("assets", "tmp"), 0o750); err != nil {
				t.Fatal(err)
			}
			if component == "user" {
				if err := os.Symlink(outside, filepath.Join("assets", "tmp", "alex")); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			} else {
				userDir := filepath.Join("assets", "tmp", "alex")
				if err := os.Mkdir(userDir, 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(userDir, "exam-42")); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			}
			recorder := servePDFTestRequest("result.pdf")
			if recorder.Code != http.StatusNotFound || recorder.Body.String() == "secret" {
				t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestServePdfNamedRejectsOperationSymlinkToAnotherUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires additional privileges on Windows")
	}
	t.Chdir(t.TempDir())
	bobWorkspace := filepath.Join("assets", "tmp", "bob", "exam-42")
	aliceDir := filepath.Join("assets", "tmp", "alice")
	if err := os.MkdirAll(bobWorkspace, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(aliceDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bobWorkspace, "result.pdf"), []byte("bob secret"), 0o640); err != nil {
		t.Fatal(err)
	}
	target, err := filepath.Abs(bobWorkspace)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(aliceDir, "exam-42")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/pdf", nil)
	recorder := httptest.NewRecorder()
	ServePdfNamed("alice", "exam-42", "result.pdf", recorder, request)
	if recorder.Code != http.StatusNotFound || recorder.Body.String() == "bob secret" {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
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
