package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const EphemeralWorkspaceRetention = 1 * time.Hour

var operationWorkspacesRoot = filepath.Join("assets", "tmp")

// PurgeExpiredEphemeralWorkspaces removes expired preview and mini workspaces
// for every direct user directory under assets/tmp.
func PurgeExpiredEphemeralWorkspaces(now time.Time) error {
	return purgeExpiredEphemeralWorkspacesAtRoot(operationWorkspacesRoot, now)
}

// PurgeExpiredUserEphemeralWorkspaces removes expired preview and mini
// workspaces belonging to one validated user.
func PurgeExpiredUserEphemeralWorkspaces(username string, now time.Time) error {
	if err := safePathComponent(username); err != nil {
		return err
	}
	return purgeExpiredUserEphemeralWorkspacesAtRoot(operationWorkspacesRoot, username, now)
}

func purgeExpiredEphemeralWorkspacesAtRoot(root string, now time.Time) error {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || safePathComponent(entry.Name()) != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() {
			continue
		}
		if err := purgeExpiredUserEphemeralWorkspacesAtRoot(root, entry.Name(), now); err != nil {
			return err
		}
	}
	return nil
}

func purgeExpiredUserEphemeralWorkspacesAtRoot(root, username string, now time.Time) error {
	if err := safePathComponent(username); err != nil {
		return err
	}
	userDir := filepath.Join(root, username)
	userInfo, err := os.Lstat(userDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if userInfo.Mode()&os.ModeSymlink != 0 || !userInfo.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(userDir)
	if err != nil {
		return err
	}
	cutoff := now.Add(-EphemeralWorkspaceRetention)
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !isEphemeralWorkspaceName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.RemoveAll(filepath.Join(userDir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func isEphemeralWorkspaceName(name string) bool {
	var suffix string
	if value, ok := strings.CutPrefix(name, "preview-"); ok {
		suffix = value
	} else if value, ok := strings.CutPrefix(name, "mini-"); ok {
		suffix = value
	} else {
		return false
	}

	parsed, err := uuid.Parse(suffix)
	return err == nil && parsed.String() == suffix
}
