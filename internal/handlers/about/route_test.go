package about

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestOverviewImageFromRuntimeDirectory(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(root, "internal/static/about/lazymarking-overview-posca.png"))
	if err != nil {
		t.Fatal(err)
	}
	runtime := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "internal"), filepath.Join(runtime, "internal")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(runtime)
	mux := http.NewServeMux()
	RegisterRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/static/about/lazymarking-overview-posca.png", nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" || !bytes.Equal(w.Body.Bytes(), expected) {
		t.Fatalf("runtime asset: status=%d content-type=%q, expected original PNG", w.Code, w.Header().Get("Content-Type"))
	}
}
