package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestRemoveStoredImageFileRemovesOnlyRequestedFile(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(config.ImageSavePath, 0o750); err != nil {
		t.Fatal(err)
	}
	requested := filepath.Join(config.ImageSavePath, "requested.png")
	sibling := filepath.Join(config.ImageSavePath, "sibling.png")
	if err := os.WriteFile(requested, []byte("requested"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sibling, []byte("sibling"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := RemoveStoredImageFile("requested.png"); err != nil {
		t.Fatalf("RemoveStoredImageFile: %v", err)
	}
	if _, err := os.Stat(requested); !os.IsNotExist(err) {
		t.Fatalf("requested file still exists: %v", err)
	}
	if contents, err := os.ReadFile(sibling); err != nil || string(contents) != "sibling" {
		t.Fatalf("sibling changed: contents=%q err=%v", contents, err)
	}
	if info, err := os.Stat(config.ImageSavePath); err != nil || !info.IsDir() {
		t.Fatalf("image parent was removed: info=%v err=%v", info, err)
	}
}

func TestRemoveStoredImageFileAcceptsAbsentFile(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := RemoveStoredImageFile("absent.png"); err != nil {
		t.Fatalf("absent file: %v", err)
	}
}

func TestRemoveStoredImageFileRejectsUnsafeFilename(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := RemoveStoredImageFile("../escape.png"); err == nil {
		t.Fatal("expected unsafe filename error")
	}
}

func TestRemoveStoredImageFileRemovesSymlinkWithoutFollowingTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires additional privileges on Windows")
	}
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(config.ImageSavePath, 0o750); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target.png")
	if err := os.WriteFile(target, []byte("target-content"), 0o640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(config.ImageSavePath, "linked.png")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if err := RemoveStoredImageFile("linked.png"); err != nil {
		t.Fatalf("RemoveStoredImageFile symlink: %v", err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("symlink still exists: %v", err)
	}
	if contents, err := os.ReadFile(target); err != nil || string(contents) != "target-content" {
		t.Fatalf("symlink target changed: contents=%q err=%v", contents, err)
	}
}
