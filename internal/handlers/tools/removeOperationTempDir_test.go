package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveOperationTempDirRemovesOnlyRequestedWorkspace(t *testing.T) {
	t.Chdir(t.TempDir())

	workspace := createOperationWorkspace(t, "alice", "exam-42")
	sibling := createOperationWorkspace(t, "alice", "exam-43")
	otherUser := createOperationWorkspace(t, "bob", "exam-42")
	userParent := filepath.Join("assets", "tmp", "alice")
	tmpParent := filepath.Join("assets", "tmp")

	if err := RemoveOperationTempDir("alice", "exam-42"); err != nil {
		t.Fatalf("RemoveOperationTempDir: %v", err)
	}
	assertFilePathAbsent(t, workspace)
	assertFilePathPresent(t, sibling)
	assertFilePathPresent(t, otherUser)
	assertFilePathPresent(t, userParent)
	assertFilePathPresent(t, tmpParent)
}

func TestRemoveOperationTempDirAcceptsAbsentWorkspace(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := RemoveOperationTempDir("alice", "exam-404"); err != nil {
		t.Fatalf("absent workspace: %v", err)
	}
}

func TestRemoveOperationTempDirRejectsInvalidComponents(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, test := range []struct {
		name      string
		username  string
		operation string
	}{
		{name: "invalid username", username: "../alice", operation: "exam-42"},
		{name: "invalid operation", username: "alice", operation: "../exam-42"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := RemoveOperationTempDir(test.username, test.operation); err == nil {
				t.Fatal("expected invalid component error")
			}
		})
	}
}

func createOperationWorkspace(t *testing.T, username, operation string) string {
	t.Helper()
	workspace, ok := CreateOperationTempDir(username, operation)
	if !ok {
		t.Fatalf("CreateOperationTempDir(%q, %q)", username, operation)
	}
	if err := os.WriteFile(filepath.Join(workspace, "marker"), []byte("test"), 0o600); err != nil {
		t.Fatalf("write workspace marker: %v", err)
	}
	return workspace
}

func assertFilePathAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("path %q still exists: %v", path, err)
	}
}

func assertFilePathPresent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("path %q should exist: %v", path, err)
	}
}
