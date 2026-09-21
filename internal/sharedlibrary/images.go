package sharedlibrary

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/grapinou/LazyMarking/internal/filesafety"
)

// OpenImage rejects path traversal, symlinks and non-regular files. Callers must
// first authorize the DB image reference; a filename is never an authorization.
func OpenImage(directory, name string) (*os.File, error) {
	if !filesafety.IsSafePathComponent(name) {
		return nil, errors.New("unsafe image name")
	}
	if err := filesafety.ValidateDirectoryTree(directory); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errors.New("image is not a regular file")
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err != nil || !os.SameFile(before, after) {
		f.Close()
		return nil, errors.New("image changed while opening")
	}
	return f, nil
}

func copyImage(directory, name string) (string, error) {
	source, err := OpenImage(directory, name)
	if err != nil {
		return "", err
	}
	defer source.Close()
	newName := "copy_" + uuid.NewString() + filepath.Ext(name)
	root, err := os.OpenRoot(directory)
	if err != nil {
		return "", err
	}
	defer root.Close()
	dest, err := root.OpenFile(newName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(dest, source)
	syncErr := dest.Sync()
	closeErr := dest.Close()
	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		return "", errors.Join(err, root.Remove(newName))
	}
	return newName, nil
}
