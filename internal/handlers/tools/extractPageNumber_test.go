package tools

import "testing"

func TestExtractPageNumberAcceptsExporterNamesWithOrWithoutSeparator(t *testing.T) {
	for _, name := range []string{
		"student-exam-12345page-2-of-3.png",
		"smoke-prof__page-2-of-3.png",
	} {
		page, total, ok := ExtractPageNumber(name)
		if !ok || page != 2 || total != 3 {
			t.Fatalf("ExtractPageNumber(%q) = (%d, %d, %v), want (2, 3, true)", name, page, total, ok)
		}
	}
}
