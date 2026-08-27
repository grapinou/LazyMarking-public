package tools

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveUploadedFileRejectsSymlinkedDestination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires additional privileges on Windows")
	}
	base := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(base, "images")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	file := &testMultipartFile{Reader: bytes.NewReader([]byte("content"))}
	if err := SaveUploadedFile(file, link, "image.png"); err == nil {
		t.Fatal("SaveUploadedFile accepted symlinked destination")
	}
	if _, err := os.Stat(filepath.Join(outside, "image.png")); !os.IsNotExist(err) {
		t.Fatalf("file escaped through destination symlink: %v", err)
	}
}

func TestSaveUploadedFileDoesNotFollowExistingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires additional privileges on Windows")
	}
	destination := t.TempDir()
	target := filepath.Join(t.TempDir(), "target.png")
	if err := os.WriteFile(target, []byte("keep"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(destination, "image.png")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	file := &testMultipartFile{Reader: bytes.NewReader([]byte("replacement"))}
	if err := SaveUploadedFile(file, destination, "image.png"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("error=%v, want os.ErrExist", err)
	}
	if contents, err := os.ReadFile(target); err != nil || string(contents) != "keep" {
		t.Fatalf("symlink target changed: contents=%q err=%v", contents, err)
	}
}

func TestSaveUploadedFileCreatesNewFileAndParentDirectory(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "missing", "images")
	file := &testMultipartFile{Reader: bytes.NewReader([]byte("new-content"))}

	if err := SaveUploadedFile(file, destination, "image.png"); err != nil {
		t.Fatalf("SaveUploadedFile: %v", err)
	}
	contents, err := os.ReadFile(filepath.Join(destination, "image.png"))
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(contents) != "new-content" {
		t.Fatalf("uploaded content = %q, want %q", contents, "new-content")
	}
}

func TestSaveUploadedFilePreservesExistingFile(t *testing.T) {
	destination := t.TempDir()
	path := filepath.Join(destination, "image.png")
	if err := os.WriteFile(path, []byte("original-content"), 0o640); err != nil {
		t.Fatal(err)
	}

	file := &testMultipartFile{Reader: bytes.NewReader([]byte("new-content"))}
	err := SaveUploadedFile(file, destination, "image.png")
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("SaveUploadedFile error = %v, want os.ErrExist", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read original file: %v", err)
	}
	if string(contents) != "original-content" {
		t.Fatalf("existing content = %q, want %q", contents, "original-content")
	}
}

func TestSaveUploadedFileRejectsUnsafeFilename(t *testing.T) {
	file := &testMultipartFile{Reader: bytes.NewReader([]byte("content"))}
	if err := SaveUploadedFile(file, t.TempDir(), "../escape.png"); err == nil {
		t.Fatal("SaveUploadedFile accepted unsafe filename")
	}
}

func TestSaveUploadedFileRemovesPartialFileAfterCopyError(t *testing.T) {
	destination := t.TempDir()
	file := &failingMultipartFile{
		reader:    bytes.NewReader([]byte("partial-content")),
		failAfter: 7,
	}

	err := SaveUploadedFile(file, destination, "image.png")
	if !errors.Is(err, errSyntheticUploadRead) {
		t.Fatalf("SaveUploadedFile error = %v, want synthetic read error", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "image.png")); !os.IsNotExist(err) {
		t.Fatalf("partial upload still exists: %v", err)
	}
}

type testMultipartFile struct {
	*bytes.Reader
}

func (f *testMultipartFile) Close() error { return nil }

var errSyntheticUploadRead = errors.New("synthetic upload read error")

type failingMultipartFile struct {
	reader    *bytes.Reader
	failAfter int64
	read      int64
}

func (f *failingMultipartFile) Read(p []byte) (int, error) {
	if f.read >= f.failAfter {
		return 0, errSyntheticUploadRead
	}
	remaining := f.failAfter - f.read
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := f.reader.Read(p)
	f.read += int64(n)
	if err == io.EOF && f.read >= f.failAfter {
		return n, errSyntheticUploadRead
	}
	return n, err
}

func (f *failingMultipartFile) ReadAt(p []byte, off int64) (int, error) {
	return f.reader.ReadAt(p, off)
}

func (f *failingMultipartFile) Seek(offset int64, whence int) (int64, error) {
	return f.reader.Seek(offset, whence)
}

func (f *failingMultipartFile) Close() error { return nil }
