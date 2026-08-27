package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const (
	testPreviewUUID = "11111111-1111-4111-8111-111111111111"
	testMiniUUID    = "22222222-2222-4222-8222-222222222222"
)

func TestIsEphemeralWorkspaceName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "preview-" + testPreviewUUID, want: true},
		{name: "mini-" + testMiniUUID, want: true},
		{name: "preview-invalid"},
		{name: "mini-invalid"},
		{name: "preview-" + testPreviewUUID + "-extra"},
		{name: "exam-" + testPreviewUUID},
		{name: "marking-" + testPreviewUUID},
		{name: "custom-folder"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isEphemeralWorkspaceName(tc.name); got != tc.want {
				t.Fatalf("isEphemeralWorkspaceName(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestPurgeExpiredEphemeralWorkspacesAtRootByAgeAndIdentity(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assets", "tmp")
	now := time.Date(2030, 1, 2, 15, 0, 0, 0, time.UTC)

	oldPreview := makeEphemeralTestDir(t, root, "alice", "preview-"+testPreviewUUID, now.Add(-time.Hour-time.Second))
	oldMini := makeEphemeralTestDir(t, root, "alice", "mini-"+testMiniUUID, now.Add(-time.Hour-time.Second))
	recentPreview := makeEphemeralTestDir(t, root, "alice", "preview-33333333-3333-4333-8333-333333333333", now.Add(-59*time.Minute))
	recentMini := makeEphemeralTestDir(t, root, "alice", "mini-44444444-4444-4444-8444-444444444444", now.Add(-time.Minute))
	atCutoff := makeEphemeralTestDir(t, root, "alice", "preview-55555555-5555-4555-8555-555555555555", now.Add(-time.Hour))
	exam := makeEphemeralTestDir(t, root, "alice", "exam-42", now.Add(-24*time.Hour))
	marking := makeEphemeralTestDir(t, root, "alice", "marking-42", now.Add(-24*time.Hour))
	custom := makeEphemeralTestDir(t, root, "alice", "custom-folder", now.Add(-24*time.Hour))
	userDir := filepath.Join(root, "alice")

	if err := purgeExpiredEphemeralWorkspacesAtRoot(root, now); err != nil {
		t.Fatalf("purgeExpiredEphemeralWorkspacesAtRoot: %v", err)
	}

	assertEphemeralPathAbsent(t, oldPreview)
	assertEphemeralPathAbsent(t, oldMini)
	for _, path := range []string{recentPreview, recentMini, atCutoff, exam, marking, custom, userDir, root} {
		assertEphemeralPathPresent(t, path)
	}
}

func TestPurgeExpiredUserEphemeralWorkspacesIsUserScoped(t *testing.T) {
	root := filepath.Join(t.TempDir(), "assets", "tmp")
	now := time.Date(2030, 1, 2, 15, 0, 0, 0, time.UTC)
	alice := makeEphemeralTestDir(t, root, "alice", "preview-"+testPreviewUUID, now.Add(-time.Hour-time.Second))
	bob := makeEphemeralTestDir(t, root, "bob", "preview-"+testMiniUUID, now.Add(-time.Hour-time.Second))

	if err := purgeExpiredUserEphemeralWorkspacesAtRoot(root, "alice", now); err != nil {
		t.Fatalf("user-scoped purge: %v", err)
	}
	assertEphemeralPathAbsent(t, alice)
	assertEphemeralPathPresent(t, bob)

	if err := purgeExpiredEphemeralWorkspacesAtRoot(root, now); err != nil {
		t.Fatalf("global purge: %v", err)
	}
	assertEphemeralPathAbsent(t, bob)
}

func TestPurgeExpiredEphemeralWorkspacesAcceptsAbsentRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing", "tmp")
	if err := purgeExpiredEphemeralWorkspacesAtRoot(root, time.Now()); err != nil {
		t.Fatalf("absent root: %v", err)
	}
	if err := purgeExpiredUserEphemeralWorkspacesAtRoot(root, "alice", time.Now()); err != nil {
		t.Fatalf("absent user root: %v", err)
	}
}

func TestPurgeExpiredEphemeralWorkspacesIgnoresSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires additional privileges on Windows")
	}
	base := t.TempDir()
	root := filepath.Join(base, "assets", "tmp")
	now := time.Date(2030, 1, 2, 15, 0, 0, 0, time.UTC)

	userTarget := makeEphemeralTestDir(t, filepath.Join(base, "user-target-root"), "target-user", "preview-"+testPreviewUUID, now.Add(-2*time.Hour))
	if err := os.MkdirAll(root, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(userTarget), filepath.Join(root, "linked-user")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	operationTarget := filepath.Join(base, "operation-target")
	if err := os.MkdirAll(operationTarget, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(operationTarget, "marker"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}
	bobDir := filepath.Join(root, "bob")
	if err := os.MkdirAll(bobDir, 0o750); err != nil {
		t.Fatal(err)
	}
	operationLink := filepath.Join(bobDir, "preview-"+testMiniUUID)
	if err := os.Symlink(operationTarget, operationLink); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if err := purgeExpiredEphemeralWorkspacesAtRoot(root, now); err != nil {
		t.Fatalf("purge with symlinks: %v", err)
	}
	assertEphemeralPathPresent(t, userTarget)
	assertEphemeralPathPresent(t, filepath.Join(operationTarget, "marker"))
	if info, err := os.Lstat(filepath.Join(root, "linked-user")); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("user symlink was changed: info=%v err=%v", info, err)
	}
	if info, err := os.Lstat(operationLink); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("operation symlink was changed: info=%v err=%v", info, err)
	}
}

func TestPurgeExpiredEphemeralWorkspacesRejectsSymlinkedRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires additional privileges on Windows")
	}
	base := t.TempDir()
	outside := t.TempDir()
	root := filepath.Join(base, "tmp-link")
	if err := os.Symlink(outside, root); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	marker := filepath.Join(outside, "marker")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := purgeExpiredEphemeralWorkspacesAtRoot(root, time.Now()); err == nil {
		t.Fatal("purge accepted symlinked root")
	}
	assertEphemeralPathPresent(t, marker)
}

func TestPurgeExpiredUserEphemeralWorkspacesRejectsInvalidUsername(t *testing.T) {
	root := t.TempDir()
	if err := purgeExpiredUserEphemeralWorkspacesAtRoot(root, "../alice", time.Now()); err == nil {
		t.Fatal("expected invalid username error")
	}
}

func makeEphemeralTestDir(t *testing.T, root, username, operation string, modTime time.Time) string {
	t.Helper()
	path := filepath.Join(root, username, operation)
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatalf("create test workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(path, "marker"), []byte("test"), 0o600); err != nil {
		t.Fatalf("write test marker: %v", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("set workspace time: %v", err)
	}
	return path
}

func assertEphemeralPathAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("path %q still exists: %v", path, err)
	}
}

func assertEphemeralPathPresent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("path %q should exist: %v", path, err)
	}
}
