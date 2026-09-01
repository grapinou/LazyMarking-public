package tools

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestResolveMarkingRejectedScanPagesUsesOriginalScanOrder(t *testing.T) {
	workspace := t.TempDir()
	splitPages := make([]string, 12)
	for index := range splitPages {
		splitPages[index] = filepath.Join(workspace, "page-"+strconv.Itoa(index+1)+".pdf")
	}
	exams := []config.Exam{
		{StudentExamID: 1, Pages: []config.Page{{Number: 1, Name: "page-2.png"}, {Number: 2, Name: "page-3.png"}}},
		{StudentExamID: 2, Pages: []config.Page{{Number: 1, Name: "page-12.png"}, {Number: 2, Name: "page-1.png"}}},
	}
	marked := []config.MarkExam{{StudentExamID: 1, Status: true}}

	got, err := ResolveMarkingRejectedScanPages(splitPages, []string{"page-10.png"}, exams, marked)
	if err != nil {
		t.Fatal(err)
	}
	numbers := make([]int, len(got))
	for index, page := range got {
		numbers[index] = page.ScanPage
	}
	if want := []int{1, 10, 12}; !reflect.DeepEqual(numbers, want) {
		t.Fatalf("rejected scan pages=%v, want %v", numbers, want)
	}
}

func TestBuildUncorrectedPagesPDFWithoutRejectedPagesCreatesNothing(t *testing.T) {
	workspace := t.TempDir()
	name, err := BuildUncorrectedPagesPDF(workspace, nil)
	if err != nil || name != "" {
		t.Fatalf("name=%q err=%v", name, err)
	}
	if _, err := os.Lstat(filepath.Join(workspace, MarkingUncorrectedPagesFilename)); !os.IsNotExist(err) {
		t.Fatalf("empty artifact exists: %v", err)
	}
}

func TestBuildUncorrectedPagesPDFPublishesValidOrderedArtifact(t *testing.T) {
	workspace := t.TempDir()
	colors := map[int]color.RGBA{
		2:  {R: 255, A: 255},
		10: {G: 255, A: 255},
		12: {B: 255, A: 255},
	}
	pages := make([]MarkingRejectedScanPage, 0, len(colors))
	for number, pageColor := range colors {
		path := filepath.Join(workspace, "page-"+strconv.Itoa(number)+".png")
		writeSolidMarkingPNG(t, path, pageColor)
		pages = append(pages, MarkingRejectedScanPage{ScanPage: number, Path: path})
	}
	// Deliberately reverse a non-lexical input order: publication must use scan
	// page 2, 10, 12 rather than caller or page-1/page-10 lexical order.
	sort.Slice(pages, func(i, j int) bool { return pages[i].ScanPage > pages[j].ScanPage })

	name, err := BuildUncorrectedPagesPDF(workspace, pages)
	if err != nil {
		t.Fatal(err)
	}
	if name != MarkingUncorrectedPagesFilename {
		t.Fatalf("name=%q", name)
	}
	artifact := filepath.Join(workspace, name)
	content, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) < 5 || !bytes.Equal(content[:5], []byte("%PDF-")) {
		t.Fatal("published artifact is not a PDF")
	}
	for _, page := range pages {
		if _, err := os.Stat(page.Path); err != nil {
			t.Fatalf("source scan page was modified: %v", err)
		}
	}
	assertExtractedPDFImageColors(t, artifact, []color.RGBA{colors[2], colors[10], colors[12]})
	assertNoCorrectedNotStaging(t, workspace)
}

func TestBuildUncorrectedPagesPDFFailureDoesNotPublishPartialCanonical(t *testing.T) {
	workspace := t.TempDir()
	valid := filepath.Join(workspace, "page-1.png")
	writeSolidMarkingPNG(t, valid, color.RGBA{R: 255, A: 255})
	missing := filepath.Join(workspace, "page-2.png")

	_, err := BuildUncorrectedPagesPDF(workspace, []MarkingRejectedScanPage{
		{ScanPage: 1, Path: valid},
		{ScanPage: 2, Path: missing},
	})
	if err == nil {
		t.Fatal("missing rejected page was accepted")
	}
	if _, err := os.Lstat(filepath.Join(workspace, MarkingUncorrectedPagesFilename)); !os.IsNotExist(err) {
		t.Fatalf("partial canonical artifact exists: %v", err)
	}
	assertNoCorrectedNotStaging(t, workspace)
}

func TestBuildUncorrectedPagesPDFRejectsSourceOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	outside := filepath.Join(t.TempDir(), "page-1.png")
	writeSolidMarkingPNG(t, outside, color.RGBA{A: 255})
	if _, err := BuildUncorrectedPagesPDF(workspace, []MarkingRejectedScanPage{{ScanPage: 1, Path: outside}}); err == nil {
		t.Fatal("outside rejected page was accepted")
	}
	if _, err := os.Lstat(filepath.Join(workspace, MarkingUncorrectedPagesFilename)); !os.IsNotExist(err) {
		t.Fatalf("canonical artifact exists: %v", err)
	}
}

func TestMarkingArtifactExistsDetectsCorrectedNotWithoutMutation(t *testing.T) {
	t.Chdir(t.TempDir())
	workspace, ok := CreateOperationTempDir("alice", "marking-7")
	if !ok {
		t.Fatal("create marking workspace")
	}
	artifact := filepath.Join(workspace, MarkingUncorrectedPagesFilename)
	want := []byte("%PDF-durable-uncorrected")
	if err := os.WriteFile(artifact, want, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(artifact)
	if err != nil {
		t.Fatal(err)
	}

	exists, err := MarkingArtifactExists("alice", 7, MarkingUncorrectedPagesFilename)
	if err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
	after, err := os.Stat(artifact)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) || before.ModTime() != after.ModTime() || before.Size() != after.Size() {
		t.Fatal("artifact inspection modified corrected_NOT.pdf")
	}
}

func assertExtractedPDFImageColors(t *testing.T, artifact string, want []color.RGBA) {
	t.Helper()
	extractDir := t.TempDir()
	prefix := filepath.Join(extractDir, "page")
	if output, err := exec.Command("pdfimages", "-png", artifact, prefix).CombinedOutput(); err != nil {
		t.Fatalf("extract PDF images: %v: %s", err, output)
	}
	files, err := filepath.Glob(prefix + "-*.png")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if len(files) != len(want) {
		t.Fatalf("extracted images=%v, want %d", files, len(want))
	}
	for index, path := range files {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		got := color.RGBAModel.Convert(img.At(0, 0)).(color.RGBA)
		if got.R != want[index].R || got.G != want[index].G || got.B != want[index].B {
			t.Fatalf("image %d color=%v, want %v", index, got, want[index])
		}
	}
}

func assertNoCorrectedNotStaging(t *testing.T, workspace string) {
	t.Helper()
	entries, err := os.ReadDir(workspace)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".corrected-not-") {
			t.Fatalf("staging remains: %s", entry.Name())
		}
	}
}

func writeSolidMarkingPNG(t *testing.T, path string, fill color.RGBA) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
