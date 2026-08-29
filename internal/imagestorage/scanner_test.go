package imagestorage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func TestScan(t *testing.T) {
	t.Run("consistent", func(t *testing.T) {
		queries, conn := setupScannerTest(t)
		insertMainName(t, conn, "a.png")
		insertVariantName(t, conn, "b.jpg")
		writeImageFile(t, "a.png")
		writeImageFile(t, "b.jpg")

		got := scanWithoutError(t, queries)
		assertConsistency(t, got, Consistency{})
	})

	t.Run("filesystem orphan", func(t *testing.T) {
		queries, conn := setupScannerTest(t)
		insertMainName(t, conn, "a.png")
		writeImageFile(t, "a.png")
		writeImageFile(t, "orphan.png")

		got := scanWithoutError(t, queries)
		assertConsistency(t, got, Consistency{Orphans: []string{"orphan.png"}})
	})

	t.Run("missing database file", func(t *testing.T) {
		queries, conn := setupScannerTest(t)
		insertMainName(t, conn, "a.png")
		insertMainName(t, conn, "missing.png")
		writeImageFile(t, "a.png")

		got := scanWithoutError(t, queries)
		assertConsistency(t, got, Consistency{Missing: []MissingImage{{
			Name: "missing.png", ReferenceType: MainImageReference,
		}}})
	})

	t.Run("mixed main variant and duplicate references", func(t *testing.T) {
		queries, conn := setupScannerTest(t)
		insertMainName(t, conn, "both.png")
		insertVariantName(t, conn, "both.png")
		insertMainName(t, conn, "main-ok.png")
		insertVariantName(t, conn, "variant-missing.png")
		writeImageFile(t, "main-ok.png")
		writeImageFile(t, "orphan-a.png")
		writeImageFile(t, "orphan-z.png")

		got := scanWithoutError(t, queries)
		assertConsistency(t, got, Consistency{
			Orphans: []string{"orphan-a.png", "orphan-z.png"},
			Missing: []MissingImage{
				{Name: "both.png", ReferenceType: MainAndVariantReference},
				{Name: "variant-missing.png", ReferenceType: VariantImageReference},
			},
		})
	})

	t.Run("symlink is unsafe and not followed", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation commonly requires additional privileges on Windows")
		}
		queries, _ := setupScannerTest(t)
		target := filepath.Join(t.TempDir(), "target.png")
		if err := os.WriteFile(target, []byte("outside"), 0o640); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(config.ImageSavePath, "link.png")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}

		got := scanWithoutError(t, queries)
		assertConsistency(t, got, Consistency{Unsafe: []UnsafeEntry{{
			Name: "link.png", Source: FilesystemSource, Kind: SymlinkEntry,
		}}})
	})

	t.Run("subdirectory is unsafe and not traversed", func(t *testing.T) {
		queries, _ := setupScannerTest(t)
		if err := os.Mkdir(filepath.Join(config.ImageSavePath, "subdir"), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(config.ImageSavePath, "subdir", "nested.png"), []byte("nested"), 0o640); err != nil {
			t.Fatal(err)
		}

		got := scanWithoutError(t, queries)
		assertConsistency(t, got, Consistency{Unsafe: []UnsafeEntry{{
			Name: "subdir", Source: FilesystemSource, Kind: DirectoryEntry,
		}}})
	})

	t.Run("filesystem read failure rejects partial scan", func(t *testing.T) {
		queries, _ := setupScannerTestWithoutDirectory(t)
		if err := os.MkdirAll(filepath.Dir(config.ImageSavePath), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(config.ImageSavePath, []byte("not a directory"), 0o640); err != nil {
			t.Fatal(err)
		}

		if _, err := Scan(context.Background(), queries); err == nil {
			t.Fatal("Scan succeeded with a non-directory image path")
		}
	})

	t.Run("unsafe database name is reported and never resolved", func(t *testing.T) {
		queries, conn := setupScannerTest(t)
		insertMainName(t, conn, "../outside.png")
		outside := filepath.Join(filepath.Dir(config.ImageSavePath), "outside.png")
		if err := os.WriteFile(outside, []byte("outside"), 0o640); err != nil {
			t.Fatal(err)
		}

		got := scanWithoutError(t, queries)
		assertConsistency(t, got, Consistency{Unsafe: []UnsafeEntry{{
			Name: "../outside.png", Source: DatabaseSource, Kind: UnsafeName,
		}}})
	})

	t.Run("absent directory with empty database is consistent", func(t *testing.T) {
		queries, _ := setupScannerTestWithoutDirectory(t)
		got := scanWithoutError(t, queries)
		assertConsistency(t, got, Consistency{})
	})

	t.Run("absent directory with references is an error", func(t *testing.T) {
		queries, conn := setupScannerTestWithoutDirectory(t)
		insertMainName(t, conn, "missing.png")

		if _, err := Scan(context.Background(), queries); err == nil {
			t.Fatal("Scan succeeded with database references and no image directory")
		}
	})

	t.Run("database read failure rejects partial scan", func(t *testing.T) {
		queries, conn := setupScannerTest(t)
		if _, err := conn.Exec("DROP TABLE alt_images"); err != nil {
			t.Fatal(err)
		}

		if _, err := Scan(context.Background(), queries); err == nil {
			t.Fatal("Scan succeeded after a database read failure")
		}
	})
}

func setupScannerTest(t *testing.T) (*db.Queries, *sql.DB) {
	t.Helper()
	queries, conn := setupScannerTestWithoutDirectory(t)
	if err := os.MkdirAll(config.ImageSavePath, 0o750); err != nil {
		t.Fatal(err)
	}
	return queries, conn
}

func setupScannerTestWithoutDirectory(t *testing.T) (*db.Queries, *sql.DB) {
	t.Helper()
	t.Chdir(t.TempDir())
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := conn.Exec(`
CREATE TABLE images(id INTEGER PRIMARY KEY, image_name TEXT NOT NULL);
CREATE TABLE alt_images(id INTEGER PRIMARY KEY, image_name TEXT NOT NULL);`); err != nil {
		t.Fatal(err)
	}
	return db.New(conn), conn
}

func insertMainName(t *testing.T, conn *sql.DB, name string) {
	t.Helper()
	if _, err := conn.Exec("INSERT INTO images(image_name) VALUES (?)", name); err != nil {
		t.Fatal(err)
	}
}

func insertVariantName(t *testing.T, conn *sql.DB, name string) {
	t.Helper()
	if _, err := conn.Exec("INSERT INTO alt_images(image_name) VALUES (?)", name); err != nil {
		t.Fatal(err)
	}
}

func writeImageFile(t *testing.T, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(config.ImageSavePath, name), []byte(name), 0o640); err != nil {
		t.Fatal(err)
	}
}

func scanWithoutError(t *testing.T, queries *db.Queries) Consistency {
	t.Helper()
	got, err := Scan(context.Background(), queries)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func assertConsistency(t *testing.T, got, want Consistency) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("consistency = %#v, want %#v", got, want)
	}
}
