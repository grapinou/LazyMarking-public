package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestResolveMarkingPageReferencesUsesDurablePNGsWithoutLegacyRenderer(t *testing.T) {
	fixture := newPageReferenceResolverFixture(t)
	page1 := fixture.addPNG(t, 1, 31, 41)
	page2 := fixture.addPNG(t, 2, 32, 42)

	pageNumbers := []int{2, 1}
	sort.Ints(pageNumbers) // MarkingStudentExam preserves this existing ordering step.
	renderCalls := 0
	references, cleanup, err := resolveMarkingPageReferences(
		context.Background(), fixture.queries, 1, "alice", 100, pageNumbers,
		func() ([]string, []string, error) {
			renderCalls++
			return nil, nil, errors.New("legacy renderer must not run")
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if renderCalls != 0 || len(cleanup) != 0 {
		t.Fatalf("legacy renderer calls=%d cleanup=%v", renderCalls, cleanup)
	}
	want := []string{page1.path, page2.path}
	for i := range want {
		wantAbsolute, err := filepath.Abs(want[i])
		if err != nil {
			t.Fatal(err)
		}
		if references[i] != wantAbsolute {
			t.Fatalf("reference[%d]=%q, want %q", i, references[i], wantAbsolute)
		}
	}
}

func TestResolveMarkingPageReferencesNeverFallsBackOnCorruption(t *testing.T) {
	fixture := newPageReferenceResolverFixture(t)
	page := fixture.addPNG(t, 1, 31, 41)
	if err := os.WriteFile(page.path, []byte("corrupt after metadata"), 0o600); err != nil {
		t.Fatal(err)
	}
	renderCalls := 0
	_, _, err := resolveMarkingPageReferences(
		context.Background(), fixture.queries, 1, "alice", 100, []int{1},
		func() ([]string, []string, error) {
			renderCalls++
			return nil, nil, nil
		},
	)
	if err == nil {
		t.Fatal("corrupt durable reference was accepted")
	}
	if renderCalls != 0 {
		t.Fatalf("corrupt durable reference triggered %d legacy render calls", renderCalls)
	}
}

func TestResolveMarkingPageReferencesUsesExplicitLegacyNullFallback(t *testing.T) {
	fixture := newPageReferenceResolverFixture(t)
	legacyPage := filepath.Join(t.TempDir(), "legacy-page.png")
	if err := os.WriteFile(legacyPage, fixture.pngBytes(t, 31, 41), 0o600); err != nil {
		t.Fatal(err)
	}
	legacyTypst := filepath.Join(t.TempDir(), "legacy.typ")
	renderCalls := 0
	references, cleanup, err := resolveMarkingPageReferences(
		context.Background(), fixture.queries, 1, "alice", 100, []int{1},
		func() ([]string, []string, error) {
			renderCalls++
			return []string{legacyPage}, []string{legacyTypst, legacyPage}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	legacyAbsolute, err := filepath.Abs(legacyPage)
	if err != nil {
		t.Fatal(err)
	}
	if renderCalls != 1 || len(references) != 1 || references[0] != legacyAbsolute {
		t.Fatalf("calls=%d references=%v", renderCalls, references)
	}
	if len(cleanup) != 2 || cleanup[0] != legacyTypst || cleanup[1] != legacyPage {
		t.Fatalf("cleanup=%v", cleanup)
	}
}

func TestResolveMarkingPageReferencesRejectsWrongOwnerWithoutFallback(t *testing.T) {
	fixture := newPageReferenceResolverFixture(t)
	fixture.addPNG(t, 1, 31, 41)
	renderCalls := 0
	_, _, err := resolveMarkingPageReferences(
		context.Background(), fixture.queries, 2, "bob", 100, []int{1},
		func() ([]string, []string, error) {
			renderCalls++
			return nil, nil, nil
		},
	)
	if err == nil || renderCalls != 0 {
		t.Fatalf("wrong owner err=%v legacy calls=%d", err, renderCalls)
	}
}
