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
	"reflect"
	"strconv"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestStoreAndResolveMarkingAlignedPagePreservesBytesAndDetections(t *testing.T) {
	fixture := newMarkingAlignedPageFixture(t)
	sourceBytes := alignedTestPNG(t, 80, 60, false)
	sourcePath := filepath.Join(fixture.workspace, "homography.png")
	if err := os.WriteFile(sourcePath, sourceBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	answers := []config.CircleValidated{{Position: config.Position{X: 20, Y: 20}, Radius: 12}, {Position: config.Position{X: 55, Y: 35}, Radius: 12}}
	before, err := GetAnswerDetections(fixture.workspace, filepath.Base(sourcePath), answers)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := StageMarkingAlignedPage(fixture.workspace, 100, 1, filepath.Base(sourcePath))
	if err != nil {
		t.Fatal(err)
	}

	// DrawMarking rewrites the working homography after staging. Simulating
	// that mutation proves the durable page comes from the pre-annotation bytes.
	if err := os.WriteFile(sourcePath, alignedTestPNG(t, 80, 60, true), 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := StoreMarkingAlignedPage(context.Background(), fixture.queries, 1, "alice", 50, 500, 100, 1, staged)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(staged); err != nil {
		t.Fatal(err)
	}
	durableBytes, err := os.ReadFile(resolved.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(durableBytes, sourceBytes) {
		t.Fatal("durable aligned page bytes differ from the pre-annotation homography")
	}
	digest := sha256.Sum256(sourceBytes)
	if resolved.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("resolved hash=%q", resolved.SHA256)
	}
	after, err := GetAnswerDetections(filepath.Dir(resolved.Path), filepath.Base(resolved.Path), answers)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("detections before=%+v after=%+v", before, after)
	}
	var storedHash string
	if err := fixture.conn.QueryRow(`SELECT sha256 FROM marking_aligned_pages WHERE copy_result_id=500 AND page_exam=1`).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if storedHash != resolved.SHA256 {
		t.Fatalf("DB hash=%q resolved=%q", storedHash, resolved.SHA256)
	}
}

func TestMarkingAlignedPagesAreDistinctAcrossPagesAndCopies(t *testing.T) {
	fixture := newMarkingAlignedPageFixture(t)
	if _, err := fixture.conn.Exec(`INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,expected_pages,detected_pages,completed_at) VALUES(501,1,50,101,'corrected',1,1,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ copyID, studentID, page int64 }{{500, 100, 1}, {500, 100, 2}, {501, 101, 1}} {
		name := "source-" + alignedInt(tc.copyID) + "-" + alignedInt(tc.page) + ".png"
		if err := os.WriteFile(filepath.Join(fixture.workspace, name), alignedTestPNG(t, int(30+tc.page), 30, false), 0o600); err != nil {
			t.Fatal(err)
		}
		staged, err := StageMarkingAlignedPage(fixture.workspace, tc.studentID, int(tc.page), name)
		if err != nil {
			t.Fatal(err)
		}
		resolved, err := StoreMarkingAlignedPage(t.Context(), fixture.queries, 1, "alice", 50, tc.copyID, tc.studentID, tc.page, staged)
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join("aligned", "student-exam-"+alignedInt(tc.studentID), "page-"+alignedInt(tc.page)+".png")
		if rel, err := filepath.Rel(fixture.workspace, resolved.Path); err != nil || rel != want {
			t.Fatalf("path=%q rel=%q err=%v want=%q", resolved.Path, rel, err, want)
		}
	}
}

func TestResolveMarkingAlignedPageRejectsCorruptionAndWrongScope(t *testing.T) {
	fixture := newMarkingAlignedPageFixture(t)
	data := alignedTestPNG(t, 40, 40, false)
	source := filepath.Join(fixture.workspace, "source.png")
	if err := os.WriteFile(source, data, 0o600); err != nil {
		t.Fatal(err)
	}
	staged, err := StageMarkingAlignedPage(fixture.workspace, 100, 1, "source.png")
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := StoreMarkingAlignedPage(t.Context(), fixture.queries, 1, "alice", 50, 500, 100, 1, staged)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveMarkingAlignedPage(t.Context(), fixture.queries, 2, "bob", 50, 500, 100, 1); !errors.Is(err, ErrMarkingAlignedPageUnavailable) {
		t.Fatalf("wrong user: %v", err)
	}
	if _, err := ResolveMarkingAlignedPage(t.Context(), fixture.queries, 1, "alice", 51, 500, 100, 1); !errors.Is(err, ErrMarkingAlignedPageUnavailable) {
		t.Fatalf("wrong job: %v", err)
	}
	if err := os.WriteFile(resolved.Path, alignedTestPNG(t, 40, 40, true), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveMarkingAlignedPage(t.Context(), fixture.queries, 1, "alice", 50, 500, 100, 1); !errors.Is(err, ErrMarkingAlignedPageCorrupt) {
		t.Fatalf("corruption: %v", err)
	}
}

func TestStoreMarkingAlignedPageRejectsUnsafeOrWrongScope(t *testing.T) {
	t.Run("wrong identities", func(t *testing.T) {
		fixture := newMarkingAlignedPageFixture(t)
		source := filepath.Join(fixture.workspace, "source.png")
		if err := os.WriteFile(source, alignedTestPNG(t, 20, 20, false), 0o600); err != nil {
			t.Fatal(err)
		}
		staged, err := StageMarkingAlignedPage(fixture.workspace, 100, 1, "source.png")
		if err != nil {
			t.Fatal(err)
		}
		for name, args := range map[string][4]int64{
			"wrong job":     {51, 500, 100, 1},
			"wrong copy":    {50, 999, 100, 1},
			"wrong student": {50, 500, 101, 1},
			"wrong page":    {50, 500, 100, 3},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := StoreMarkingAlignedPage(t.Context(), fixture.queries, 1, "alice", args[0], args[1], args[2], args[3], staged)
				if !errors.Is(err, ErrMarkingAlignedPageUnavailable) {
					t.Fatalf("error=%v", err)
				}
			})
		}
	})

	t.Run("symlink source", func(t *testing.T) {
		fixture := newMarkingAlignedPageFixture(t)
		target := filepath.Join(fixture.workspace, "target.png")
		if err := os.WriteFile(target, alignedTestPNG(t, 20, 20, false), 0o600); err != nil {
			t.Fatal(err)
		}
		stagingDir := filepath.Join(fixture.workspace, ".aligned-staging", "student-exam-100")
		if err := os.MkdirAll(stagingDir, 0o750); err != nil {
			t.Fatal(err)
		}
		symlink := filepath.Join(stagingDir, "page-1.png")
		if err := os.Symlink(target, symlink); err != nil {
			t.Fatal(err)
		}
		if _, err := StoreMarkingAlignedPage(t.Context(), fixture.queries, 1, "alice", 50, 500, 100, 1, symlink); !errors.Is(err, ErrMarkingAlignedPageUnsafe) {
			t.Fatalf("error=%v", err)
		}
	})

	t.Run("symlink destination parent", func(t *testing.T) {
		fixture := newMarkingAlignedPageFixture(t)
		source := filepath.Join(fixture.workspace, "source.png")
		if err := os.WriteFile(source, alignedTestPNG(t, 20, 20, false), 0o600); err != nil {
			t.Fatal(err)
		}
		staged, err := StageMarkingAlignedPage(fixture.workspace, 100, 1, "source.png")
		if err != nil {
			t.Fatal(err)
		}
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(fixture.workspace, "aligned")); err != nil {
			t.Fatal(err)
		}
		if _, err := StoreMarkingAlignedPage(t.Context(), fixture.queries, 1, "alice", 50, 500, 100, 1, staged); !errors.Is(err, ErrMarkingAlignedPageUnsafe) {
			t.Fatalf("error=%v", err)
		}
	})

	t.Run("corrupt PNG", func(t *testing.T) {
		fixture := newMarkingAlignedPageFixture(t)
		source := filepath.Join(fixture.workspace, "bad.png")
		if err := os.WriteFile(source, []byte("not png"), 0o600); err != nil {
			t.Fatal(err)
		}
		staged, err := StageMarkingAlignedPage(fixture.workspace, 100, 1, "bad.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := StoreMarkingAlignedPage(t.Context(), fixture.queries, 1, "alice", 50, 500, 100, 1, staged); !errors.Is(err, ErrMarkingAlignedPageCorrupt) {
			t.Fatalf("error=%v", err)
		}
	})
}

func TestResolveMarkingAlignedPageRejectsMetadataMismatchAndTraversal(t *testing.T) {
	for _, tc := range []struct {
		name   string
		update string
		want   error
	}{
		{name: "dimensions", update: `UPDATE marking_aligned_pages SET width=width+1`, want: ErrMarkingAlignedPageCorrupt},
		{name: "traversal", update: `UPDATE marking_aligned_pages SET storage_key='../outside.png'`, want: ErrMarkingAlignedPageUnsafe},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newMarkingAlignedPageFixture(t)
			source := filepath.Join(fixture.workspace, "source.png")
			if err := os.WriteFile(source, alignedTestPNG(t, 20, 20, false), 0o600); err != nil {
				t.Fatal(err)
			}
			staged, err := StageMarkingAlignedPage(fixture.workspace, 100, 1, "source.png")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := StoreMarkingAlignedPage(t.Context(), fixture.queries, 1, "alice", 50, 500, 100, 1, staged); err != nil {
				t.Fatal(err)
			}
			if _, err := fixture.conn.Exec(tc.update); err != nil {
				t.Fatal(err)
			}
			if _, err := ResolveMarkingAlignedPage(t.Context(), fixture.queries, 1, "alice", 50, 500, 100, 1); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}
		})
	}
}

type markingAlignedPageFixture struct {
	conn      *sql.DB
	queries   *db.Queries
	workspace string
}

func newMarkingAlignedPageFixture(t *testing.T) markingAlignedPageFixture {
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
		CREATE TABLE marking_jobs(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam_page_content(student_exam_id INTEGER NOT NULL,page INTEGER NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE marking_copy_results(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,marking_job_id INTEGER NOT NULL,student_exam_id INTEGER NOT NULL,outcome TEXT NOT NULL,expected_pages INTEGER NOT NULL,detected_pages INTEGER NOT NULL,completed_at TIMESTAMP NOT NULL);
		CREATE TABLE marking_aligned_pages(id INTEGER PRIMARY KEY AUTOINCREMENT,user_id INTEGER NOT NULL,copy_result_id INTEGER NOT NULL,page_exam INTEGER NOT NULL,storage_key TEXT NOT NULL,width INTEGER NOT NULL,height INTEGER NOT NULL,sha256 TEXT NOT NULL,created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,UNIQUE(copy_result_id,page_exam));
		INSERT INTO users VALUES(1,'alice'),(2,'bob');
		INSERT INTO marking_jobs VALUES(50,1);
		INSERT INTO student_exam_page_content VALUES(100,1,1),(100,2,1),(101,1,1);
		INSERT INTO marking_copy_results VALUES(500,1,50,100,'corrected',2,2,CURRENT_TIMESTAMP);
	`); err != nil {
		t.Fatal(err)
	}
	workspace, ok := CreateOperationTempDir("alice", "marking-50")
	if !ok {
		t.Fatal("create marking workspace")
	}
	return markingAlignedPageFixture{conn: conn, queries: db.New(conn), workspace: workspace}
}

func alignedTestPNG(t *testing.T, width, height int, annotated bool) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetGray(x, y, color.Gray{Y: 230})
		}
	}
	for y := 14; y < 26; y++ {
		for x := 14; x < 26; x++ {
			img.SetGray(x, y, color.Gray{Y: 40})
		}
	}
	if annotated {
		for x := 0; x < width; x++ {
			img.SetGray(x, 0, color.Gray{Y: 0})
		}
	}
	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func alignedInt(value int64) string { return strconv.FormatInt(value, 10) }
