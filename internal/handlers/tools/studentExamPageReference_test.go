package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func TestResolveStudentExamPageReferenceSuccessAndPageIdentity(t *testing.T) {
	fixture := newPageReferenceResolverFixture(t)
	page1 := fixture.addPNG(t, 1, 3, 2)
	page2 := fixture.addPNG(t, 2, 4, 5)

	for _, tc := range []struct {
		page int64
		want resolverTestPNG
	}{{1, page1}, {2, page2}} {
		got, err := ResolveStudentExamPageReference(context.Background(), fixture.queries, 1, "alice", 100, tc.page)
		if err != nil {
			t.Fatalf("resolve page %d: %v", tc.page, err)
		}
		if got.Page != tc.page || got.GenerationID != 10 || got.StudentExamID != 100 || got.Width != tc.want.width || got.Height != tc.want.height || got.DPI != 300 || got.SHA256 != tc.want.hash || got.Path != tc.want.path {
			t.Fatalf("page %d result=%+v want=%+v", tc.page, got, tc.want)
		}
	}
	if _, err := ResolveStudentExamPageReference(context.Background(), fixture.queries, 2, "alice", 100, 1); !errors.Is(err, ErrStudentExamPageReferenceUnavailable) {
		t.Fatalf("wrong user error=%v", err)
	}
	if _, err := ResolveStudentExamPageReference(context.Background(), fixture.queries, 1, "bob", 100, 1); !errors.Is(err, ErrStudentExamPageReferenceUnavailable) {
		t.Fatalf("wrong username error=%v", err)
	}
	if _, err := ResolveStudentExamPageReference(context.Background(), fixture.queries, 1, "alice", 100, 99); !errors.Is(err, ErrStudentExamPageReferenceUnavailable) {
		t.Fatalf("absent page error=%v", err)
	}
}

func TestStoreStudentExamPageReferencePreservesNativeBytes(t *testing.T) {
	fixture := newPageReferenceResolverFixture(t)
	source := filepath.Join(fixture.workspace, "native-pre-qr.png")
	sourceBytes := fixture.pngBytes(t, 7, 5)
	if err := os.WriteFile(source, sourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	stored, err := StoreStudentExamPageReference(context.Background(), fixture.queries, 1, "alice", 10, 100, 1, source)
	if err != nil {
		t.Fatal(err)
	}
	destinationBytes, err := os.ReadFile(stored.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(sourceBytes, destinationBytes) || stored.SHA256 != sha256Hex(sourceBytes) || sha256Hex(destinationBytes) != stored.SHA256 {
		t.Fatal("stored reference is not byte-for-byte identical to the source")
	}
	if stored.Width != 7 || stored.Height != 5 || stored.DPI != StudentExamPageReferenceDPI {
		t.Fatalf("stored metadata=%+v", stored)
	}
	if err := os.WriteFile(source, []byte("source changed after storage"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveStudentExamPageReference(context.Background(), fixture.queries, 1, "alice", 100, 1)
	if err != nil || resolved.SHA256 != stored.SHA256 {
		t.Fatalf("resolve after source mutation=%+v err=%v", resolved, err)
	}
	if _, err := StoreStudentExamPageReference(context.Background(), fixture.queries, 1, "alice", 10, 100, 1, stored.Path); !errors.Is(err, ErrStudentExamPageReferenceUnsafe) {
		// Durable reference paths are intentionally not accepted as native root
		// sources; idempotence is exercised below with the restored native file.
		t.Fatalf("durable path accepted as source: %v", err)
	}
	if err := os.WriteFile(source, sourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := StoreStudentExamPageReference(context.Background(), fixture.queries, 1, "alice", 10, 100, 1, source); err != nil {
		t.Fatalf("identical idempotent store: %v", err)
	}
}

func TestStoreStudentExamPageReferenceRejectsDifferentDestination(t *testing.T) {
	fixture := newPageReferenceResolverFixture(t)
	source := filepath.Join(fixture.workspace, "native-pre-qr.png")
	if err := os.WriteFile(source, fixture.pngBytes(t, 3, 2), 0o600); err != nil {
		t.Fatal(err)
	}
	destination := fixture.referencePath(1)
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		t.Fatal(err)
	}
	originalDestination := []byte("different historical bytes")
	if err := os.WriteFile(destination, originalDestination, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := StoreStudentExamPageReference(context.Background(), fixture.queries, 1, "alice", 10, 100, 1, source); err == nil {
		t.Fatal("different existing destination was accepted")
	}
	got, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(got, originalDestination) {
		t.Fatalf("existing destination was overwritten: %q err=%v", got, err)
	}
}

func TestValidateExamGenerationReferencesMultiPageAndCorruption(t *testing.T) {
	fixture := newPageReferenceResolverFixture(t)
	for page, size := range map[int64][2]int{1: {3, 2}, 2: {4, 5}} {
		source := filepath.Join(fixture.workspace, "native-"+strconv.FormatInt(page, 10)+".png")
		if err := os.WriteFile(source, fixture.pngBytes(t, size[0], size[1]), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := StoreStudentExamPageReference(context.Background(), fixture.queries, 1, "alice", 10, 100, page, source); err != nil {
			t.Fatalf("store page %d: %v", page, err)
		}
	}
	if err := ValidateExamGenerationReferences(context.Background(), fixture.queries, 1, "alice", 10); err != nil {
		t.Fatalf("complete reference set: %v", err)
	}
	if err := os.WriteFile(fixture.referencePath(2), []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExamGenerationReferences(context.Background(), fixture.queries, 1, "alice", 10); err == nil {
		t.Fatal("corrupt reference set validated")
	}
}

func TestResolveStudentExamPageReferenceRejectsCorruption(t *testing.T) {
	t.Run("hash mismatch", func(t *testing.T) {
		fixture := newPageReferenceResolverFixture(t)
		page := fixture.addPNG(t, 1, 3, 2)
		if err := os.WriteFile(page.path, append(fixture.pngBytes(t, 3, 2), 'x'), 0o600); err != nil {
			t.Fatal(err)
		}
		assertPageReferenceError(t, fixture, ErrStudentExamPageReferenceCorrupt)
	})
	t.Run("dimension mismatch", func(t *testing.T) {
		fixture := newPageReferenceResolverFixture(t)
		fixture.addPNG(t, 1, 3, 2)
		if _, err := fixture.conn.Exec(`UPDATE student_exam_page_content SET reference_width=4 WHERE student_exam_id=100 AND page=1`); err != nil {
			t.Fatal(err)
		}
		assertPageReferenceError(t, fixture, ErrStudentExamPageReferenceCorrupt)
	})
	t.Run("invalid PNG with matching hash", func(t *testing.T) {
		fixture := newPageReferenceResolverFixture(t)
		path := fixture.referencePath(1)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		data := []byte("not a png")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		fixture.setMetadata(t, 1, 3, 2, sha256Hex(data), filepath.ToSlash(filepath.Join("references", "student-exam-100", "page-1.png")))
		assertPageReferenceError(t, fixture, ErrStudentExamPageReferenceCorrupt)
	})
	t.Run("legacy metadata", func(t *testing.T) {
		fixture := newPageReferenceResolverFixture(t)
		assertPageReferenceError(t, fixture, ErrStudentExamPageReferenceUnavailable)
	})
}

func TestResolveStudentExamPageReferenceRejectsUnsafePaths(t *testing.T) {
	for _, key := range []string{"../../outside.png", "/tmp/outside.png", "references/../outside.png", "references/file.jpg", "references/student-exam-100/page-2.png", "references/student-exam-200/page-1.png"} {
		t.Run(key, func(t *testing.T) {
			fixture := newPageReferenceResolverFixture(t)
			data := fixture.pngBytes(t, 3, 2)
			fixture.setMetadata(t, 1, 3, 2, sha256Hex(data), key)
			assertPageReferenceError(t, fixture, ErrStudentExamPageReferenceUnsafe)
		})
	}
	if runtime.GOOS == "windows" {
		return
	}
	t.Run("file symlink", func(t *testing.T) {
		fixture := newPageReferenceResolverFixture(t)
		outside := filepath.Join(t.TempDir(), "outside.png")
		data := fixture.pngBytes(t, 3, 2)
		if err := os.WriteFile(outside, data, 0o600); err != nil {
			t.Fatal(err)
		}
		path := fixture.referencePath(1)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, path); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		fixture.setMetadata(t, 1, 3, 2, sha256Hex(data), "references/student-exam-100/page-1.png")
		assertPageReferenceError(t, fixture, ErrStudentExamPageReferenceUnsafe)
	})
	t.Run("parent symlink", func(t *testing.T) {
		fixture := newPageReferenceResolverFixture(t)
		outside := t.TempDir()
		references := filepath.Join(fixture.workspace, "references")
		if err := os.Symlink(outside, references); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		data := fixture.pngBytes(t, 3, 2)
		fixture.setMetadata(t, 1, 3, 2, sha256Hex(data), "references/student-exam-100/page-1.png")
		assertPageReferenceError(t, fixture, ErrStudentExamPageReferenceUnsafe)
	})
}

type pageReferenceResolverFixture struct {
	conn      *sql.DB
	queries   *db.Queries
	workspace string
}

type resolverTestPNG struct {
	path          string
	width, height int64
	hash          string
}

func newPageReferenceResolverFixture(t *testing.T) pageReferenceResolverFixture {
	t.Helper()
	t.Chdir(t.TempDir())
	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY, username TEXT NOT NULL);
		CREATE TABLE exams_generated(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam(id INTEGER PRIMARY KEY, exam_generated_id INTEGER NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam_page_content(
			id INTEGER PRIMARY KEY, student_exam_id INTEGER NOT NULL, page INTEGER NOT NULL,
			content TEXT NOT NULL, user_id INTEGER NOT NULL,
			reference_storage_key TEXT, reference_width INTEGER, reference_height INTEGER,
			reference_dpi INTEGER, reference_sha256 TEXT);
		INSERT INTO users VALUES(1,'alice'),(2,'bob');
		INSERT INTO exams_generated VALUES(10,1),(20,2);
		INSERT INTO student_exam VALUES(100,10,1),(200,20,2);
		INSERT INTO student_exam_page_content(id,student_exam_id,page,content,user_id) VALUES
			(1,100,1,'{}',1),(2,100,2,'{}',1),(3,200,1,'{}',2);
	`); err != nil {
		t.Fatal(err)
	}
	workspace, ok := CreateOperationTempDir("alice", "exam-10")
	if !ok {
		t.Fatal("create workspace")
	}
	return pageReferenceResolverFixture{conn: conn, queries: db.New(conn), workspace: workspace}
}

func (f pageReferenceResolverFixture) addPNG(t *testing.T, page, width, height int64) resolverTestPNG {
	t.Helper()
	data := f.pngBytes(t, int(width), int(height))
	path := f.referencePath(page)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256Hex(data)
	f.setMetadata(t, page, width, height, hash, filepath.ToSlash(filepath.Join("references", "student-exam-100", "page-"+strconv.FormatInt(page, 10)+".png")))
	return resolverTestPNG{path: path, width: width, height: height, hash: hash}
}

func (f pageReferenceResolverFixture) setMetadata(t *testing.T, page, width, height int64, hash, key string) {
	t.Helper()
	if _, err := f.conn.Exec(`UPDATE student_exam_page_content SET reference_storage_key=?, reference_width=?, reference_height=?, reference_dpi=300, reference_sha256=? WHERE student_exam_id=100 AND page=?`, key, width, height, hash, page); err != nil {
		t.Fatal(err)
	}
}

func (f pageReferenceResolverFixture) referencePath(page int64) string {
	return filepath.Join(f.workspace, "references", "student-exam-100", "page-"+strconv.FormatInt(page, 10)+".png")
}

func (f pageReferenceResolverFixture) pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 20, G: 40, B: 60, A: 255})
	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func assertPageReferenceError(t *testing.T, fixture pageReferenceResolverFixture, want error) {
	t.Helper()
	_, err := ResolveStudentExamPageReference(context.Background(), fixture.queries, 1, "alice", 100, 1)
	if !errors.Is(err, want) {
		t.Fatalf("error=%v, want %v", err, want)
	}
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
