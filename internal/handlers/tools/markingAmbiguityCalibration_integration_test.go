package tools

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
)

// TestRealMarkingAmbiguityCalibration reuses the real marking pipeline and
// emits only aggregate, anonymous statistics. Private fixture paths, technical
// IDs, scans and per-copy values are never included in its output.
func TestRealMarkingAmbiguityCalibration(t *testing.T) {
	fixtures := []struct {
		pdfEnv, studentIDEnv string
	}{
		{pdfEnv: "LAZYMARKING_TEST_PDF_1_PAGE", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_1_PAGE"},
		{pdfEnv: "LAZYMARKING_TEST_PDF_2_PAGES", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_2_PAGES"},
		{pdfEnv: "LAZYMARKING_TEST_PDF", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_3_PAGES"},
	}
	if os.Getenv("LAZYMARKING_TEST_DB") == "" || os.Getenv("LAZYMARKING_TEST_USER_ID") == "" {
		t.Skip("private historical DB and user ID are not configured; skipping ambiguity calibration")
	}
	for _, fixture := range fixtures {
		if os.Getenv(fixture.pdfEnv) == "" || os.Getenv(fixture.studentIDEnv) == "" {
			t.Skip("private PDFs and technical IDs are not all configured; skipping ambiguity calibration")
		}
	}

	userID := parsePositiveFixtureID(t, "LAZYMARKING_TEST_USER_ID")
	queries, closeDB := openCopiedHistoricalDB(t, os.Getenv("LAZYMARKING_TEST_DB"))
	t.Cleanup(closeDB)
	copies := make([]calibrationCopy, 0, len(fixtures))
	pages := 0
	for _, fixture := range fixtures {
		studentExamID := parsePositiveFixtureID(t, fixture.studentIDEnv)
		corpus := readAndGroupScannedExams(t, os.Getenv(fixture.pdfEnv), newMarkingIntegrationTempDir(t))
		exam := findExamByIDWithTechnicalSummary(t, corpus.Exams, studentExamID)
		marked, err := markStudentExamWithoutPanic(userID, corpus.TempDir, exam, queries)
		if err != nil {
			t.Fatal(err)
		}
		validateSuccessfulMark(t, marked)
		pages += marked.Pages
		copyResult := calibrationCopy{}
		for _, question := range marked.DetailedResult.Questions {
			copyResult.detections = append(copyResult.detections, question.AnswerDetections...)
		}
		copies = append(copies, copyResult)
	}

	report := buildCalibrationReport(copies, pages)
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("anonymous ambiguity calibration:\n%s", encoded)
}

type calibrationCopy struct {
	detections []config.AnswerDetection
}

type calibrationSummary struct {
	Count  int     `json:"count"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	P1     float64 `json:"p1"`
	P5     float64 `json:"p5"`
	P10    float64 `json:"p10"`
	P25    float64 `json:"p25"`
	P75    float64 `json:"p75"`
	P90    float64 `json:"p90"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
}

type calibrationCandidate struct {
	Delta            int     `json:"delta"`
	Count            int     `json:"count"`
	Percent          float64 `json:"percent"`
	Checked          int     `json:"checked"`
	Unchecked        int     `json:"unchecked"`
	MeanPerCopy      float64 `json:"mean_per_copy"`
	CopiesWithoutAny int     `json:"copies_without_any"`
}

type calibrationBin struct {
	From  float64 `json:"from"`
	To    float64 `json:"to"`
	Count int     `json:"count"`
}

type calibrationReport struct {
	Copies               int                    `json:"copies"`
	Pages                int                    `json:"pages"`
	Cases                int                    `json:"cases"`
	All                  calibrationSummary     `json:"mean_gray_all"`
	Checked              calibrationSummary     `json:"mean_gray_checked"`
	Unchecked            calibrationSummary     `json:"mean_gray_unchecked"`
	Distance             calibrationSummary     `json:"distance_to_150"`
	DistanceCounts       map[string]int         `json:"distance_counts"`
	DistancePercents     map[string]float64     `json:"distance_percents"`
	Candidates           []calibrationCandidate `json:"candidates"`
	Histogram            []calibrationBin       `json:"histogram"`
	Exact150             int                    `json:"exactly_150"`
	WithinPointFive      int                    `json:"distance_le_0_5"`
	FiniteAndInRange     bool                   `json:"finite_and_in_range"`
	UncheckedCopyMedians calibrationSummary     `json:"unchecked_copy_medians"`
}

func buildCalibrationReport(copies []calibrationCopy, pages int) calibrationReport {
	all := make([]float64, 0)
	checked := make([]float64, 0)
	unchecked := make([]float64, 0)
	distances := make([]float64, 0)
	copyMedians := make([]float64, 0, len(copies))
	valid := true
	exact150 := 0
	withinPointFive := 0
	for _, copyResult := range copies {
		copyUnchecked := make([]float64, 0)
		for _, detection := range copyResult.detections {
			value := detection.MeanGray
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 255 {
				valid = false
			}
			all = append(all, value)
			distance := math.Abs(value - MarkingDetectionThreshold)
			distances = append(distances, distance)
			if value == MarkingDetectionThreshold {
				exact150++
			}
			if distance <= 0.5 {
				withinPointFive++
			}
			if detection.State == 1 {
				checked = append(checked, value)
			} else {
				unchecked = append(unchecked, value)
				copyUnchecked = append(copyUnchecked, value)
			}
		}
		if len(copyUnchecked) > 0 {
			copyMedians = append(copyMedians, quantile(copyUnchecked, 0.5))
		}
	}

	report := calibrationReport{
		Copies: len(copies), Pages: pages, Cases: len(all),
		All: summarizeCalibration(all), Checked: summarizeCalibration(checked),
		Unchecked: summarizeCalibration(unchecked), Distance: summarizeCalibration(distances),
		DistanceCounts: make(map[string]int), DistancePercents: make(map[string]float64),
		Exact150: exact150, WithinPointFive: withinPointFive, FiniteAndInRange: valid,
		UncheckedCopyMedians: summarizeCalibration(copyMedians),
	}
	for _, limit := range []int{2, 5, 10, 15, 20, 25, 30} {
		count := countWithin(distances, float64(limit))
		key := "le_" + calibrationInt(limit)
		report.DistanceCounts[key] = count
		report.DistancePercents[key] = percentage(count, len(all))
	}
	for _, delta := range []int{5, 10, 15, 20} {
		candidate := calibrationCandidate{Delta: delta}
		for _, copyResult := range copies {
			copyCount := 0
			for _, detection := range copyResult.detections {
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
		}
		candidate.Percent = percentage(candidate.Count, len(all))
		candidate.MeanPerCopy = float64(candidate.Count) / float64(len(copies))
		report.Candidates = append(report.Candidates, candidate)
	}
	report.Histogram = calibrationHistogram(all)
	return report
}

func summarizeCalibration(values []float64) calibrationSummary {
	if len(values) == 0 {
		return calibrationSummary{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	sum := 0.0
	for _, value := range sorted {
		sum += value
	}
	return calibrationSummary{
		Count: len(sorted), Min: sorted[0], Max: sorted[len(sorted)-1], Mean: sum / float64(len(sorted)),
		Median: quantileSorted(sorted, 0.5), P1: quantileSorted(sorted, 0.01), P5: quantileSorted(sorted, 0.05),
		P10: quantileSorted(sorted, 0.10), P25: quantileSorted(sorted, 0.25), P75: quantileSorted(sorted, 0.75),
		P90: quantileSorted(sorted, 0.90), P95: quantileSorted(sorted, 0.95), P99: quantileSorted(sorted, 0.99),
	}
}

func quantile(values []float64, probability float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	return quantileSorted(sorted, probability)
}

func quantileSorted(sorted []float64, probability float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	position := probability * float64(len(sorted)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return sorted[lower]
	}
	return sorted[lower] + (sorted[upper]-sorted[lower])*(position-float64(lower))
}

func calibrationHistogram(values []float64) []calibrationBin {
	edges := []float64{0, 20, 40, 60, 80, 100, 120, 130, 140, 145, 150, 155, 160, 170, 180, 200, 220, 240, 256}
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

func countWithin(values []float64, limit float64) int {
	count := 0
	for _, value := range values {
		if value <= limit {
			count++
		}
	}
	return count
}

func percentage(count, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100 * float64(count) / float64(total)
}

func calibrationInt(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 3)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for left, right := 0, len(digits)-1; left < right; left, right = left+1, right-1 {
		digits[left], digits[right] = digits[right], digits[left]
	}
	return string(digits)
}
