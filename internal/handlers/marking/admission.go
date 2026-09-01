package marking

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	MaxMarkingUploadBytes            int64 = 100 << 20
	MaxMarkingPDFPages                     = 500
	MaxConcurrentMarkingJobs               = 2
	MaxMarkingPDFPageDimensionPoints       = 2000
	markingPDFInspectionTimeout            = 30 * time.Second
)

var (
	errInvalidMarkingPDF      = errors.New("invalid marking PDF")
	errTooManyMarkingPDFPages = errors.New("too many marking PDF pages")
	errMarkingPDFPageTooLarge = errors.New("marking PDF page dimensions are too large")
	markingPageSizePattern    = regexp.MustCompile(`^Page\s+\d+ size:\s+([0-9]+(?:\.[0-9]+)?) x ([0-9]+(?:\.[0-9]+)?) pts`)
	globalMarkingJobAdmission = newMarkingJobAdmission(MaxConcurrentMarkingJobs)
)

type markingPDFInfo struct {
	Pages int
}

func parseMarkingMultipartForm(w http.ResponseWriter, r *http.Request, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	return r.ParseMultipartForm(maxBytes)
}

func inspectMarkingPDF(ctx context.Context, path string) (markingPDFInfo, error) {
	inspectionCtx, cancel := context.WithTimeout(ctx, markingPDFInspectionTimeout)
	defer cancel()

	output, err := exec.CommandContext(
		inspectionCtx,
		"pdfinfo",
		"-f", "1",
		"-l", strconv.Itoa(MaxMarkingPDFPages+1),
		"-box",
		path,
	).Output()
	if err != nil {
		return markingPDFInfo{}, errInvalidMarkingPDF
	}

	info, dimensions, err := parseMarkingPDFInfo(string(output))
	if err != nil {
		return markingPDFInfo{}, err
	}
	if info.Pages > MaxMarkingPDFPages {
		return markingPDFInfo{}, errTooManyMarkingPDFPages
	}
	if dimensions != info.Pages {
		return markingPDFInfo{}, errInvalidMarkingPDF
	}
	return info, nil
}

func parseMarkingPDFInfo(output string) (markingPDFInfo, int, error) {
	var info markingPDFInfo
	dimensions := 0
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Pages:") {
			pages, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
			if err != nil || pages <= 0 {
				return markingPDFInfo{}, 0, errInvalidMarkingPDF
			}
			info.Pages = pages
			continue
		}
		matches := markingPageSizePattern.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}
		width, widthErr := strconv.ParseFloat(matches[1], 64)
		height, heightErr := strconv.ParseFloat(matches[2], 64)
		if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
			return markingPDFInfo{}, 0, errInvalidMarkingPDF
		}
		if width > MaxMarkingPDFPageDimensionPoints || height > MaxMarkingPDFPageDimensionPoints {
			return markingPDFInfo{}, 0, errMarkingPDFPageTooLarge
		}
		dimensions++
	}
	if err := scanner.Err(); err != nil || info.Pages == 0 {
		return markingPDFInfo{}, 0, fmt.Errorf("%w: unreadable metadata", errInvalidMarkingPDF)
	}
	return info, dimensions, nil
}

type markingJobAdmission struct {
	slots chan struct{}
}

func newMarkingJobAdmission(capacity int) *markingJobAdmission {
	return &markingJobAdmission{slots: make(chan struct{}, capacity)}
}

func (a *markingJobAdmission) tryAcquire() (func(), bool) {
	select {
	case a.slots <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-a.slots }) }, true
	default:
		return nil, false
	}
}
