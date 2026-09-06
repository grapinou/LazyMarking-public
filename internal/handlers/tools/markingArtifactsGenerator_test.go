package tools

import (
	"reflect"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func pageContentWithCounts(questions, answers int) config.PageContent {
	return config.PageContent{
		Questions: make([]config.CircleValidated, questions),
		Answers:   make([]config.CircleValidated, answers),
	}
}

func TestMarkingSnapshotCursorPagination(t *testing.T) {
	marks := []config.QuestionMark{{Score: 1}, {Score: 2}}
	expected := []int{1, 0, 0, 1, 1, 0}
	effective := []int{0, 1, 0, 1, 0, 1}

	tests := []struct {
		name  string
		pages []config.PageContent
	}{
		{
			name:  "questions and answers stay on their page",
			pages: []config.PageContent{pageContentWithCounts(1, 4), pageContentWithCounts(1, 2)},
		},
		{
			name:  "question answers cross a page boundary",
			pages: []config.PageContent{pageContentWithCounts(1, 2), pageContentWithCounts(1, 4)},
		},
		{
			name:  "page has answers but no question",
			pages: []config.PageContent{pageContentWithCounts(1, 2), pageContentWithCounts(0, 2), pageContentWithCounts(1, 2)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := markingSnapshotCursor{questionMarks: marks, expectedAnswers: expected, effectiveAnswers: effective}
			var gotMarks []config.QuestionMark
			var gotAnswers []int
			var gotEffective []int
			for _, page := range tt.pages {
				pageMarks, pageAnswers, pageEffective, err := cursor.consume(page)
				if err != nil {
					t.Fatalf("consume page: %v", err)
				}
				gotMarks = append(gotMarks, pageMarks...)
				gotAnswers = append(gotAnswers, pageAnswers...)
				gotEffective = append(gotEffective, pageEffective...)
			}
			if err := cursor.validateComplete(); err != nil {
				t.Fatalf("validate complete: %v", err)
			}
			if !reflect.DeepEqual(gotMarks, marks) {
				t.Errorf("marks = %+v, want %+v", gotMarks, marks)
			}
			if !reflect.DeepEqual(gotAnswers, expected) {
				t.Errorf("answers = %v, want %v", gotAnswers, expected)
			}
			if !reflect.DeepEqual(gotEffective, effective) {
				t.Errorf("effective answers = %v, want %v", gotEffective, effective)
			}
		})
	}
}

func TestMarkingSnapshotCursorRejectsInconsistentSnapshots(t *testing.T) {
	tests := []struct {
		name string
		run  func(*markingSnapshotCursor) error
		want string
	}{
		{
			name: "too many questions",
			run: func(cursor *markingSnapshotCursor) error {
				_, _, _, err := cursor.consume(pageContentWithCounts(3, 0))
				return err
			},
			want: "page question snapshot overflow",
		},
		{
			name: "too many answers",
			run: func(cursor *markingSnapshotCursor) error {
				_, _, _, err := cursor.consume(pageContentWithCounts(0, 7))
				return err
			},
			want: "page answer snapshot overflow",
		},
		{
			name: "questions remain",
			run: func(cursor *markingSnapshotCursor) error {
				_, _, _, _ = cursor.consume(pageContentWithCounts(1, 6))
				return cursor.validateComplete()
			},
			want: "page snapshots do not cover questions",
		},
		{
			name: "answers remain",
			run: func(cursor *markingSnapshotCursor) error {
				_, _, _, _ = cursor.consume(pageContentWithCounts(2, 5))
				return cursor.validateComplete()
			},
			want: "page snapshots do not cover answers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := markingSnapshotCursor{
				questionMarks:    make([]config.QuestionMark, 2),
				expectedAnswers:  make([]int, 6),
				effectiveAnswers: make([]int, 6),
			}
			if err := tt.run(&cursor); err == nil || err.Error() != tt.want {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
