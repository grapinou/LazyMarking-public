package tools

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

type fakeArtifactRevisionQueries struct {
	target       db.GetMarkingArtifactsRegenerationTargetRow
	getErr       error
	advanceRows  int64
	advanceErr   error
	advanceCalls int
}

func (f *fakeArtifactRevisionQueries) GetMarkingArtifactsRegenerationTarget(context.Context, db.GetMarkingArtifactsRegenerationTargetParams) (db.GetMarkingArtifactsRegenerationTargetRow, error) {
	return f.target, f.getErr
}
func (f *fakeArtifactRevisionQueries) AdvanceMarkingArtifactsRevision(context.Context, db.AdvanceMarkingArtifactsRevisionParams) (int64, error) {
	f.advanceCalls++
	return f.advanceRows, f.advanceErr
}

type fakeArtifactGenerator struct {
	calls            int
	err              error
	corrected, table []byte
}

func (f *fakeArtifactGenerator) Generate(_ context.Context, _ *db.Queries, input MarkingArtifactsGenerationInput) (MarkingArtifactsGenerationOutput, error) {
	f.calls++
	if f.err != nil {
		return MarkingArtifactsGenerationOutput{}, f.err
	}
	corrected := filepath.Join(input.StagingDir, "corrected.pdf")
	table := filepath.Join(input.StagingDir, "mark-table.pdf")
	if err := os.WriteFile(corrected, f.corrected, 0o600); err != nil {
		return MarkingArtifactsGenerationOutput{}, err
	}
	if err := os.WriteFile(table, f.table, 0o600); err != nil {
		return MarkingArtifactsGenerationOutput{}, err
	}
	return MarkingArtifactsGenerationOutput{CorrectedPDF: corrected, MarkTablePDF: table}, nil
}

func TestRegenerateMarkingArtifactsCurrentIsNoOp(t *testing.T) {
	queries := &fakeArtifactRevisionQueries{target: db.GetMarkingArtifactsRegenerationTargetRow{ReviewRevision: 2, ArtifactsRevision: 2}}
	generator := &fakeArtifactGenerator{}
	result, err := regenerateMarkingArtifacts(t.Context(), &db.Queries{}, queries, generator, 1, "alice", 7)
	if err != nil || result.Regenerated || generator.calls != 0 || queries.advanceCalls != 0 {
		t.Fatalf("result=%+v err=%v generator=%d advance=%d", result, err, generator.calls, queries.advanceCalls)
	}
}

func TestRegenerateHybridArtifactsRejectsPendingDisagreement(t *testing.T) {
	withArtifactWorkspace(t, func(workspace string) {
		revisions := currentArtifactTarget(workspace, 0, 0)
		revisions.target.ReviewPolicyVersion = sql.NullString{String: MarkingReviewPolicyVersion, Valid: true}
		revisions.target.PendingCandidates = 1
		generator := &fakeArtifactGenerator{}
		_, err := regenerateMarkingArtifacts(t.Context(), &db.Queries{}, revisions, generator, 1, "alice", 7)
		if !errors.Is(err, ErrMarkingArtifactsUnavailable) || generator.calls != 0 {
			t.Fatalf("err=%v generator calls=%d", err, generator.calls)
		}
	})
}

func TestRegenerateMarkingArtifactsPublishesPairThenRevision(t *testing.T) {
	withArtifactWorkspace(t, func(workspace string) {
		queries := currentArtifactTarget(workspace, 3, 2)
		queries.advanceRows = 1
		generator := &fakeArtifactGenerator{corrected: []byte("%PDF-new-corrected"), table: []byte("%PDF-new-table")}
		result, err := regenerateMarkingArtifacts(t.Context(), &db.Queries{}, queries, generator, 1, "alice", 7)
		if err != nil || !result.Regenerated || result.ArtifactsRevision != 3 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		assertFileBytes(t, filepath.Join(workspace, "corrected.pdf"), generator.corrected)
		assertFileBytes(t, filepath.Join(workspace, "mark-table.pdf"), generator.table)
		assertNoRegenerationTemps(t, workspace)
	})
}

func TestRegenerateMarkingArtifactsConcurrentReviewRestoresOldPair(t *testing.T) {
	withArtifactWorkspace(t, func(workspace string) {
		queries := currentArtifactTarget(workspace, 3, 2)
		generator := &fakeArtifactGenerator{corrected: []byte("%PDF-new-corrected"), table: []byte("%PDF-new-table")}
		_, err := regenerateMarkingArtifacts(t.Context(), &db.Queries{}, queries, generator, 1, "alice", 7)
		if !errors.Is(err, ErrMarkingArtifactsConflict) {
			t.Fatalf("err=%v", err)
		}
		assertFileBytes(t, filepath.Join(workspace, "corrected.pdf"), []byte("%PDF-old-corrected"))
		assertFileBytes(t, filepath.Join(workspace, "mark-table.pdf"), []byte("%PDF-old-table"))
		assertNoRegenerationTemps(t, workspace)
	})
}

func TestRegenerateMarkingArtifactsGenerationFailureLeavesCanonicalPair(t *testing.T) {
	withArtifactWorkspace(t, func(workspace string) {
		queries := currentArtifactTarget(workspace, 3, 2)
		generator := &fakeArtifactGenerator{err: errors.New("injected corrected generation failure")}
		_, err := regenerateMarkingArtifacts(t.Context(), &db.Queries{}, queries, generator, 1, "alice", 7)
		if !errors.Is(err, ErrMarkingArtifactsRegeneration) {
			t.Fatalf("err=%v", err)
		}
		if queries.advanceCalls != 0 {
			t.Fatalf("advance calls=%d", queries.advanceCalls)
		}
		assertFileBytes(t, filepath.Join(workspace, "corrected.pdf"), []byte("%PDF-old-corrected"))
		assertFileBytes(t, filepath.Join(workspace, "mark-table.pdf"), []byte("%PDF-old-table"))
		assertNoRegenerationTemps(t, workspace)
	})
}

func TestRegenerateMarkingArtifactsInvalidMarkTableLeavesCanonicalPairStale(t *testing.T) {
	withArtifactWorkspace(t, func(workspace string) {
		queries := currentArtifactTarget(workspace, 3, 2)
		generator := &fakeArtifactGenerator{corrected: []byte("%PDF-new-corrected"), table: []byte("not-a-pdf")}
		_, err := regenerateMarkingArtifacts(t.Context(), &db.Queries{}, queries, generator, 1, "alice", 7)
		if !errors.Is(err, ErrMarkingArtifactsRegeneration) {
			t.Fatalf("err=%v", err)
		}
		if queries.advanceCalls != 0 {
			t.Fatalf("advance calls=%d", queries.advanceCalls)
		}
		assertFileBytes(t, filepath.Join(workspace, "corrected.pdf"), []byte("%PDF-old-corrected"))
		assertFileBytes(t, filepath.Join(workspace, "mark-table.pdf"), []byte("%PDF-old-table"))
		assertNoRegenerationTemps(t, workspace)
	})
}

func currentArtifactTarget(workspace string, review, artifacts int64) *fakeArtifactRevisionQueries {
	return &fakeArtifactRevisionQueries{target: db.GetMarkingArtifactsRegenerationTargetRow{
		ReviewRevision: review, ArtifactsRevision: artifacts,
		AmbiguityDelta: sql.NullFloat64{Float64: 5, Valid: true},
		ExamName:       sql.NullString{String: filepath.Join(workspace, "corrected.pdf"), Valid: true},
		MarkTableName:  sql.NullString{String: filepath.Join(workspace, "mark-table.pdf"), Valid: true},
	}}
}

func withArtifactWorkspace(t *testing.T, run func(string)) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	workspace := filepath.Join("assets", "tmp", "alice", "marking-7")
	if err := os.MkdirAll(workspace, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "corrected.pdf"), []byte("%PDF-old-corrected"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "mark-table.pdf"), []byte("%PDF-old-table"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(workspace)
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s=%q want %q", path, got, want)
	}
}
func assertNoRegenerationTemps(t *testing.T, workspace string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(workspace, ".review-artifacts-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary paths remain: %v", matches)
	}
}
