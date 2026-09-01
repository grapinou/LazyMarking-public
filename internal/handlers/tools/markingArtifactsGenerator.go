package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/markingscoring"
)

type defaultMarkingArtifactsGenerator struct{}

func (defaultMarkingArtifactsGenerator) Generate(ctx context.Context, queries *db.Queries, input MarkingArtifactsGenerationInput) (MarkingArtifactsGenerationOutput, error) {
	copies, err := queries.ListMarkingArtifactCopyResults(ctx, db.ListMarkingArtifactCopyResultsParams{MarkingJobID: input.MarkingJobID, UserID: input.UserID})
	if err != nil {
		return MarkingArtifactsGenerationOutput{}, err
	}
	markExams := make([]config.MarkExam, 0, len(copies))
	notMarked := make([]config.MarkExam, 0)
	studentPDFs := make([]string, 0)
	for _, copyResult := range copies {
		var qcm config.QCM
		if err := json.Unmarshal([]byte(copyResult.SnapshotContent), &qcm); err != nil {
			return MarkingArtifactsGenerationOutput{}, fmt.Errorf("decode student exam snapshot: %w", err)
		}
		identity := config.MarkExam{StudentExamID: copyResult.StudentExamID, ExamName: qcm.Name, FirstName: qcm.Student.FirstName, LastName: qcm.Student.LastName, ClassName: qcm.Student.ClassCodes.Name}
		if copyResult.Outcome != "corrected" {
			notMarked = append(notMarked, identity)
			continue
		}
		markExam, pdfPath, err := regenerateCorrectedCopy(ctx, queries, input, copyResult, qcm)
		if err != nil {
			return MarkingArtifactsGenerationOutput{}, err
		}
		markExams = append(markExams, markExam)
		studentPDFs = append(studentPDFs, pdfPath)
	}
	if len(markExams) == 0 {
		return MarkingArtifactsGenerationOutput{}, fmt.Errorf("no corrected copies")
	}

	contentTypst, ok := TypstBuildContent(input.StagingDir, markExams, studentPDFs)
	if !ok {
		return MarkingArtifactsGenerationOutput{}, fmt.Errorf("build corrected PDF contents")
	}
	contentPDF, ok := CompileTypst(contentTypst)
	if !ok {
		return MarkingArtifactsGenerationOutput{}, fmt.Errorf("compile corrected PDF contents")
	}
	correctedPDF := filepath.Join(input.StagingDir, "corrected.pdf")
	if err := MergePdf(append([]string{contentPDF}, studentPDFs...), correctedPDF); err != nil {
		return MarkingArtifactsGenerationOutput{}, fmt.Errorf("merge corrected PDF: %w", err)
	}

	globalSkills, globalThemeSkills := AgregateThemeSkill(markExams)
	mean, stdDev, median := ComputeStatMarking(markExams)
	markTableTypst, ok := TypstBuildMarkTable(input.StagingDir, markExams, mean, stdDev, median, globalSkills, globalThemeSkills, nil, notMarked)
	if !ok {
		return MarkingArtifactsGenerationOutput{}, fmt.Errorf("build mark table")
	}
	markTablePDF, ok := CompileTypst(markTableTypst)
	if !ok {
		return MarkingArtifactsGenerationOutput{}, fmt.Errorf("compile mark table")
	}
	return MarkingArtifactsGenerationOutput{CorrectedPDF: correctedPDF, MarkTablePDF: markTablePDF}, nil
}

func regenerateCorrectedCopy(ctx context.Context, queries *db.Queries, input MarkingArtifactsGenerationInput, copyResult db.ListMarkingArtifactCopyResultsRow, qcm config.QCM) (config.MarkExam, string, error) {
	rows, err := queries.ListEffectiveMarkingAnswersForArtifacts(ctx, db.ListEffectiveMarkingAnswersForArtifactsParams{CopyResultID: copyResult.ID, MarkingJobID: input.MarkingJobID, UserID: input.UserID})
	if err != nil {
		return config.MarkExam{}, "", err
	}
	questionMarks := make([]config.QuestionMark, len(qcm.Questions))
	effectiveAnswers := make([][]int, len(qcm.Questions))
	rowIndex := 0
	for questionIndex, question := range qcm.Questions {
		effectiveAnswers[questionIndex] = make([]int, len(question.Answers))
		var persisted *db.ListEffectiveMarkingAnswersForArtifactsRow
		for answerIndex := range question.Answers {
			if rowIndex >= len(rows) || rows[rowIndex].QuestionIndex != int64(questionIndex) || rows[rowIndex].AnswerIndex != int64(answerIndex) {
				return config.MarkExam{}, "", fmt.Errorf("incomplete effective detections")
			}
			row := rows[rowIndex]
			if persisted == nil {
				persisted = &row
			}
			effectiveAnswers[questionIndex][answerIndex] = int(row.EffectiveState)
			rowIndex++
		}
		if persisted == nil {
			return config.MarkExam{}, "", fmt.Errorf("question without detections")
		}
		expected := make([]int, len(question.Answers))
		for i, answer := range question.Answers {
			expected[i] = int(answer.State)
		}
		computed := markingscoring.ScoreQuestion(expected, effectiveAnswers[questionIndex], persisted.TotalPoints)
		if computed.Score*2 != float64(persisted.ScoreHalfUnits) || questionStateName(computed.State) != persisted.QuestionState {
			return config.MarkExam{}, "", fmt.Errorf("persisted question result differs from effective scoring")
		}
		questionMarks[questionIndex] = computed
	}
	if rowIndex != len(rows) {
		return config.MarkExam{}, "", fmt.Errorf("unexpected effective detections")
	}
	score, total := CountingTotalPoint(questionMarks)
	if !copyResult.ScoreHalfUnits.Valid || score*2 != float64(copyResult.ScoreHalfUnits.Int64) || !copyResult.TotalPoints.Valid || int64(total) != copyResult.TotalPoints.Int64 {
		return config.MarkExam{}, "", fmt.Errorf("persisted copy result differs from effective questions")
	}

	pagePDFs := make([]string, 0, copyResult.ExpectedPages)
	questionOffset := 0
	for page := int64(1); page <= copyResult.ExpectedPages; page++ {
		pageJSON, err := queries.GetPageContent(ctx, db.GetPageContentParams{StudentExamID: copyResult.StudentExamID, Page: page, UserID: input.UserID})
		if err != nil {
			return config.MarkExam{}, "", fmt.Errorf("load page snapshot: %w", err)
		}
		var pageContent config.PageContent
		if err := json.Unmarshal([]byte(pageJSON), &pageContent); err != nil {
			return config.MarkExam{}, "", fmt.Errorf("decode page snapshot: %w", err)
		}
		resolved, err := ResolveMarkingAlignedPage(ctx, queries, input.UserID, input.Username, input.MarkingJobID, copyResult.ID, copyResult.StudentExamID, page)
		if err != nil {
			return config.MarkExam{}, "", err
		}
		pageName := fmt.Sprintf("student-exam-%d-page-%d.png", copyResult.StudentExamID, page)
		pagePath := filepath.Join(input.StagingDir, pageName)
		if err := copyArtifactInput(resolved.Path, pagePath); err != nil {
			return config.MarkExam{}, "", err
		}
		end := questionOffset + len(pageContent.Questions)
		if end > len(questionMarks) {
			return config.MarkExam{}, "", fmt.Errorf("page question snapshot overflow")
		}
		expectedAnswers := make([]int, 0, len(pageContent.Answers))
		for i := questionOffset; i < end; i++ {
			for _, answer := range qcm.Questions[i].Answers {
				expectedAnswers = append(expectedAnswers, int(answer.State))
			}
		}
		if len(expectedAnswers) != len(pageContent.Answers) {
			return config.MarkExam{}, "", fmt.Errorf("page answer snapshot mismatch")
		}
		DrawMarking(input.StagingDir, pageName, questionMarks[questionOffset:end], pageContent.Questions, expectedAnswers, pageContent.Answers)
		pdfName, err := ConvertPngTopdf(input.StagingDir, pageName)
		if err != nil {
			return config.MarkExam{}, "", err
		}
		pagePDFs = append(pagePDFs, filepath.Join(input.StagingDir, pdfName))
		questionOffset = end
	}
	if questionOffset != len(questionMarks) {
		return config.MarkExam{}, "", fmt.Errorf("page snapshots do not cover questions")
	}
	studentPDF := filepath.Join(input.StagingDir, fmt.Sprintf("student-exam-%d.pdf", copyResult.StudentExamID))
	if err := MergePdf(pagePDFs, studentPDF); err != nil {
		return config.MarkExam{}, "", err
	}
	skill, themeSkill := GetThemeSkill(qcm, questionMarks)
	return config.MarkExam{StudentExamID: copyResult.StudentExamID, Status: true, ExamName: qcm.Name, FirstName: qcm.Student.FirstName, LastName: qcm.Student.LastName, ClassName: qcm.Student.ClassCodes.Name, Pages: int(copyResult.ExpectedPages), Score: score, Total: total, Skill: skill, ThemeSkill: themeSkill}, studentPDF, nil
}

func questionStateName(state config.QuestionState) string {
	switch state {
	case config.Correct:
		return "correct"
	case config.Partial:
		return "partial"
	default:
		return "incorrect"
	}
}

func copyArtifactInput(source, destination string) error {
	info, err := os.Lstat(source)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return ErrMarkingArtifactsUnavailable
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
