package tools

import (
	"math"
	"sort"

	"github.com/grapinou/LazyMarking/internal/config"
)

// ExtractScores retourne une slice de tous les scores d'une liste de MarkExam
func ExtractScores(exams []config.MarkExam) []float64 {
	scores := make([]float64, 0, len(exams))
	for _, e := range exams {
		scores = append(scores, e.Score)
	}
	return scores
}

// Mean calcule la moyenne
func Mean(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range scores {
		sum += x
	}
	return sum / float64(len(scores))
}

// StdDev calcule l’écart-type (population)
func StdDev(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	mean := Mean(scores)
	var sum float64
	for _, x := range scores {
		d := x - mean
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(scores)))
}

// Median calcule la médiane
func Median(scores []float64) float64 {
	n := len(scores)
	if n == 0 {
		return 0
	}
	tmp := make([]float64, n)
	copy(tmp, scores)
	sort.Float64s(tmp)

	if n%2 == 0 {
		return (tmp[n/2-1] + tmp[n/2]) / 2
	}
	return tmp[n/2]
}

func ComputeStatMarking(markExams []config.MarkExam) (float64, float64, float64) {
	scores := ExtractScores(markExams)
	mean := Mean(scores)
	stdDev := StdDev(scores)
	median := Median(scores)

	return mean, stdDev, median
}
