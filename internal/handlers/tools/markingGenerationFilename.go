package tools

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const markingGenerationFilenameBaseMaxLength = 120

// MarkingGenerationTitle identifies a cumulative report independently of the
// page around it. Empty metadata is omitted without leaving stray separators.
func MarkingGenerationTitle(className, examName string) string {
	parts := make([]string, 0, 2)
	for _, value := range []string{className, examName} {
		if value = strings.TrimSpace(value); value != "" {
			parts = append(parts, value)
		}
	}
	if len(parts) == 0 {
		return "Évaluation"
	}
	return strings.Join(parts, " — ")
}

// MarkingGenerationReportFilename returns an ASCII download name suitable for
// Content-Disposition. It is a display name and is never used as a storage path.
func MarkingGenerationReportFilename(className, examName string) string {
	parts := make([]string, 0, 3)
	for _, value := range []string{className, examName} {
		if slug := markingFilenameSlug(value); slug != "" {
			parts = append(parts, slug)
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "evaluation")
	}
	base := strings.Join(parts, "-")
	suffix := "-bilan"
	maxMetadataLength := markingGenerationFilenameBaseMaxLength - len(suffix)
	if len(base) > maxMetadataLength {
		base = strings.TrimRight(base[:maxMetadataLength], "-")
	}
	base += suffix
	if len(base) > markingGenerationFilenameBaseMaxLength {
		base = strings.TrimRight(base[:markingGenerationFilenameBaseMaxLength], "-")
	}
	return base + ".pdf"
}

func markingFilenameSlug(value string) string {
	value = norm.NFD.String(strings.ToLower(strings.TrimSpace(value)))
	var slug strings.Builder
	separator := false
	for _, r := range value {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r == '°':
			continue
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			if separator && slug.Len() > 0 {
				slug.WriteByte('-')
			}
			slug.WriteRune(r)
			separator = false
		default:
			separator = true
		}
	}
	return slug.String()
}
