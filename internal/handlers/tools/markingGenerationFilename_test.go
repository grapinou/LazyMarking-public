package tools

import (
	"strings"
	"testing"
)

func TestMarkingGenerationTitle(t *testing.T) {
	if got := MarkingGenerationTitle("2nde 3", "Contrôle de physique n°2"); got != "2nde 3 — Contrôle de physique n°2" {
		t.Fatalf("MarkingGenerationTitle() = %q", got)
	}
	if got := MarkingGenerationTitle(" ", ""); got != "Évaluation" {
		t.Fatalf("empty MarkingGenerationTitle() = %q", got)
	}
}

func TestMarkingGenerationReportFilename(t *testing.T) {
	tests := []struct {
		name, className, examName, want string
	}{
		{"identified report", "2nde 3", "Contrôle de physique n°2", "2nde-3-controle-de-physique-n2-bilan.pdf"},
		{"accents and punctuation", "6e B", "Électricité & énergie", "6e-b-electricite-energie-bilan.pdf"},
		{"empty metadata", " \t", "—", "evaluation-bilan.pdf"},
		{"one usable value", "", " Contrôle !!! ", "controle-bilan.pdf"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MarkingGenerationReportFilename(test.className, test.examName); got != test.want {
				t.Fatalf("MarkingGenerationReportFilename(%q, %q) = %q, want %q", test.className, test.examName, got, test.want)
			}
		})
	}
	long := MarkingGenerationReportFilename(strings.Repeat("Évaluation très longue ", 20), strings.Repeat("Classe ", 20))
	if len(strings.TrimSuffix(long, ".pdf")) > markingGenerationFilenameBaseMaxLength || !strings.HasSuffix(long, "-bilan.pdf") {
		t.Fatalf("long filename is not bounded or loses its suffix: %q", long)
	}
}
