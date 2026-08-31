package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestProcessExamsConcurrentlyConvertsWorkerPanicToError(t *testing.T) {
	marked, unmarked, err := ProcessExamsConcurrently(
		[]config.Exam{{StudentExamID: 1}},
		1,
		"alice",
		t.TempDir(),
		context.Background(),
		nil, // Forces the worker's first database call to panic.
		1,
		map[int64]int64{1: 1},
	)
	if err == nil || !strings.Contains(err.Error(), "exam worker panic") {
		t.Fatalf("error=%v, want recovered worker panic", err)
	}
	if len(marked) != 0 || len(unmarked) != 0 {
		t.Fatalf("marked=%d unmarked=%d, want no results", len(marked), len(unmarked))
	}
}

func TestTerminalOutcomeForMarkingError(t *testing.T) {
	for _, tc := range []struct {
		name     string
		pages    []config.Page
		outcome  string
		detected int64
	}{
		{name: "missing page", pages: []config.Page{{Number: 1}}, outcome: "incomplete", detected: 1},
		{name: "duplicate page", pages: []config.Page{{Number: 1}, {Number: 1}}, outcome: "incomplete", detected: 1},
		{name: "technical error with complete pages", pages: []config.Page{{Number: 1}, {Number: 2}}, outcome: "error", detected: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outcome, detected := terminalOutcomeForMarkingError(config.Exam{Pages: tc.pages}, 2)
			if outcome != tc.outcome || detected != tc.detected {
				t.Fatalf("got (%q,%d), want (%q,%d)", outcome, detected, tc.outcome, tc.detected)
			}
		})
	}
}
