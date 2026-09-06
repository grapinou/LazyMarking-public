package tools

import (
	"path"
	"path/filepath"
	"strings"
)

type MarkingArtifactKind string

const (
	MarkingArtifactCorrected MarkingArtifactKind = "corrected"
	MarkingArtifactMarks     MarkingArtifactKind = "marks"
)

// MarkingArtifactFilename derives a display-only filename from the uploaded
// PDF name. It must never be used as a server-side storage path.
func MarkingArtifactFilename(originalFilename string, kind MarkingArtifactKind) string {
	fallback := string(kind) + ".pdf"
	if kind != MarkingArtifactCorrected && kind != MarkingArtifactMarks {
		return fallback
	}

	// path.Base handles both native and browser-supplied Windows separators
	// after normalization. Header control characters are discarded.
	name := path.Base(strings.ReplaceAll(originalFilename, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == 0 {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." {
		return fallback
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if base == "" {
		return fallback
	}
	return base + "_" + string(kind) + ".pdf"
}

// MarkingSourceFilename keeps only the user-visible basename for durable
// provenance; it is deliberately unrelated to artifact storage paths.
func MarkingSourceFilename(originalFilename string) string {
	name := path.Base(strings.ReplaceAll(originalFilename, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == 0 {
			return -1
		}
		return r
	}, name)
	return strings.TrimSpace(name)
}
