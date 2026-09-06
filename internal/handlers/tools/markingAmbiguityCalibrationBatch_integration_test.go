package tools

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

// TestRealMarkingAmbiguityCalibrationBatches processes every PDF directly
// contained in LAZYMARKING_CALIBRATION_DIR. It only logs anonymous aggregates.
func TestRealMarkingAmbiguityCalibrationBatches(t *testing.T) {
	directory := os.Getenv("LAZYMARKING_CALIBRATION_DIR")
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	if directory == "" || dbPath == "" {
		t.Skip("LAZYMARKING_CALIBRATION_DIR and LAZYMARKING_TEST_DB are not configured; skipping batch ambiguity calibration")
	}
	userID := int64(1)
	if os.Getenv("LAZYMARKING_TEST_USER_ID") != "" {
		userID = parsePositiveFixtureID(t, "LAZYMARKING_TEST_USER_ID")
	}
	pdfs, err := calibrationPDFs(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(pdfs) == 0 {
		t.Fatal("calibration directory contains no PDF")
	}
	queries, closeDB := openCopiedHistoricalDB(t, dbPath)
	t.Cleanup(closeDB)

	result := expandedCalibrationReport{PDFs: len(pdfs)}
	copyResults := make([]expandedCalibrationCopy, 0)
	for _, pdfPath := range pdfs {
		corpus, unreadable, err := readCalibrationBatch(pdfPath, newMarkingIntegrationTempDir(t))
		if err != nil {
			t.Fatal(err)
		}
		result.Pages += corpus.PageCount
		result.UnreadableQRPages += unreadable
		result.CopiesDiscovered += len(corpus.Exams)
		for _, exam := range corpus.Exams {
			expected, reason, err := validateCalibrationCopy(t.Context(), queries, userID, exam)
			if err != nil {
				t.Fatal(err)
			}
			switch reason {
			case "absent":
				result.CopiesAbsentDB++
				result.CopiesIgnored++
				continue
			case "incomplete":
				result.CopiesIncomplete++
				result.CopiesIgnored++
				continue
			}
			marked, markErr := markStudentExamWithoutPanic(userID, corpus.TempDir, exam, queries)
			if markErr != nil || !marked.Status || marked.DetailedResult == nil {
				result.HomographyOrPipelineErrors++
				result.CopiesIgnored++
				continue
			}
			if marked.Pages != expected {
				t.Fatal("calibration integrity: marked page count differs from validated snapshot")
			}
			copyResult, metricErr := collectExpandedCalibrationCopy(t.Context(), queries, userID, exam, marked)
			if metricErr != nil {
				t.Fatalf("anonymous ROI metric collection failed: %v", metricErr)
			}
			copyResults = append(copyResults, copyResult)
			result.CopiesAnalyzed++
		}
	}
	populateExpandedCalibrationReport(&result, copyResults)
	if reviewDirectory := os.Getenv("LAZYMARKING_CALIBRATION_REVIEW_DIR"); reviewDirectory != "" {
		review, exportErr := exportCalibrationReview(reviewDirectory, copyResults)
		if exportErr != nil {
			t.Fatalf("anonymous crop review export failed: %v", exportErr)
		}
		result.Review = &review
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("anonymous expanded ambiguity calibration:\n%s", encoded)
}

func calibrationPDFs(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read calibration directory: %w", err)
	}
	paths := make([]string, 0)
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			paths = append(paths, filepath.Join(directory, entry.Name()))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func readCalibrationBatch(pdfPath, tempDir string) (scannedExamCorpus, int, error) {
	pdf, err := os.Open(pdfPath)
	if err != nil {
		return scannedExamCorpus{}, 0, errors.New("configured calibration PDF is not readable")
	}
	defer pdf.Close()
	if err := SplitPdf(pdf, tempDir, "page-%d.pdf"); err != nil {
		return scannedExamCorpus{}, 0, fmt.Errorf("split calibration PDF: %w", err)
	}
	pages, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		return scannedExamCorpus{}, 0, fmt.Errorf("list calibration pages: %w", err)
	}
	qrCodes := make([]config.QrCodeInfo, 0, len(pages))
	unreadable := 0
	for _, pagePath := range pages {
		pageName := filepath.Base(pagePath)
		pngName, convertErr := ConvertPdfToPng(tempDir, pageName, "")
		if convertErr != nil {
			unreadable++
			continue
		}
		qrData, qrErr := QrReader(filepath.Join(tempDir, pngName))
		if qrErr != nil {
			unreadable++
			continue
		}
		var info config.QrCodeInfo
		if json.Unmarshal([]byte(qrData), &info) != nil || info.StudentExamID <= 0 || info.PageExam <= 0 {
			unreadable++
			continue
		}
		info.PageName = pngName
		qrCodes = append(qrCodes, info)
	}
	return scannedExamCorpus{Exams: GroupQrCodes(qrCodes), TempDir: tempDir, PageCount: len(pages), QRCount: len(qrCodes)}, unreadable, nil
}

func validateCalibrationCopy(ctx context.Context, queries *db.Queries, userID int64, exam config.Exam) (int, string, error) {
	content, err := queries.GetStudentContentExam(ctx, db.GetStudentContentExamParams{StudentExamID: exam.StudentExamID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "absent", nil
	}
	if err != nil {
		return 0, "", fmt.Errorf("load calibration snapshot: %w", err)
	}
	expected := int(content.PageTot)
	seen := make(map[int]struct{}, len(exam.Pages))
	for _, page := range exam.Pages {
		if page.Number < 1 || page.Number > expected {
			return expected, "incomplete", nil
		}
		if _, duplicate := seen[page.Number]; duplicate {
			return expected, "incomplete", nil
		}
		seen[page.Number] = struct{}{}
	}
	if len(seen) != expected {
		return expected, "incomplete", nil
	}
	return expected, "", nil
}

type expandedCalibrationDetection struct {
	MeanGray       float64
	State          int
	StdDev         float64
	DarkProportion float64
	Minimum        float64
	Maximum        float64
	sourcePath     string
	answer         config.CircleValidated
}

type expandedCalibrationCopy struct {
	Detections []expandedCalibrationDetection
}

func collectExpandedCalibrationCopy(ctx context.Context, queries *db.Queries, userID int64, exam config.Exam, marked config.MarkExam) (expandedCalibrationCopy, error) {
	pages := append([]config.Page(nil), exam.Pages...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })
	staged := make(map[int]string, len(marked.AlignedPages))
	for _, page := range marked.AlignedPages {
		staged[page.PageExam] = page.Path
	}
	automatic := make([]config.AnswerDetection, 0)
	for _, question := range marked.DetailedResult.Questions {
		automatic = append(automatic, question.AnswerDetections...)
	}
	result := expandedCalibrationCopy{Detections: make([]expandedCalibrationDetection, 0, len(automatic))}
	detectionIndex := 0
	for _, page := range pages {
		pageJSON, err := queries.GetPageContent(ctx, db.GetPageContentParams{StudentExamID: exam.StudentExamID, Page: int64(page.Number), UserID: userID})
		if err != nil {
			return expandedCalibrationCopy{}, err
		}
		var content config.PageContent
		if err := json.Unmarshal([]byte(pageJSON), &content); err != nil {
			return expandedCalibrationCopy{}, err
		}
		imageFile, err := os.Open(staged[page.Number])
		if err != nil {
			return expandedCalibrationCopy{}, err
		}
		aligned, _, err := image.Decode(imageFile)
		_ = imageFile.Close()
		if err != nil {
			return expandedCalibrationCopy{}, err
		}
		for _, answer := range content.Answers {
			if detectionIndex >= len(automatic) {
				return expandedCalibrationCopy{}, errors.New("ROI count exceeds automatic detection count")
			}
			metric, err := calibrationROIMetric(aligned, answer)
			if err != nil {
				return expandedCalibrationCopy{}, err
			}
			detection := automatic[detectionIndex]
			if math.Abs(metric.MeanGray-detection.MeanGray) > 1.1 {
				return expandedCalibrationCopy{}, fmt.Errorf("test ROI mean %.4f differs from production MeanGray %.4f", metric.MeanGray, detection.MeanGray)
			}
			metric.MeanGray = detection.MeanGray
			metric.State = detection.State
			metric.sourcePath = staged[page.Number]
			metric.answer = answer
			result.Detections = append(result.Detections, metric)
			detectionIndex++
		}
	}
	if detectionIndex != len(automatic) {
		return expandedCalibrationCopy{}, errors.New("automatic detection count exceeds ROI count")
	}
	return result, nil
}

func calibrationROIMetric(source image.Image, answer config.CircleValidated) (expandedCalibrationDetection, error) {
	halfRadius := int(answer.Radius / 2)
	rect := image.Rect(answer.Position.X-halfRadius, answer.Position.Y-halfRadius, answer.Position.X+halfRadius, answer.Position.Y+halfRadius).Intersect(source.Bounds())
	if rect.Empty() {
		return expandedCalibrationDetection{}, errors.New("calibration ROI is outside aligned page")
	}
	count := 0
	sum := 0.0
	sumSquares := 0.0
	dark := 0
	minimum := 255.0
	maximum := 0.0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			gray := float64(imageGray(source.At(x, y)))
			count++
			sum += gray
			sumSquares += gray * gray
			if gray < 128 {
				dark++
			}
			minimum = math.Min(minimum, gray)
			maximum = math.Max(maximum, gray)
		}
	}
	mean := sum / float64(count)
	variance := math.Max(0, sumSquares/float64(count)-mean*mean)
	return expandedCalibrationDetection{MeanGray: mean, StdDev: math.Sqrt(variance), DarkProportion: float64(dark) / float64(count), Minimum: minimum, Maximum: maximum}, nil
}

func imageGray(value color.Color) uint8 {
	return color.GrayModel.Convert(value).(color.Gray).Y
}

type expandedCandidate struct {
	Delta            int     `json:"delta"`
	Count            int     `json:"count"`
	Percent          float64 `json:"percent"`
	Checked          int     `json:"checked"`
	Unchecked        int     `json:"unchecked"`
	MeanPerCopy      float64 `json:"mean_per_copy"`
	MedianPerCopy    float64 `json:"median_per_copy"`
	MaximumOnOneCopy int     `json:"maximum_on_one_copy"`
	CopiesWithoutAny int     `json:"copies_without_any"`
}

type expandedCalibrationReport struct {
	PDFs                       int                      `json:"pdfs"`
	Pages                      int                      `json:"pages"`
	CopiesDiscovered           int                      `json:"copies_discovered"`
	CopiesAnalyzed             int                      `json:"copies_analyzed"`
	CopiesIgnored              int                      `json:"copies_ignored"`
	UnreadableQRPages          int                      `json:"unreadable_qr_pages"`
	CopiesAbsentDB             int                      `json:"copies_absent_db"`
	CopiesIncomplete           int                      `json:"copies_incomplete"`
	HomographyOrPipelineErrors int                      `json:"homography_or_pipeline_errors"`
	Cases                      int                      `json:"cases"`
	All                        calibrationSummary       `json:"mean_gray_all"`
	Checked                    calibrationSummary       `json:"mean_gray_checked"`
	Unchecked                  calibrationSummary       `json:"mean_gray_unchecked"`
	Distance                   calibrationSummary       `json:"distance_to_150"`
	DistanceCounts             map[string]int           `json:"distance_counts"`
	DistancePercents           map[string]float64       `json:"distance_percents"`
	Candidates                 []expandedCandidate      `json:"candidates"`
	Histogram                  []calibrationBin         `json:"threshold_histogram"`
	CheckedCopyMedians         calibrationSummary       `json:"checked_copy_medians"`
	UncheckedCopyMedians       calibrationSummary       `json:"unchecked_copy_medians"`
	CopyMinimumDistances       calibrationSummary       `json:"copy_minimum_distances"`
	ROIStdDev                  calibrationSummary       `json:"roi_stddev"`
	ROIDarkProportion          calibrationSummary       `json:"roi_dark_proportion"`
	ROIMinimum                 calibrationSummary       `json:"roi_minimum"`
	ROIMaximum                 calibrationSummary       `json:"roi_maximum"`
	DarkHeterogeneousCases     int                      `json:"dark_heterogeneous_cases"`
	FiniteAndInRange           bool                     `json:"finite_and_in_range"`
	Review                     *calibrationReviewResult `json:"review,omitempty"`
}

func populateExpandedCalibrationReport(report *expandedCalibrationReport, copies []expandedCalibrationCopy) {
	all, checked, unchecked, distances := []float64{}, []float64{}, []float64{}, []float64{}
	checkedMedians, uncheckedMedians, minimumDistances := []float64{}, []float64{}, []float64{}
	stdDevs, darkProportions, minimums, maximums := []float64{}, []float64{}, []float64{}, []float64{}
	valid := true
	for _, copyResult := range copies {
		copyChecked, copyUnchecked, copyDistances := []float64{}, []float64{}, []float64{}
		for _, detection := range copyResult.Detections {
			all = append(all, detection.MeanGray)
			distance := math.Abs(detection.MeanGray - MarkingDetectionThreshold)
			distances = append(distances, distance)
			copyDistances = append(copyDistances, distance)
			stdDevs = append(stdDevs, detection.StdDev)
			darkProportions = append(darkProportions, detection.DarkProportion)
			minimums = append(minimums, detection.Minimum)
			maximums = append(maximums, detection.Maximum)
			if detection.State == 1 {
				checked = append(checked, detection.MeanGray)
				copyChecked = append(copyChecked, detection.MeanGray)
				if detection.MeanGray < 75 && detection.StdDev >= 80 {
					report.DarkHeterogeneousCases++
				}
			} else {
				unchecked = append(unchecked, detection.MeanGray)
				copyUnchecked = append(copyUnchecked, detection.MeanGray)
			}
			if math.IsNaN(detection.MeanGray) || math.IsInf(detection.MeanGray, 0) || detection.MeanGray < 0 || detection.MeanGray > 255 {
				valid = false
			}
		}
		if len(copyChecked) > 0 {
			checkedMedians = append(checkedMedians, quantile(copyChecked, 0.5))
		}
		if len(copyUnchecked) > 0 {
			uncheckedMedians = append(uncheckedMedians, quantile(copyUnchecked, 0.5))
		}
		if len(copyDistances) > 0 {
			sort.Float64s(copyDistances)
			minimumDistances = append(minimumDistances, copyDistances[0])
		}
	}
	report.Cases = len(all)
	report.All, report.Checked, report.Unchecked = summarizeCalibration(all), summarizeCalibration(checked), summarizeCalibration(unchecked)
	report.Distance = summarizeCalibration(distances)
	report.CheckedCopyMedians, report.UncheckedCopyMedians = summarizeCalibration(checkedMedians), summarizeCalibration(uncheckedMedians)
	report.CopyMinimumDistances = summarizeCalibration(minimumDistances)
	report.ROIStdDev, report.ROIDarkProportion = summarizeCalibration(stdDevs), summarizeCalibration(darkProportions)
	report.ROIMinimum, report.ROIMaximum = summarizeCalibration(minimums), summarizeCalibration(maximums)
	report.FiniteAndInRange = valid
	report.DistanceCounts, report.DistancePercents = map[string]int{}, map[string]float64{}
	for _, limit := range []int{2, 5, 10, 15, 20, 25, 30, 40, 50, 75} {
		count := countWithin(distances, float64(limit))
		key := "le_" + calibrationInt(limit)
		report.DistanceCounts[key] = count
		report.DistancePercents[key] = percentage(count, len(all))
	}
	for _, delta := range []int{5, 10, 15, 20, 25, 30} {
		candidate := expandedCandidate{Delta: delta}
		perCopy := make([]float64, 0, len(copies))
		for _, copyResult := range copies {
			copyCount := 0
			for _, detection := range copyResult.Detections {
				if math.Abs(detection.MeanGray-MarkingDetectionThreshold) <= float64(delta) {
					candidate.Count++
					copyCount++
					if detection.State == 1 {
						candidate.Checked++
					} else {
						candidate.Unchecked++
					}
				}
			}
			if copyCount == 0 {
				candidate.CopiesWithoutAny++
			}
			if copyCount > candidate.MaximumOnOneCopy {
				candidate.MaximumOnOneCopy = copyCount
			}
			perCopy = append(perCopy, float64(copyCount))
		}
		candidate.Percent = percentage(candidate.Count, len(all))
		if len(copies) > 0 {
			candidate.MeanPerCopy = float64(candidate.Count) / float64(len(copies))
		}
		candidate.MedianPerCopy = quantile(perCopy, 0.5)
		report.Candidates = append(report.Candidates, candidate)
	}
	report.Histogram = expandedThresholdHistogram(all)
}

func expandedThresholdHistogram(values []float64) []calibrationBin {
	edges := []float64{0, 20, 40, 60, 75, 100, 110, 120, 130, 135, 140, 145, 150, 155, 160, 165, 170, 180, 200, 225, 240, 256}
	bins := make([]calibrationBin, len(edges)-1)
	for index := range bins {
		bins[index] = calibrationBin{From: edges[index], To: edges[index+1]}
	}
	for _, value := range values {
		for index := range bins {
			if value >= bins[index].From && value < bins[index].To {
				bins[index].Count++
				break
			}
		}
	}
	return bins
}

type calibrationReviewResult struct {
	CategoryA        int     `json:"category_a"`
	CategoryB        int     `json:"category_b"`
	CategoryC        int     `json:"category_c"`
	MergedDuplicates int     `json:"merged_duplicates"`
	Crops            int     `json:"crops"`
	MeanGrayMin      float64 `json:"mean_gray_min"`
	MeanGrayMax      float64 `json:"mean_gray_max"`
	StdDevMin        float64 `json:"stddev_min"`
	StdDevMax        float64 `json:"stddev_max"`
	ManifestCreated  bool    `json:"manifest_created"`
	ContactCreated   bool    `json:"contact_sheet_created"`
}

type calibrationReviewKey struct {
	copyIndex, detectionIndex int
}

type calibrationReviewCandidate struct {
	Name       string
	Category   string
	MeanGray   float64
	State      string
	Distance   float64
	StdDev     float64
	DarkRatio  float64
	Minimum    float64
	Maximum    float64
	SourcePath string
	Answer     config.CircleValidated
}

func exportCalibrationReview(configuredDirectory string, copies []expandedCalibrationCopy) (calibrationReviewResult, error) {
	directory, err := prepareCalibrationReviewDirectory(configuredDirectory)
	if err != nil {
		return calibrationReviewResult{}, err
	}
	selected := make(map[calibrationReviewKey]map[string]struct{})
	result := calibrationReviewResult{}
	add := func(key calibrationReviewKey, category string) {
		if selected[key] == nil {
			selected[key] = make(map[string]struct{})
		}
		selected[key][category] = struct{}{}
	}
	for copyIndex, copyResult := range copies {
		for detectionIndex, detection := range copyResult.Detections {
			key := calibrationReviewKey{copyIndex: copyIndex, detectionIndex: detectionIndex}
			if math.Abs(detection.MeanGray-MarkingDetectionThreshold) <= 30 {
				add(key, "A")
				result.CategoryA++
			}
			if detection.MeanGray < 75 && detection.StdDev >= 80 {
				add(key, "B")
				result.CategoryB++
			}
		}
	}
	for _, extreme := range calibrationReviewExtremes(copies) {
		add(extreme, "C")
		result.CategoryC++
	}

	keys := make([]calibrationReviewKey, 0, len(selected))
	memberships := 0
	for key, categories := range selected {
		keys = append(keys, key)
		memberships += len(categories)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].copyIndex != keys[j].copyIndex {
			return keys[i].copyIndex < keys[j].copyIndex
		}
		return keys[i].detectionIndex < keys[j].detectionIndex
	})
	result.MergedDuplicates = memberships - len(keys)
	result.Crops = len(keys)
	candidates := make([]calibrationReviewCandidate, 0, len(keys))
	means, deviations := make([]float64, 0, len(keys)), make([]float64, 0, len(keys))
	for index, key := range keys {
		detection := copies[key.copyIndex].Detections[key.detectionIndex]
		name := fmt.Sprintf("candidate-%04d", index+1)
		categories := make([]string, 0, len(selected[key]))
		for _, category := range []string{"A", "B", "C"} {
			if _, present := selected[key][category]; present {
				categories = append(categories, category)
			}
		}
		state := "unchecked"
		if detection.State == 1 {
			state = "checked"
		}
		candidate := calibrationReviewCandidate{
			Name: name, Category: strings.Join(categories, "+"), MeanGray: detection.MeanGray,
			State: state, Distance: math.Abs(detection.MeanGray - MarkingDetectionThreshold),
			StdDev: detection.StdDev, DarkRatio: detection.DarkProportion,
			Minimum: detection.Minimum, Maximum: detection.Maximum,
			SourcePath: detection.sourcePath, Answer: detection.answer,
		}
		if err := writeCalibrationCrop(directory, candidate); err != nil {
			return calibrationReviewResult{}, err
		}
		candidates = append(candidates, candidate)
		means = append(means, detection.MeanGray)
		deviations = append(deviations, detection.StdDev)
	}
	if len(means) > 0 {
		sort.Float64s(means)
		sort.Float64s(deviations)
		result.MeanGrayMin, result.MeanGrayMax = means[0], means[len(means)-1]
		result.StdDevMin, result.StdDevMax = deviations[0], deviations[len(deviations)-1]
	}
	if err := writeCalibrationManifest(directory, candidates); err != nil {
		return calibrationReviewResult{}, err
	}
	result.ManifestCreated = true
	if err := writeCalibrationContactSheet(directory, candidates); err != nil {
		return calibrationReviewResult{}, err
	}
	result.ContactCreated = true
	return result, nil
}

func calibrationReviewExtremes(copies []expandedCalibrationCopy) []calibrationReviewKey {
	type copyMedian struct {
		index  int
		median float64
	}
	checkedCopies, uncheckedCopies := []copyMedian{}, []copyMedian{}
	for copyIndex, copyResult := range copies {
		checked, unchecked := []float64{}, []float64{}
		for _, detection := range copyResult.Detections {
			if detection.State == 1 {
				checked = append(checked, detection.MeanGray)
			} else {
				unchecked = append(unchecked, detection.MeanGray)
			}
		}
		if len(checked) > 0 {
			checkedCopies = append(checkedCopies, copyMedian{index: copyIndex, median: quantile(checked, 0.5)})
		}
		if len(unchecked) > 0 {
			uncheckedCopies = append(uncheckedCopies, copyMedian{index: copyIndex, median: quantile(unchecked, 0.5)})
		}
	}
	keys := make([]calibrationReviewKey, 0, 6)
	if len(checkedCopies) > 0 {
		sort.Slice(checkedCopies, func(i, j int) bool { return checkedCopies[i].median > checkedCopies[j].median })
		keys = append(keys, representativeCalibrationDetections(copies, checkedCopies[0].index, 1, checkedCopies[0].median, 3)...)
	}
	if len(uncheckedCopies) > 0 {
		sort.Slice(uncheckedCopies, func(i, j int) bool { return uncheckedCopies[i].median < uncheckedCopies[j].median })
		keys = append(keys, representativeCalibrationDetections(copies, uncheckedCopies[0].index, 0, uncheckedCopies[0].median, 3)...)
	}
	return keys
}

func representativeCalibrationDetections(copies []expandedCalibrationCopy, copyIndex, state int, median float64, limit int) []calibrationReviewKey {
	keys := make([]calibrationReviewKey, 0)
	for detectionIndex, detection := range copies[copyIndex].Detections {
		if detection.State == state {
			keys = append(keys, calibrationReviewKey{copyIndex: copyIndex, detectionIndex: detectionIndex})
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		left := math.Abs(copies[copyIndex].Detections[keys[i].detectionIndex].MeanGray - median)
		right := math.Abs(copies[copyIndex].Detections[keys[j].detectionIndex].MeanGray - median)
		if left != right {
			return left < right
		}
		return keys[i].detectionIndex < keys[j].detectionIndex
	})
	if len(keys) > limit {
		keys = keys[:limit]
	}
	return keys
}

func prepareCalibrationReviewDirectory(configured string) (string, error) {
	absolute, err := filepath.Abs(configured)
	if err != nil {
		return "", err
	}
	repository, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(repository, "go.mod")); statErr == nil {
			break
		}
		parent := filepath.Dir(repository)
		if parent == repository {
			return "", errors.New("cannot locate repository root")
		}
		repository = parent
	}
	prospective, err := resolveProspectiveCalibrationPath(absolute)
	if err != nil {
		return "", err
	}
	if calibrationPathIsWithin(repository, prospective) {
		return "", errors.New("calibration review directory must be outside the repository")
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	if calibrationPathIsWithin(repository, resolved) {
		return "", errors.New("calibration review directory must be outside the repository")
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		name := entry.Name()
		controlled := name == "manifest.csv" || name == "review-contact-sheet.html" || isCalibrationCandidatePNG(name)
		if !controlled {
			continue
		}
		if entry.IsDir() {
			return "", errors.New("controlled review artifact name is unexpectedly a directory")
		}
		if err := os.Remove(filepath.Join(resolved, name)); err != nil {
			return "", err
		}
	}
	return resolved, nil
}

func resolveProspectiveCalibrationPath(path string) (string, error) {
	existing := path
	missing := make([]string, 0)
	for {
		_, err := os.Lstat(existing)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", errors.New("calibration review path has no existing ancestor")
		}
		missing = append(missing, filepath.Base(existing))
		existing = parent
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", err
	}
	for index := len(missing) - 1; index >= 0; index-- {
		resolved = filepath.Join(resolved, missing[index])
	}
	return resolved, nil
}

func calibrationPathIsWithin(repository, candidate string) bool {
	relative, err := filepath.Rel(repository, candidate)
	return err == nil && (relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))))
}

func isCalibrationCandidatePNG(name string) bool {
	if len(name) != len("candidate-0000.png") || !strings.HasPrefix(name, "candidate-") || !strings.HasSuffix(name, ".png") {
		return false
	}
	for _, character := range name[len("candidate-"):len("candidate-0000")] {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func writeCalibrationCrop(directory string, candidate calibrationReviewCandidate) error {
	file, err := os.Open(candidate.SourcePath)
	if err != nil {
		return err
	}
	source, _, err := image.Decode(file)
	_ = file.Close()
	if err != nil {
		return err
	}
	halfSize := candidate.Answer.Radius * 2
	if halfSize < 30 {
		halfSize = 30
	}
	center := candidate.Answer.Position
	rect := image.Rect(center.X-halfSize, center.Y-halfSize, center.X+halfSize, center.Y+halfSize).Intersect(source.Bounds())
	if rect.Empty() {
		return errors.New("review crop is outside aligned page")
	}
	destination := filepath.Join(directory, candidate.Name+".png")
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	encodeErr := png.Encode(output, source.(interface {
		SubImage(image.Rectangle) image.Image
	}).SubImage(rect))
	closeErr := output.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

func writeCalibrationManifest(directory string, candidates []calibrationReviewCandidate) error {
	file, err := os.OpenFile(filepath.Join(directory, "manifest.csv"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"candidate", "category", "mean_gray", "detected_state", "distance_to_150", "stddev", "dark_pixel_ratio", "min_gray", "max_gray", "human_label", "human_comment"}); err != nil {
		_ = file.Close()
		return err
	}
	for _, candidate := range candidates {
		record := []string{candidate.Name, candidate.Category, fmt.Sprintf("%.4f", candidate.MeanGray), candidate.State,
			fmt.Sprintf("%.4f", candidate.Distance), fmt.Sprintf("%.4f", candidate.StdDev), fmt.Sprintf("%.6f", candidate.DarkRatio),
			fmt.Sprintf("%.0f", candidate.Minimum), fmt.Sprintf("%.0f", candidate.Maximum), "", ""}
		if err := writer.Write(record); err != nil {
			_ = file.Close()
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writeCalibrationContactSheet(directory string, candidates []calibrationReviewCandidate) error {
	const document = `<!doctype html><html lang="fr"><head><meta charset="utf-8"><title>Calibration review</title><style>body{font-family:sans-serif;margin:1rem}.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:1rem}.card{border:1px solid #ccc;padding:.75rem}.card img{width:100%;image-rendering:auto}.meta{font-size:.85rem;line-height:1.4}</style></head><body><h1>Calibration review</h1><div class="grid">{{range .}}<div class="card"><img src="{{.Name}}.png" alt="{{.Name}}"><div class="meta">{{.Name}}<br>catégorie {{.Category}}<br>MeanGray {{printf "%.2f" .MeanGray}}<br>{{.State}}<br>distance {{printf "%.2f" .Distance}}<br>stddev {{printf "%.2f" .StdDev}}</div></div>{{end}}</div></body></html>`
	page, err := template.New("review").Parse(document)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(directory, "review-contact-sheet.html"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	executeErr := page.Execute(file, candidates)
	closeErr := file.Close()
	if executeErr != nil {
		return executeErr
	}
	return closeErr
}
