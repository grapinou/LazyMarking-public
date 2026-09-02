package tools

import (
	"regexp"
	"strconv"
)

// ExportTypstToPNGs appends "page-…" to the generated Typst basename. Exam
// files created with os.CreateTemp do not necessarily end in an underscore,
// while preview names historically do. Accept both exporter outputs.
var rePage = regexp.MustCompile(`(?:_)?page-(\d+)-of-(\d+)\.png$`)

func ExtractPageNumber(pageName string) (page, total int, ok bool) {
	m := rePage.FindStringSubmatch(pageName)
	if m == nil {
		return 0, 0, false
	}

	page, err1 := strconv.Atoi(m[1])
	total, err2 := strconv.Atoi(m[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}

	return page, total, true
}
