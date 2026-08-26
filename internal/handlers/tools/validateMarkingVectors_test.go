package tools

import (
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestValidateMarkingVectors(t *testing.T) {
	tests := []struct {
		name         string
		qcm          config.QCM
		homoPages    []config.HomoPage
		answersState []int
		wantErr      bool
	}{
		{
			name:         "coherent structure",
			qcm:          markingVectorQCM([]int64{1, 0}),
			homoPages:    markingVectorPages([2]int{1, 2}),
			answersState: []int{1, 0},
		},
		{
			name:         "recognized answers too short",
			qcm:          markingVectorQCM([]int64{1, 0}),
			homoPages:    markingVectorPages([2]int{1, 2}),
			answersState: []int{1},
			wantErr:      true,
		},
		{
			name:         "recognized answers too long",
			qcm:          markingVectorQCM([]int64{1, 0}),
			homoPages:    markingVectorPages([2]int{1, 2}),
			answersState: []int{1, 0, 1},
			wantErr:      true,
		},
		{
			name:         "page answer count differs",
			qcm:          markingVectorQCM([]int64{1, 0}),
			homoPages:    markingVectorPages([2]int{1, 1}),
			answersState: []int{1, 0},
			wantErr:      true,
		},
		{
			name:         "page question count differs",
			qcm:          markingVectorQCM([]int64{1, 0}),
			homoPages:    markingVectorPages([2]int{2, 2}),
			answersState: []int{1, 0},
			wantErr:      true,
		},
		{
			name:         "recognized state is not binary",
			qcm:          markingVectorQCM([]int64{1, 0}),
			homoPages:    markingVectorPages([2]int{1, 2}),
			answersState: []int{2, 0},
			wantErr:      true,
		},
		{
			name:         "expected state is not binary",
			qcm:          markingVectorQCM([]int64{2, 0}),
			homoPages:    markingVectorPages([2]int{1, 2}),
			answersState: []int{1, 0},
			wantErr:      true,
		},
		{
			name: "multiple coherent pages",
			qcm: markingVectorQCM(
				[]int64{1},
				[]int64{0, 1},
			),
			homoPages: markingVectorPages(
				[2]int{1, 1},
				[2]int{1, 2},
			),
			answersState: []int{1, 0, 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateMarkingVectors(test.qcm, test.homoPages, test.answersState)
			if test.wantErr && err == nil {
				t.Fatal("validateMarkingVectors returned nil, want error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("validateMarkingVectors returned error: %v", err)
			}
		})
	}
}

func markingVectorQCM(answerStates ...[]int64) config.QCM {
	questions := make([]config.Question, len(answerStates))
	for questionIndex, states := range answerStates {
		questions[questionIndex].Answers = make([]config.Answer, len(states))
		for answerIndex, state := range states {
			questions[questionIndex].Answers[answerIndex].State = state
		}
	}
	return config.QCM{Questions: questions}
}

func markingVectorPages(counts ...[2]int) []config.HomoPage {
	pages := make([]config.HomoPage, len(counts))
	for index, count := range counts {
		pages[index].Content.Questions = make([]config.CircleValidated, count[0])
		pages[index].Content.Answers = make([]config.CircleValidated, count[1])
	}
	return pages
}
