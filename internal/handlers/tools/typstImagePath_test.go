package tools

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTypstImagePathIsStableAcrossWorkspaceDepths(t *testing.T) {
	want := filepath.Join("assets", "images", "test.png")
	for _, document := range []string{
		filepath.Join("assets", "tmp", "Sighto", "preview-123", "document.typ"),
		filepath.Join("assets", "tmp", "Sighto", "exam-42", "student-exam.typ"),
		filepath.Join("assets", "tmp", "Sighto", "mini-123", "Sighto_mini.typ"),
		filepath.Join("assets", "tmp", "Sighto", "operation", "subdir", "document.typ"),
	} {
		t.Run(document, func(t *testing.T) {
			got, err := typstImagePath("test.png")
			if err != nil {
				t.Fatal(err)
			}
			if got != "/assets/images/test.png" {
				t.Fatalf("typstImagePath()=%q, want root-relative image path", got)
			}
			resolved := filepath.FromSlash(strings.TrimPrefix(got, "/"))
			if resolved != want {
				t.Fatalf("document=%q resolved=%q, want %q", document, resolved, want)
			}
		})
	}
}

func TestTypstImagePathRejectsPathComponents(t *testing.T) {
	for _, name := range []string{"../escape.png", "subdir/image.png", `subdir\image.png`, ""} {
		if got, err := typstImagePath(name); err == nil || got != "" {
			t.Fatalf("typstImagePath(%q)=(%q, %v), want rejection", name, got, err)
		}
	}
}

func TestTypstCompilesRootRelativeImageFromNestedWorkspace(t *testing.T) {
	typstBinary, err := exec.LookPath("typst")
	if err != nil {
		t.Skip("typst executable is not available")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	imageDir := filepath.Join(root, "assets", "images")
	workspaceRoot := filepath.Join(root, "assets", "tmp")
	if err := os.MkdirAll(imageDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workspaceRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	workspaceBase, err := os.MkdirTemp(workspaceRoot, "typst-image-path-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(workspaceBase) })
	workspace := filepath.Join(workspaceBase, "operation", "subdir")
	if err := os.MkdirAll(workspace, 0o750); err != nil {
		t.Fatal(err)
	}

	var encoded bytes.Buffer
	pixel := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pixel.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&encoded, pixel); err != nil {
		t.Fatal(err)
	}
	imageFile, err := os.CreateTemp(imageDir, "typst-image-path-test-*.png")
	if err != nil {
		t.Fatal(err)
	}
	imageName := filepath.Base(imageFile.Name())
	t.Cleanup(func() { _ = os.Remove(imageFile.Name()) })
	if _, err := imageFile.Write(encoded.Bytes()); err != nil {
		_ = imageFile.Close()
		t.Fatal(err)
	}
	if err := imageFile.Close(); err != nil {
		t.Fatal(err)
	}
	imagePath, err := typstImagePath(imageName)
	if err != nil {
		t.Fatal(err)
	}
	document := filepath.Join(workspace, "document.typ")
	if err := os.WriteFile(document, []byte("#image("+typstStringLiteral(imagePath)+")"), 0o640); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(workspace, "document.pdf")
	documentRelative, err := filepath.Rel(root, document)
	if err != nil {
		t.Fatal(err)
	}
	outputRelative, err := filepath.Rel(root, output)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(typstBinary, "compile", "--root", root, documentRelative, outputRelative)
	command.Dir = root
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("Typst compilation failed: %v\n%s", err, combined)
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("compiled PDF missing or empty: info=%v err=%v", info, err)
	}
}
