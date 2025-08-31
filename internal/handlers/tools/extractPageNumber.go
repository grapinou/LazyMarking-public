package tools

import (
	"regexp"
	"strconv"
)

var rePage = regexp.MustCompile(`_page-(\d+)-of-(\d+)\.png$`)

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
