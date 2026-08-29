package imagestorage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestPurgeOrphans(t *testing.T) {
	now := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)

	t.Run("default dry run keeps old orphan", func(t *testing.T) {
		queries, _ := setupScannerTest(t)
		path := writeDatedImageFile(t, "orphan.png", now.Add(-25*time.Hour))
		options := DefaultPurgeOptions()
		options.Now = now

		result, err := PurgeOrphans(context.Background(), queries, options)
		if err != nil {
			t.Fatal(err)
		}
		assertStrings(t, result.Candidates, []string{"orphan.png"})
		assertStrings(t, result.Deleted, nil)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("dry-run removed orphan: %v", err)
		}
	})

	t.Run("execute deletes orphan exactly at grace boundary", func(t *testing.T) {
		queries, _ := setupScannerTest(t)
		path := writeDatedImageFile(t, "orphan.png", now.Add(-DefaultOrphanGracePeriod))

		result, err := PurgeOrphans(context.Background(), queries, PurgeOptions{
			Execute: true, GracePeriod: DefaultOrphanGracePeriod, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		assertStrings(t, result.Deleted, []string{"orphan.png"})
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("eligible orphan still exists: %v", err)
		}
	})

	t.Run("recent orphan is skipped", func(t *testing.T) {
		queries, _ := setupScannerTest(t)
		path := writeDatedImageFile(t, "recent.png", now.Add(-23*time.Hour))

		result, err := PurgeOrphans(context.Background(), queries, PurgeOptions{
			Execute: true, GracePeriod: DefaultOrphanGracePeriod, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		assertStrings(t, result.SkippedRecent, []string{"recent.png"})
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("recent orphan was removed: %v", err)
		}
	})

	t.Run("referenced file is never a candidate", func(t *testing.T) {
		queries, conn := setupScannerTest(t)
		insertMainName(t, conn, "referenced.png")
		path := writeDatedImageFile(t, "referenced.png", now.Add(-48*time.Hour))

		result, err := PurgeOrphans(context.Background(), queries, PurgeOptions{
			Execute: true, GracePeriod: DefaultOrphanGracePeriod, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		assertStrings(t, result.Candidates, nil)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("referenced file was removed: %v", err)
		}
	})

	t.Run("missing database file is never mutated", func(t *testing.T) {
		queries, conn := setupScannerTest(t)
		insertVariantName(t, conn, "missing.png")

		result, err := PurgeOrphans(context.Background(), queries, PurgeOptions{
			Execute: true, GracePeriod: DefaultOrphanGracePeriod, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		assertStrings(t, result.Candidates, nil)
		var count int
		if err := conn.QueryRow("SELECT COUNT(*) FROM alt_images WHERE image_name='missing.png'").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("missing DB references=%d, want 1", count)
		}
	})

	t.Run("unsafe entries are never removed", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation commonly requires additional privileges on Windows")
		}
		queries, conn := setupScannerTest(t)
		insertMainName(t, conn, "../outside.png")
		if err := os.Mkdir(filepath.Join(config.ImageSavePath, "subdir"), 0o750); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(t.TempDir(), "target.png")
		if err := os.WriteFile(target, []byte("target"), 0o640); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(config.ImageSavePath, "link.png")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}

		result, err := PurgeOrphans(context.Background(), queries, PurgeOptions{
			Execute: true, GracePeriod: 0, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		assertStrings(t, result.Candidates, nil)
		if _, err := os.Lstat(link); err != nil {
			t.Fatalf("unsafe symlink was removed: %v", err)
		}
		if _, err := os.Stat(filepath.Join(config.ImageSavePath, "subdir")); err != nil {
			t.Fatalf("unsafe directory was removed: %v", err)
		}
	})

	t.Run("symlinked storage root cannot delete external file", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation commonly requires additional privileges on Windows")
		}
		queries, _ := setupScannerTestWithoutDirectory(t)
		if err := os.MkdirAll(filepath.Dir(config.ImageSavePath), 0o750); err != nil {
			t.Fatal(err)
		}
		external := filepath.Join(t.TempDir(), "external")
		if err := os.Mkdir(external, 0o750); err != nil {
			t.Fatal(err)
		}
		externalFile := filepath.Join(external, "old-orphan.png")
		if err := os.WriteFile(externalFile, []byte("outside"), 0o640); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(externalFile, now.Add(-48*time.Hour), now.Add(-48*time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(external, config.ImageSavePath); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		removeCalls := 0
		originalRemove := removeOrphanFile
		removeOrphanFile = func(string) error { removeCalls++; return nil }
		t.Cleanup(func() { removeOrphanFile = originalRemove })

		if result, err := PurgeOrphans(context.Background(), queries, PurgeOptions{
			Execute: true, GracePeriod: DefaultOrphanGracePeriod, Now: now,
		}); err == nil {
			t.Fatalf("purge accepted a symlinked storage root: %#v", result)
		}
		if removeCalls != 0 {
			t.Fatalf("remove calls=%d, want 0", removeCalls)
		}
		if _, err := os.Stat(externalFile); err != nil {
			t.Fatalf("external file was removed or changed: %v", err)
		}
	})

	t.Run("reference created after scan blocks deletion", func(t *testing.T) {
		queries, conn := setupScannerTest(t)
		path := writeDatedImageFile(t, "orphan.png", now.Add(-48*time.Hour))
		originalScan := scanForOrphanPurge
		scanForOrphanPurge = func(ctx context.Context, q *db.Queries) (Consistency, error) {
			result, err := Scan(ctx, q)
			if err == nil {
				insertMainName(t, conn, "orphan.png")
			}
			return result, err
		}
		t.Cleanup(func() { scanForOrphanPurge = originalScan })

		result, err := PurgeOrphans(context.Background(), queries, PurgeOptions{
			Execute: true, GracePeriod: DefaultOrphanGracePeriod, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		assertStrings(t, result.SkippedReferenced, []string{"orphan.png"})
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("newly referenced file was removed: %v", err)
		}
	})

	t.Run("one remove failure does not stop later candidates", func(t *testing.T) {
		queries, _ := setupScannerTest(t)
		for _, name := range []string{"a.png", "b.png", "c.png"} {
			writeDatedImageFile(t, name, now.Add(-48*time.Hour))
		}
		originalRemove := removeOrphanFile
		removeOrphanFile = func(path string) error {
			if filepath.Base(path) == "b.png" {
				return errors.New("injected remove failure")
			}
			return os.Remove(path)
		}
		t.Cleanup(func() { removeOrphanFile = originalRemove })

		result, err := PurgeOrphans(context.Background(), queries, PurgeOptions{
			Execute: true, GracePeriod: DefaultOrphanGracePeriod, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		assertStrings(t, result.Deleted, []string{"a.png", "c.png"})
		if len(result.Failed) != 1 || result.Failed[0].Name != "b.png" {
			t.Fatalf("failures=%#v, want b.png", result.Failed)
		}
		for _, name := range []string{"a.png", "c.png"} {
			if _, err := os.Stat(filepath.Join(config.ImageSavePath, name)); !os.IsNotExist(err) {
				t.Fatalf("%s still exists: %v", name, err)
			}
		}
		if _, err := os.Stat(filepath.Join(config.ImageSavePath, "b.png")); err != nil {
			t.Fatalf("failed candidate b.png disappeared: %v", err)
		}
	})

	t.Run("scan failure prevents every removal", func(t *testing.T) {
		queries, _ := setupScannerTest(t)
		path := writeDatedImageFile(t, "orphan.png", now.Add(-48*time.Hour))
		originalScan := scanForOrphanPurge
		scanForOrphanPurge = func(context.Context, *db.Queries) (Consistency, error) {
			return Consistency{}, errors.New("injected scan failure")
		}
		t.Cleanup(func() { scanForOrphanPurge = originalScan })
		removeCalls := 0
		originalRemove := removeOrphanFile
		removeOrphanFile = func(string) error { removeCalls++; return nil }
		t.Cleanup(func() { removeOrphanFile = originalRemove })

		if _, err := PurgeOrphans(context.Background(), queries, PurgeOptions{Execute: true, Now: now}); err == nil {
			t.Fatal("purge succeeded after scan failure")
		}
		if removeCalls != 0 {
			t.Fatalf("remove calls=%d, want 0", removeCalls)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("file changed after scan failure: %v", err)
		}
	})

	t.Run("reference recheck failure prevents every removal", func(t *testing.T) {
		queries, conn := setupScannerTest(t)
		path := writeDatedImageFile(t, "orphan.png", now.Add(-48*time.Hour))
		originalScan := scanForOrphanPurge
		scanForOrphanPurge = func(ctx context.Context, q *db.Queries) (Consistency, error) {
			result, err := Scan(ctx, q)
			if err == nil {
				if _, dropErr := conn.Exec("DROP TABLE alt_images"); dropErr != nil {
					t.Fatal(dropErr)
				}
			}
			return result, err
		}
		t.Cleanup(func() { scanForOrphanPurge = originalScan })
		removeCalls := 0
		originalRemove := removeOrphanFile
		removeOrphanFile = func(string) error { removeCalls++; return nil }
		t.Cleanup(func() { removeOrphanFile = originalRemove })

		if _, err := PurgeOrphans(context.Background(), queries, PurgeOptions{
			Execute: true, GracePeriod: DefaultOrphanGracePeriod, Now: now,
		}); err == nil {
			t.Fatal("purge succeeded after reference recheck failure")
		}
		if removeCalls != 0 {
			t.Fatalf("remove calls=%d, want 0", removeCalls)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("file changed after DB recheck failure: %v", err)
		}
	})
}

func writeDatedImageFile(t *testing.T, name string, modTime time.Time) string {
	t.Helper()
	path := filepath.Join(config.ImageSavePath, name)
	if err := os.WriteFile(path, []byte(name), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("values=%v, want %v", got, want)
	}
}
