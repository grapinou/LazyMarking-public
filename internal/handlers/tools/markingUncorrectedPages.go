package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
)

const MarkingUncorrectedPagesFilename = "corrected_NOT.pdf"

type MarkingRejectedScanPage struct {
	ScanPage int
	Path     string
}

// ResolveMarkingRejectedScanPages selects only original scan PNGs: pages whose
// QR was unreadable and pages assigned to a copy that was not corrected.
func ResolveMarkingRejectedScanPages(splitPages []string, qrNotDetected []string, exams []config.Exam, marked []config.MarkExam) ([]MarkingRejectedScanPage, error) {
	catalog := make(map[string]MarkingRejectedScanPage, len(splitPages))
	for _, splitPage := range splitPages {
		base := filepath.Base(splitPage)
		if filepath.Ext(base) != ".pdf" || !strings.HasPrefix(base, "page-") {
			return nil, fmt.Errorf("unexpected split page name")
		}
		pageNumber, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(base, "page-"), ".pdf"))
		if err != nil || pageNumber <= 0 {
			return nil, fmt.Errorf("invalid split page number")
		}
		pngBase := strings.TrimSuffix(base, ".pdf") + ".png"
		if _, duplicate := catalog[pngBase]; duplicate {
			return nil, fmt.Errorf("duplicate split page number")
		}
		catalog[pngBase] = MarkingRejectedScanPage{ScanPage: pageNumber, Path: filepath.Join(filepath.Dir(splitPage), pngBase)}
	}

	rejectedNames := make(map[string]struct{}, len(qrNotDetected))
	for _, name := range qrNotDetected {
		rejectedNames[filepath.Base(name)] = struct{}{}
	}
	correctedCopies := make(map[int64]struct{}, len(marked))
	for _, result := range marked {
		if result.Status {
			correctedCopies[result.StudentExamID] = struct{}{}
		}
	}
	for _, exam := range exams {
		if _, corrected := correctedCopies[exam.StudentExamID]; corrected {
			continue
		}
		for _, page := range exam.Pages {
			rejectedNames[filepath.Base(page.Name)] = struct{}{}
		}
	}

	rejected := make([]MarkingRejectedScanPage, 0, len(rejectedNames))
	for name := range rejectedNames {
		page, ok := catalog[name]
		if !ok {
			return nil, fmt.Errorf("rejected page is outside scan catalog")
		}
		rejected = append(rejected, page)
	}
	sort.Slice(rejected, func(i, j int) bool { return rejected[i].ScanPage < rejected[j].ScanPage })
	return rejected, nil
}

// BuildUncorrectedPagesPDF publishes one canonical PDF atomically without
// modifying the original rejected scan PNGs.
func BuildUncorrectedPagesPDF(workspace string, pages []MarkingRejectedScanPage) (string, error) {
	if len(pages) == 0 {
		return "", nil
	}
	if err := ensureDirectoryTree(workspace, false, 0); err != nil {
		return "", fmt.Errorf("validate marking workspace: %w", err)
	}
	canonical := filepath.Join(workspace, MarkingUncorrectedPagesFilename)
	if _, err := os.Lstat(canonical); err == nil {
		return "", errors.New("uncorrected pages artifact already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect uncorrected pages artifact: %w", err)
	}

	ordered := append([]MarkingRejectedScanPage(nil), pages...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ScanPage < ordered[j].ScanPage })
	stagingDir, err := os.MkdirTemp(workspace, ".corrected-not-*")
	if err != nil {
		return "", fmt.Errorf("create uncorrected pages staging: %w", err)
	}
	defer func() { _ = os.RemoveAll(stagingDir) }()

	pagePDFs := make([]string, 0, len(ordered))
	previousPage := 0
	for index, page := range ordered {
		if page.ScanPage <= 0 || page.ScanPage == previousPage {
			return "", errors.New("invalid rejected scan page order")
		}
		previousPage = page.ScanPage
		if err := validateRejectedScanPNG(workspace, page); err != nil {
			return "", err
		}
		pagePDF := filepath.Join(stagingDir, fmt.Sprintf("page-%06d.pdf", index+1))
		if err := convertPngToPDFFile(page.Path, pagePDF); err != nil {
			return "", fmt.Errorf("convert rejected scan page: %w", err)
		}
		pagePDFs = append(pagePDFs, pagePDF)
	}

	stagedArtifact := filepath.Join(stagingDir, MarkingUncorrectedPagesFilename)
	if err := MergePdf(pagePDFs, stagedArtifact); err != nil {
		return "", fmt.Errorf("merge rejected scan pages: %w", err)
	}
	if err := validateGeneratedPDF(stagingDir, stagedArtifact); err != nil {
		return "", fmt.Errorf("validate rejected scan PDF: %w", err)
	}
	if err := os.Rename(stagedArtifact, canonical); err != nil {
		return "", fmt.Errorf("publish rejected scan PDF: %w", err)
	}
	return MarkingUncorrectedPagesFilename, nil
}

func validateRejectedScanPNG(workspace string, page MarkingRejectedScanPage) error {
	cleanWorkspace, cleanPath := filepath.Clean(workspace), filepath.Clean(page.Path)
	relative, err := filepath.Rel(cleanWorkspace, cleanPath)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.Dir(relative) != "." {
		return errors.New("rejected scan page is outside marking workspace")
	}
	expectedName := "page-" + strconv.Itoa(page.ScanPage) + ".png"
	if filepath.Base(cleanPath) != expectedName {
		return errors.New("rejected scan page name does not match scan order")
	}
	info, err := os.Lstat(cleanPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("rejected scan page is not a regular file")
	}
	return nil
}
