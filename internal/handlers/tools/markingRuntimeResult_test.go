package tools

import (
	"math"
	"reflect"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TestAnswerDetectionFromMeanPreservesThresholdContract(t *testing.T) {
	for _, tc := range []struct {
		mean  float64
		state int
	}{
		{mean: 0, state: 1},
		{mean: 143.25, state: 1},
		{mean: 149.999, state: 1},
		{mean: 150, state: 0},
		{mean: 200.5, state: 0},
		{mean: 255, state: 0},
	} {
		detection, err := answerDetectionFromMean(tc.mean)
		if err != nil {
			t.Fatalf("mean %v: %v", tc.mean, err)
		}
		if detection.State != tc.state || detection.MeanGray != tc.mean {
			t.Fatalf("mean %v detection=%+v, want state=%d and unchanged mean", tc.mean, detection, tc.state)
		}
	}

	for _, mean := range []float64{-0.01, 255.01, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := answerDetectionFromMean(mean); err == nil {
			t.Fatalf("invalid mean %v was accepted", mean)
		}
	}
}

func TestDetailedRuntimeResultPreservesMarksAndRepositoryContract(t *testing.T) {
	qcm := config.QCM{Questions: []config.Question{
		runtimeQuestion(2, 7, "Skill", 11, "Theme", 1, 0),
		runtimeQuestion(1, 7, "Skill", 11, "Theme", 1, 1),
		runtimeQuestion(3, 8, "Other skill", 12, "Other theme", 1, 1),
	}}
	detections := []config.AnswerDetection{
		{State: 1, MeanGray: 143.25},
		{State: 0, MeanGray: 201.75},
		{State: 1, MeanGray: 120},
		{State: 0, MeanGray: 150},
		{State: 1, MeanGray: 80},
		{State: 0, MeanGray: 220},
	}
	states := answerDetectionStates(detections)
	marks := CountingPoints(qcm, states)
	wantMarks := []config.QuestionMark{
		{Score: 2, Total: 2, State: config.Correct},
		{Score: 0.5, Total: 1, State: config.Partial},
		{Score: 1.5, Total: 3, State: config.Partial},
	}
	if !reflect.DeepEqual(marks, wantMarks) {
		t.Fatalf("marks=%+v, want %+v", marks, wantMarks)
	}

	result, err := BuildMarkingCopyResult(42, 2, 2, qcm, marks, detections)
	if err != nil {
		t.Fatal(err)
	}
	legacyScore, legacyTotal := CountingTotalPoint(marks)
	if result.Score != legacyScore || result.Total != legacyTotal || result.Score != 4 || result.Total != 6 {
		t.Fatalf("result score=%v/%d, legacy=%v/%d", result.Score, result.Total, legacyScore, legacyTotal)
	}
	if len(result.Questions) != 3 || result.Questions[0].QuestionIndex != 0 || result.Questions[1].QuestionIndex != 1 {
		t.Fatalf("question indices=%+v", result.Questions)
	}
	if !reflect.DeepEqual(result.Questions[0].Mark, marks[0]) || !reflect.DeepEqual(result.Questions[1].Mark, marks[1]) {
		t.Fatal("detailed question marks differ from CountingPoints output")
	}

	legacySkill, legacyThemeSkill := GetThemeSkill(qcm, marks)
	markExam := config.MarkExam{
		StudentExamID:  42,
		Status:         true,
		Pages:          2,
		Score:          result.Score,
		Total:          result.Total,
		Skill:          legacySkill,
		ThemeSkill:     legacyThemeSkill,
		DetailedResult: &result,
	}
	if markExam.Score != legacyScore || markExam.Total != legacyTotal || !reflect.DeepEqual(markExam.Skill, legacySkill) || !reflect.DeepEqual(markExam.ThemeSkill, legacyThemeSkill) {
		t.Fatal("MarkExam compatibility fields changed")
	}

	input, err := MarkingCopyResultToPersistedInput(1, 99, result)
	if err != nil {
		t.Fatal(err)
	}
	if input.UserID != 1 || input.MarkingJobID != 99 || input.StudentExamID != 42 || input.ScoreHalfUnits != 8 || input.TotalPoints != 6 {
		t.Fatalf("repository input=%+v", input)
	}
	if input.Questions[0].QuestionIndex != 0 || input.Questions[0].Answers[0].AnswerIndex != 0 || input.Questions[0].Answers[1].AnswerIndex != 1 || input.Questions[1].QuestionIndex != 1 || input.Questions[1].Answers[0].AnswerIndex != 0 {
		t.Fatalf("repository indices=%+v", input.Questions)
	}
	if input.Questions[0].Answers[0].MeanGray != 143.25 {
		t.Fatalf("mean gray=%v, want 143.25", input.Questions[0].Answers[0].MeanGray)
	}
	if input.Questions[1].ScoreHalfUnits != 1 || input.Questions[2].ScoreHalfUnits != 3 {
		t.Fatalf("partial half units=%d,%d, want 1,3", input.Questions[1].ScoreHalfUnits, input.Questions[2].ScoreHalfUnits)
	}
}

func TestScoreHalfUnits(t *testing.T) {
	for score, want := range map[float64]int64{0: 0, 0.5: 1, 1: 2, 1.5: 3, 12.5: 25} {
		got, err := ScoreHalfUnits(score)
		if err != nil || got != want {
			t.Fatalf("ScoreHalfUnits(%v)=(%d,%v), want (%d,nil)", score, got, err, want)
		}
	}
	for _, score := range []float64{0.25, 1.25, -0.5, math.NaN(), math.Inf(1)} {
		if _, err := ScoreHalfUnits(score); err == nil {
			t.Fatalf("ScoreHalfUnits(%v) succeeded", score)
		}
	}
}

func TestMarkingQuestionStateName(t *testing.T) {
	for state, want := range map[config.QuestionState]string{
		config.Incorrect: "incorrect",
		config.Partial:   "partial",
		config.Correct:   "correct",
	} {
		got, err := MarkingQuestionStateName(state)
		if err != nil || got != want {
			t.Fatalf("state %d=(%q,%v), want %q", state, got, err, want)
		}
	}
	if _, err := MarkingQuestionStateName(config.QuestionState(99)); err == nil {
		t.Fatal("unknown question state succeeded")
	}
}

func TestBuildMarkingCopyResultRejectsMisalignment(t *testing.T) {
	qcm := config.QCM{Questions: []config.Question{runtimeQuestion(1, 1, "skill", 1, "theme", 1, 0)}}
	marks := []config.QuestionMark{{Score: 1, Total: 1, State: config.Correct}}
	if _, err := BuildMarkingCopyResult(1, 1, 1, qcm, marks, []config.AnswerDetection{{State: 1, MeanGray: 100}}); err == nil {
		t.Fatal("missing answer detection succeeded")
	}
	if _, err := BuildMarkingCopyResult(1, 1, 1, qcm, marks, []config.AnswerDetection{{State: 1, MeanGray: 100}, {State: 1, MeanGray: 100}, {State: 1, MeanGray: 100}}); err == nil {
		t.Fatal("extra answer detection succeeded")
	}
}

func runtimeQuestion(points, skillID int64, skillName string, themeID int64, themeName string, expected ...int64) config.Question {
	question := testQuestion(points, expected...)
	question.Tags.Skill = config.Skill{ID: skillID, Name: skillName}
	question.Tags.Theme = config.Theme{ID: themeID, Name: themeName}
	return question
}
