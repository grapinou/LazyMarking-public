package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grapinou/LazyMarking/internal/config"
	appdb "github.com/grapinou/LazyMarking/internal/db"
)

var fixedMaintenanceTime = time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)

func TestRunImagesDryRunDoesNotDeleteOldOrphan(t *testing.T) {
	queries, conn := setupMaintenanceTest(t)
	defer conn.Close()
	createStoredFile(t, "old.png", fixedMaintenanceTime.Add(-48*time.Hour))

	code, stdout, _ := executeImagesCommand(t, queries)

	if code != 0 {
		t.Fatalf("runImages() code = %d, want 0", code)
	}
	assertPathExists(t, filepath.Join(config.ImageSavePath, "old.png"))
	assertOutputContains(t, stdout, "DRY RUN", "old.png")
}

func TestRunImagesExecuteDeletesOldOrphan(t *testing.T) {
	queries, conn := setupMaintenanceTest(t)
	defer conn.Close()
	createStoredFile(t, "old.png", fixedMaintenanceTime.Add(-48*time.Hour))

	code, stdout, _ := executeImagesCommand(t, queries, "--execute")

	if code != 0 {
		t.Fatalf("runImages() code = %d, want 0", code)
	}
	assertPathAbsent(t, filepath.Join(config.ImageSavePath, "old.png"))
	assertOutputContains(t, stdout, "EXÉCUTION", "Supprimés", "old.png")
}

func TestRunImagesExecuteKeepsRecentOrphan(t *testing.T) {
	queries, conn := setupMaintenanceTest(t)
	defer conn.Close()
	createStoredFile(t, "recent.png", fixedMaintenanceTime.Add(-time.Hour))

	code, stdout, _ := executeImagesCommand(t, queries, "--execute")

	if code != 0 {
		t.Fatalf("runImages() code = %d, want 0", code)
	}
	assertPathExists(t, filepath.Join(config.ImageSavePath, "recent.png"))
	assertOutputContains(t, stdout, "Ignorés car récents", "recent.png")
}

func TestRunImagesReportsMissingWithoutDatabaseMutation(t *testing.T) {
	queries, conn := setupMaintenanceTest(t)
	defer conn.Close()
	insertImageName(t, conn, "images", "missing.png")

	code, stdout, _ := executeImagesCommand(t, queries, "--execute")

	if code != 0 {
		t.Fatalf("runImages() code = %d, want 0", code)
	}
	assertOutputContains(t, stdout, "Missing", "missing.png", "aucune ligne DB modifiée")
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM images WHERE image_name = ?", "missing.png").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("missing DB reference count = %d, want 1", count)
	}
}

func TestRunImagesReportsUnsafeWithoutDeletingIt(t *testing.T) {
	queries, conn := setupMaintenanceTest(t)
	defer conn.Close()
	unsafePath := filepath.Join(config.ImageSavePath, "subdir")
	if err := os.Mkdir(unsafePath, 0o755); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := executeImagesCommand(t, queries, "--execute")

	if code != 0 {
		t.Fatalf("runImages() code = %d, want 0", code)
	}
	assertPathExists(t, unsafePath)
	assertOutputContains(t, stdout, "Unsafe", "subdir", "directory")
}

func TestRunImagesRejectsInvalidGraceBeforeAnyAction(t *testing.T) {
	queries, conn := setupMaintenanceTest(t)
	defer conn.Close()
	createStoredFile(t, "old.png", fixedMaintenanceTime.Add(-48*time.Hour))

	code, _, stderr := executeImagesCommand(t, queries, "--execute", "--grace", "-1h")

	if code == 0 {
		t.Fatal("runImages() code = 0, want non-zero")
	}
	assertPathExists(t, filepath.Join(config.ImageSavePath, "old.png"))
	assertOutputContains(t, stderr, "ne peut pas être négatif")
}

func TestRunImagesRejectsInvalidStorageRootWithoutDeletingOutsideFile(t *testing.T) {
	queries, conn := setupMaintenanceTest(t)
	defer conn.Close()
	if err := os.Remove(config.ImageSavePath); err != nil {
		t.Fatal(err)
	}
	externalPath := filepath.Join(t.TempDir(), "old-orphan.png")
	if err := os.WriteFile(externalPath, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(externalPath), config.ImageSavePath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	code, _, stderr := executeImagesCommand(t, queries, "--execute")

	if code == 0 {
		t.Fatal("runImages() code = 0, want non-zero")
	}
	assertPathExists(t, externalPath)
	assertOutputContains(t, stderr, "Audit du stockage impossible")
}

func setupMaintenanceTest(t *testing.T) (*appdb.Queries, *sql.DB) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(config.ImageSavePath, 0o755); err != nil {
		t.Fatal(err)
	}
	conn, err := appdb.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	for _, statement := range []string{
		"CREATE TABLE images (id INTEGER PRIMARY KEY, image_name TEXT NOT NULL)",
		"CREATE TABLE alt_images (id INTEGER PRIMARY KEY, image_name TEXT NOT NULL)",
	} {
		if _, err := conn.Exec(statement); err != nil {
			conn.Close()
			t.Fatal(err)
		}
	}
	previousNow := maintenanceNow
	maintenanceNow = func() time.Time { return fixedMaintenanceTime }
	t.Cleanup(func() { maintenanceNow = previousNow })
	return appdb.New(conn), conn
}

func executeImagesCommand(t *testing.T, queries *appdb.Queries, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runImages(context.Background(), args, &stdout, &stderr, queries)
	return code, stdout.String(), stderr.String()
}

func createStoredFile(t *testing.T, name string, modTime time.Time) {
	t.Helper()
	path := filepath.Join(config.ImageSavePath, name)
	if err := os.WriteFile(path, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}

func insertImageName(t *testing.T, conn *sql.DB, table, name string) {
	t.Helper()
	if _, err := conn.Exec("INSERT INTO "+table+" (image_name) VALUES (?)", name); err != nil {
		t.Fatal(err)
	}
}

func assertOutputContains(t *testing.T, output string, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(output, value) {
			t.Errorf("output %q does not contain %q", output, value)
		}
	}
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("expected %q to exist: %v", path, err)
	}
}

func assertPathAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %q to be absent, got %v", path, err)
	}
}
