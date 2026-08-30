package data

import "testing"

func TestExamGenerationProgressPercentage(t *testing.T) {
	tests := []struct {
		name      string
		processed int64
		total     int64
		want      int64
	}{
		{name: "zero total", processed: 0, total: 0, want: 0},
		{name: "incoherent zero total", processed: 3, total: 0, want: 0},
		{name: "negative values", processed: -1, total: 4, want: 0},
		{name: "in progress", processed: 3, total: 8, want: 37},
		{name: "complete", processed: 8, total: 8, want: 100},
		{name: "over complete", processed: 9, total: 8, want: 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			progress := ExamGenerationProgress{ProcessedStudents: tc.processed, TotalStudents: tc.total}
			if got := progress.Percentage(); got != tc.want {
				t.Fatalf("Percentage()=%d, want %d", got, tc.want)
			}
		})
	}
}
