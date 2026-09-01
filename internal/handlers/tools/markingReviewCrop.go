package tools

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

var ErrMarkingReviewCandidateUnavailable = errors.New("marking review candidate unavailable")

const markingReviewCropMinimumHalfSide = 30

type MarkingReviewCandidateROI struct {
	AlignedPagePath string
	PageExam        int64
	CenterX         int
	CenterY         int
	Radius          int
	ImageWidth      int
	ImageHeight     int
}

type markingReviewPageSnapshot struct {
	Page    int64
	Content config.PageContent
}

// ResolveMarkingReviewCandidateROI maps a persisted detection to its historical
// page geometry, then resolves the immutable aligned PNG for that exact copy.
func ResolveMarkingReviewCandidateROI(ctx context.Context, queries *db.Queries, userID int64, username string, markingJobID, answerDetectionID int64) (MarkingReviewCandidateROI, error) {
	target, err := queries.GetMarkingAnswerReviewTarget(ctx, db.GetMarkingAnswerReviewTargetParams{
		MarkingJobID: markingJobID, UserID: userID, AnswerDetectionID: answerDetectionID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return MarkingReviewCandidateROI{}, ErrMarkingReviewCandidateUnavailable
	}
	if err != nil {
		return MarkingReviewCandidateROI{}, fmt.Errorf("load review detection target: %w", err)
	}
	var snapshot config.QCM
	if err := json.Unmarshal([]byte(target.SnapshotContent), &snapshot); err != nil {
		return MarkingReviewCandidateROI{}, ErrMarkingReviewCandidateUnavailable
	}
	rows, err := queries.ListMarkingReviewPageSnapshots(ctx, db.ListMarkingReviewPageSnapshotsParams{
		MarkingJobID: markingJobID, UserID: userID, AnswerDetectionID: answerDetectionID,
	})
	if err != nil {
		return MarkingReviewCandidateROI{}, fmt.Errorf("load review page snapshots: %w", err)
	}
	pages := make([]markingReviewPageSnapshot, 0, len(rows))
	for _, row := range rows {
		var content config.PageContent
		if err := json.Unmarshal([]byte(row.Content), &content); err != nil {
			return MarkingReviewCandidateROI{}, ErrMarkingReviewCandidateUnavailable
		}
		pages = append(pages, markingReviewPageSnapshot{Page: row.Page, Content: content})
	}
	pageExam, circle, err := resolveHistoricalAnswerROI(snapshot, pages, target.QuestionIndex, target.AnswerIndex)
	if err != nil {
		return MarkingReviewCandidateROI{}, ErrMarkingReviewCandidateUnavailable
	}
	aligned, err := ResolveMarkingAlignedPage(ctx, queries, userID, username, markingJobID, target.CopyResultID, target.StudentExamID, pageExam)
	if err != nil {
		if errors.Is(err, ErrMarkingAlignedPageUnavailable) || errors.Is(err, ErrMarkingAlignedPageCorrupt) || errors.Is(err, ErrMarkingAlignedPageUnsafe) {
			return MarkingReviewCandidateROI{}, ErrMarkingReviewCandidateUnavailable
		}
		return MarkingReviewCandidateROI{}, fmt.Errorf("resolve review aligned page: %w", err)
	}
	if circle.Radius <= 0 || circle.Position.X < 0 || circle.Position.Y < 0 || circle.Position.X >= int(aligned.Width) || circle.Position.Y >= int(aligned.Height) || MarkingAnswerMeasurementRect(image.Pt(circle.Position.X, circle.Position.Y), circle.Radius, image.Rect(0, 0, int(aligned.Width), int(aligned.Height))).Empty() {
		return MarkingReviewCandidateROI{}, ErrMarkingReviewCandidateUnavailable
	}
	return MarkingReviewCandidateROI{
		AlignedPagePath: aligned.Path, PageExam: pageExam,
		CenterX: circle.Position.X, CenterY: circle.Position.Y, Radius: circle.Radius,
		ImageWidth: int(aligned.Width), ImageHeight: int(aligned.Height),
	}, nil
}

// resolveHistoricalAnswerROI is the single offset rule. Runtime detections are
// appended page by page; question/answer indexes are first converted to that
// global answer offset, then located in the ordered page snapshots.
func resolveHistoricalAnswerROI(snapshot config.QCM, pages []markingReviewPageSnapshot, questionIndex, answerIndex int64) (int64, config.CircleValidated, error) {
	if questionIndex < 0 || questionIndex >= int64(len(snapshot.Questions)) {
		return 0, config.CircleValidated{}, ErrMarkingReviewCandidateUnavailable
	}
	question := snapshot.Questions[questionIndex]
	if answerIndex < 0 || answerIndex >= int64(len(question.Answers)) {
		return 0, config.CircleValidated{}, ErrMarkingReviewCandidateUnavailable
	}
	globalOffset := int(answerIndex)
	totalExpectedAnswers := 0
	for i := range snapshot.Questions {
		totalExpectedAnswers += len(snapshot.Questions[i].Answers)
	}
	for i := int64(0); i < questionIndex; i++ {
		globalOffset += len(snapshot.Questions[i].Answers)
	}
	totalPageAnswers := 0
	for index, page := range pages {
		if page.Page != int64(index+1) {
			return 0, config.CircleValidated{}, ErrMarkingReviewCandidateUnavailable
		}
		totalPageAnswers += len(page.Content.Answers)
	}
	if totalPageAnswers != totalExpectedAnswers {
		return 0, config.CircleValidated{}, ErrMarkingReviewCandidateUnavailable
	}
	for _, page := range pages {
		if globalOffset < len(page.Content.Answers) {
			return page.Page, page.Content.Answers[globalOffset], nil
		}
		globalOffset -= len(page.Content.Answers)
	}
	return 0, config.CircleValidated{}, ErrMarkingReviewCandidateUnavailable
}

// MarkingAnswerMeasurementRect preserves the exact center ± radius/2 ROI used
// for persisted MeanGray measurements.
func MarkingAnswerMeasurementRect(center image.Point, radius int, bounds image.Rectangle) image.Rectangle {
	half := radius / 2
	return image.Rect(center.X-half, center.Y-half, center.X+half, center.Y+half).Intersect(bounds)
}

// BuildMarkingReviewCrop returns an inspection crop in memory and never writes
// to the durable aligned page.
func BuildMarkingReviewCrop(roi MarkingReviewCandidateROI) ([]byte, error) {
	file, err := os.Open(roi.AlignedPagePath)
	if err != nil {
		return nil, ErrMarkingReviewCandidateUnavailable
	}
	defer file.Close()
	source, err := png.Decode(file)
	if err != nil {
		return nil, ErrMarkingReviewCandidateUnavailable
	}
	if source.Bounds().Dx() != roi.ImageWidth || source.Bounds().Dy() != roi.ImageHeight || roi.Radius <= 0 {
		return nil, ErrMarkingReviewCandidateUnavailable
	}
	halfSide := 2 * roi.Radius
	if halfSide < markingReviewCropMinimumHalfSide {
		halfSide = markingReviewCropMinimumHalfSide
	}
	rect := image.Rect(roi.CenterX-halfSide, roi.CenterY-halfSide, roi.CenterX+halfSide, roi.CenterY+halfSide).Intersect(source.Bounds())
	if rect.Empty() {
		return nil, ErrMarkingReviewCandidateUnavailable
	}
	crop := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(crop, crop.Bounds(), source, rect.Min, draw.Src)
	var output bytes.Buffer
	if err := png.Encode(&output, crop); err != nil {
		return nil, fmt.Errorf("encode review crop: %w", err)
	}
	return output.Bytes(), nil
}
