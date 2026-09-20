package marking

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func pedagogicalQuestions() []config.Question {
	var questions []config.Question
	for i, content := range []string{"Calculer une somme", "Comparer deux fractions", "Résoudre une équation", "Lire un graphique", "Justifier une propriété"} {
		questions = append(questions, config.Question{
			Tags:    config.Tags{MainQuestionID: int64(11 + i), Point: config.Point{PointValue: 4}, Skill: config.Skill{ID: 1, Name: "Calculer"}, Theme: config.Theme{ID: 1, Name: "Nombres"}},
			Content: content, Answers: []config.Answer{{Content: "Oui", State: 1}, {Content: "Aussi", State: 1}},
		})
	}
	return questions
}

func pedagogicalResult(t *testing.T, questions []config.Question, scores []int64) db.ListCurrentExamResultsForGenerationRow {
	t.Helper()
	snapshot, err := json.Marshal(config.QCM{Questions: questions})
	if err != nil {
		t.Fatal(err)
	}
	var persisted []persistedPedagogicalQuestion
	var half, total int64
	for i, question := range questions {
		state := "incorrect"
		if scores[i] == 2*question.Tags.Point.PointValue {
			state = "correct"
		} else if scores[i] == question.Tags.Point.PointValue {
			state = "partial"
		}
		persisted = append(persisted, persistedPedagogicalQuestion{Index: int64(i), State: state, Half: scores[i], Total: question.Tags.Point.PointValue})
		half += scores[i]
		total += question.Tags.Point.PointValue
	}
	details, err := json.Marshal(persisted)
	if err != nil {
		t.Fatal(err)
	}
	return db.ListCurrentExamResultsForGenerationRow{Outcome: "corrected", ScoreHalfUnits: sql.NullInt64{Int64: half, Valid: true}, TotalPoints: sql.NullInt64{Int64: total, Valid: true}, SnapshotContent: string(snapshot), QuestionResults: string(details)}
}

func seedPedagogicalCopy(t *testing.T, f reviewPageFixture, copyID, jobID, studentID int64, row db.ListCurrentExamResultsForGenerationRow) {
	t.Helper()
	if _, err := f.conn.Exec(`INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,expected_pages,detected_pages,score_half_units,total_points) VALUES(?,1,?,?,'corrected',1,1,?,?)`, copyID, jobID, studentID, row.ScoreHalfUnits.Int64, row.TotalPoints.Int64); err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(`INSERT INTO student_exam_content(student_exam_id,user_id,content) VALUES(?,1,?)`, studentID, row.SnapshotContent); err != nil {
		t.Fatal(err)
	}
	var questions []persistedPedagogicalQuestion
	if err := json.Unmarshal([]byte(row.QuestionResults), &questions); err != nil {
		t.Fatal(err)
	}
	for _, question := range questions {
		id := copyID*10 + question.Index
		if _, err := f.conn.Exec(`INSERT INTO marking_question_results VALUES(?,?,?,?,?,?)`, id, copyID, question.Index, question.State, question.Half, question.Total); err != nil {
			t.Fatal(err)
		}
		first, second := 0, 0
		if question.State == "correct" {
			first, second = 1, 1
		} else if question.State == "partial" {
			first = 1
		}
		if _, err := f.conn.Exec(`INSERT INTO marking_answer_detections(id,question_result_id,answer_index,detected_state,mean_gray) VALUES(?,?,0,?,220),(?,?,1,?,220)`, id*10, id, first, id*10+1, id, second); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCumulativePedagogyThroughCatchUpAndReview(t *testing.T) {
	f, mux := newCumulativeFixture(t)
	if _, err := f.conn.Exec(`
		DELETE FROM marking_answer_reviews; DELETE FROM marking_answer_detections;
		DELETE FROM marking_question_results; DELETE FROM marking_copy_results; DELETE FROM student_exam_content;
		INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,expected_pages,detected_pages) VALUES(504,1,50,104,'not_seen',1,0);
	`); err != nil {
		t.Fatal(err)
	}
	questions := pedagogicalQuestions()
	seedPedagogicalCopy(t, f, 500, 50, 100, pedagogicalResult(t, questions, []int64{8, 8, 4, 0, 0})) // 10/20
	reversed := append([]config.Question(nil), questions...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	seedPedagogicalCopy(t, f, 501, 50, 101, pedagogicalResult(t, reversed, []int64{0, 4, 8, 8, 8})) // 14/20, different order
	check := func(stage string, count int, mean, questionSuccess string) {
		t.Helper()
		page, err := loadMarkingGeneration(t.Context(), f.queries, 1, 10)
		if err != nil {
			t.Fatal(err)
		}
		stats := page.Pedagogy
		if stats.IncludedCopies != count || stats.DetailedCopies != count || len(stats.ScoreGroups) != 1 || stats.ScoreGroups[0].Mean != mean {
			t.Fatalf("%s stats=%+v", stage, stats)
		}
		if len(stats.Questions) != 5 || stats.Questions[0].Success != questionSuccess || stats.Questions[0].Count != count {
			t.Fatalf("%s questions=%+v", stage, stats.Questions)
		}
		if len(stats.Skills) != 1 || len(stats.ThemeSkills) != 1 {
			t.Fatalf("skills missing: %+v", stats)
		}
		t.Run(stage+" PDF", func(t *testing.T) {
			text := cumulativePDFText(t, mux, page.PDFURL)
			normalized := strings.Join(strings.Fields(text), " ")
			if !strings.Contains(normalized, "Moyenne : "+mean+" / 20") {
				t.Fatalf("PDF mean: %s", text)
			}
			for _, want := range []string{"Réussite par question", "Calculer une somme", questionSuccess + " %", "Réussite des compétences globales", "Réussite des compétences par thème"} {
				if !strings.Contains(normalized, want) {
					t.Fatalf("PDF missing %q: %s", want, text)
				}
			}
		})
	}
	check("premier lot", 2, "12,00", "100,00")
	initial, err := loadMarkingGeneration(t.Context(), f.queries, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Pedagogy.ScoreGroups[0].Median != "12,00" || initial.Pedagogy.ScoreGroups[0].StdDev != "2,00" || initial.Pedagogy.Skills[0].Success != "60,00" || initial.Pedagogy.ThemeSkills[0].Success != "60,00" {
		t.Fatalf("historical metrics: %+v", initial.Pedagogy)
	}
	for i, want := range []string{"100,00", "100,00", "75,00", "25,00", "0,00"} {
		if initial.Pedagogy.Questions[i].Success != want {
			t.Fatalf("question %d success: %+v", i, initial.Pedagogy.Questions[i])
		}
	}
	if _, err := f.conn.Exec(`
		INSERT INTO marking_jobs(id,user_id,status,status_pdf,review_revision,artifacts_revision,exam_generated_id) VALUES(60,1,'success','success',0,0,10);
		INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,expected_pages,detected_pages) VALUES(6000,1,60,100,'not_seen',1,0),(6001,1,60,101,'incomplete',1,0);
	`); err != nil {
		t.Fatal(err)
	}
	seedPedagogicalCopy(t, f, 604, 60, 104, pedagogicalResult(t, questions, []int64{8, 8, 8, 8, 4})) // 18/20
	check("rattrapage", 3, "14,00", "100,00")
	if _, err := f.conn.Exec(`UPDATE marking_answer_detections SET mean_gray=150 WHERE id=50000`); err != nil {
		t.Fatal(err)
	}
	check("pending exclu", 2, "16,00", "100,00")
	if _, err := db.ApplyMarkingAnswerReview(t.Context(), f.queries, db.ApplyMarkingAnswerReviewInput{UserID: 1, MarkingJobID: 50, AnswerDetectionID: 50000, ReviewedState: 1, ExpectedJobReviewRevision: 1}); err != nil {
		t.Fatal(err)
	}
	check("revue confirmée", 3, "14,00", "100,00")
	if _, err := db.ApplyMarkingAnswerReview(t.Context(), f.queries, db.ApplyMarkingAnswerReviewInput{UserID: 1, MarkingJobID: 50, AnswerDetectionID: 50000, ReviewedState: 0, ExpectedJobReviewRevision: 2, ExpectedAnswerReviewRevision: 1}); err != nil {
		t.Fatal(err)
	}
	check("revue modifiée", 3, "13,33", "83,33")
}

func TestPedagogicalQuestionsDistinguishFamiliesVariantsAndShuffling(t *testing.T) {
	base := pedagogicalQuestions()[:1]
	first := pedagogicalResult(t, base, []int64{8})
	shuffled := append([]config.Question(nil), base...)
	shuffled[0].Answers = []config.Answer{base[0].Answers[1], base[0].Answers[0]}
	shuffled[0].Answers[0].Symbol = "random layout"
	second := pedagogicalResult(t, shuffled, []int64{4})
	variant := append([]config.Question(nil), base...)
	variant[0].Content = "Calculer une autre somme"
	third := pedagogicalResult(t, variant, []int64{0})
	different := append([]config.Question(nil), base...)
	different[0].Tags.MainQuestionID = 99 // same wording and skill do not establish equivalence.
	fourth := pedagogicalResult(t, different, []int64{0})
	summary := buildMarkingPedagogicalSummary([]db.ListCurrentExamResultsForGenerationRow{first, second, third, fourth})
	if len(summary.Questions) != 3 {
		t.Fatalf("variants merged incorrectly: %+v", summary.Questions)
	}
	var combined bool
	for _, question := range summary.Questions {
		if question.Count == 2 {
			combined = true
			if question.Success != "75,00" || question.Correct != 1 {
				t.Fatalf("partial credit: %+v", question)
			}
		}
	}
	if !combined {
		t.Fatal("answer shuffling created artificial variants")
	}
	if !reflect.DeepEqual(summary, buildMarkingPedagogicalSummary([]db.ListCurrentExamResultsForGenerationRow{fourth, third, second, first})) {
		t.Fatal("unstable statistics order")
	}
	// Legacy snapshots without family identity only group identical contents.
	base[0].Tags.MainQuestionID = 0
	variant[0].Tags.MainQuestionID = 0
	legacy := buildMarkingPedagogicalSummary([]db.ListCurrentExamResultsForGenerationRow{pedagogicalResult(t, base, []int64{8}), pedagogicalResult(t, variant, []int64{0})})
	if len(legacy.Questions) != 2 {
		t.Fatal("legacy unrelated contents merged")
	}
}

func TestPedagogicalSummaryExclusionsAndHistoricalCoverage(t *testing.T) {
	valid := pedagogicalResult(t, pedagogicalQuestions()[:1], []int64{4})
	var rows []db.ListCurrentExamResultsForGenerationRow
	for _, outcome := range []string{"not_seen", "incomplete", "error"} {
		row := valid
		row.Outcome = outcome
		rows = append(rows, row)
	}
	pending := valid
	pending.PendingReviews = 1
	rows = append(rows, pending)
	zero := valid
	zero.TotalPoints.Int64 = 0
	rows = append(rows, zero)
	invalid := valid
	invalid.ScoreHalfUnits.Valid = false
	rows = append(rows, invalid)
	stats := buildMarkingPedagogicalSummary(rows)
	if stats.IncludedCopies != 0 || stats.ExcludedCopies != 6 || len(stats.ScoreGroups) != 0 || len(stats.Questions) != 0 {
		t.Fatalf("unfinished copies counted: %+v", stats)
	}
	missing := valid
	missing.SnapshotContent = ""
	incoherent := valid
	incoherent.QuestionResults = `[{"question_index":0,"state":"correct","score_half_units":8,"total_points":4}]`
	stats = buildMarkingPedagogicalSummary([]db.ListCurrentExamResultsForGenerationRow{valid, missing, incoherent})
	if stats.IncludedCopies != 3 || stats.DetailedCopies != 1 || stats.ScoreGroups[0].Mean != "2,00" || stats.Questions[0].Count != 1 {
		t.Fatalf("legacy coverage: %+v", stats)
	}
	other := pedagogicalQuestions()[:1]
	other[0].Tags.Point.PointValue = 10
	stats = buildMarkingPedagogicalSummary([]db.ListCurrentExamResultsForGenerationRow{valid, pedagogicalResult(t, other, []int64{20})})
	if len(stats.ScoreGroups) != 2 || stats.ScoreGroups[0].Mean != "2,00" || stats.ScoreGroups[1].Mean != "10,00" {
		t.Fatalf("different grading scales mixed: %+v", stats)
	}
}
