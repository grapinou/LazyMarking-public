package tools

import "testing"

func TestMarkingArtifactFilename(t *testing.T) {
	tests := []struct {
		name     string
		original string
		kind     MarkingArtifactKind
		want     string
	}{
		{"simple corrected", "6eB.pdf", MarkingArtifactCorrected, "6eB_corrected.pdf"},
		{"simple marks", "6eB.pdf", MarkingArtifactMarks, "6eB_marks.pdf"},
		{"last extension only", "classe.6eB.pdf", MarkingArtifactCorrected, "classe.6eB_corrected.pdf"},
		{"spaces", "Contrôle 6e B.pdf", MarkingArtifactMarks, "Contrôle 6e B_marks.pdf"},
		{"unicode", "évaluation-électricité.pdf", MarkingArtifactCorrected, "évaluation-électricité_corrected.pdf"},
		{"uppercase extension", "TEST.PDF", MarkingArtifactCorrected, "TEST_corrected.pdf"},
		{"without extension", "fichier sans extension", MarkingArtifactMarks, "fichier sans extension_marks.pdf"},
		{"traversal", "../../6eB.pdf", MarkingArtifactCorrected, "6eB_corrected.pdf"},
		{"windows traversal", `..\..\6eB.pdf`, MarkingArtifactMarks, "6eB_marks.pdf"},
		{"header controls", "classe\r\nX-Evil: yes.pdf", MarkingArtifactCorrected, "classeX-Evil: yes_corrected.pdf"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MarkingArtifactFilename(test.original, test.kind); got != test.want {
				t.Fatalf("MarkingArtifactFilename(%q, %q) = %q, want %q", test.original, test.kind, got, test.want)
			}
		})
	}
}

func TestMarkingSourceFilenameKeepsOnlySafeBasename(t *testing.T) {
	if got := MarkingSourceFilename("../../scan\r\n.pdf"); got != "scan.pdf" {
		t.Fatalf("MarkingSourceFilename() = %q", got)
	}
}
