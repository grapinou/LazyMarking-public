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
	)
	if err == nil || !strings.Contains(err.Error(), "exam worker panic") {
		t.Fatalf("error=%v, want recovered worker panic", err)
	}
	if len(marked) != 0 || len(unmarked) != 0 {
		t.Fatalf("marked=%d unmarked=%d, want no results", len(marked), len(unmarked))
	}
}
