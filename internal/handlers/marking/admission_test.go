package marking

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectMarkingPDFPageLimit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pages   int
		wantErr error
	}{
		{name: "under limit", pages: MaxMarkingPDFPages - 1},
		{name: "exactly limit", pages: MaxMarkingPDFPages},
		{name: "over limit", pages: MaxMarkingPDFPages + 1, wantErr: errTooManyMarkingPDFPages},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "copies.pdf")
			if err := os.WriteFile(path, syntheticMarkingPDF(tc.pages, 595, 842), 0o600); err != nil {
				t.Fatal(err)
			}
			info, err := inspectMarkingPDF(context.Background(), path)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("error=%v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if info.Pages != tc.pages {
				t.Fatalf("pages=%d, want %d", info.Pages, tc.pages)
			}
		})
	}
}

func TestParseMarkingMultipartFormEnforcesRawUploadLimit(t *testing.T) {
	const testLimit = 256
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("pdffile", "copies.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(bytes.Repeat([]byte("x"), testLimit)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/dashboard/marking/processing", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	err = parseMarkingMultipartForm(response, request, testLimit)
	var maxBytesError *http.MaxBytesError
	if !errors.As(err, &maxBytesError) {
		t.Fatalf("error=%v, want http.MaxBytesError", err)
	}
}

func TestInspectMarkingPDFRejectsOversizedPage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.pdf")
	if err := os.WriteFile(path, syntheticMarkingPDF(1, MaxMarkingPDFPageDimensionPoints+1, 842), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectMarkingPDF(context.Background(), path); err != errMarkingPDFPageTooLarge {
		t.Fatalf("error=%v, want %v", err, errMarkingPDFPageTooLarge)
	}
}

func TestMarkingJobAdmissionIsBoundedAndReleasesSlots(t *testing.T) {
	admission := newMarkingJobAdmission(2)
	releaseFirst, ok := admission.tryAcquire()
	if !ok {
		t.Fatal("first job rejected")
	}
	releaseSecond, ok := admission.tryAcquire()
	if !ok {
		t.Fatal("second job rejected")
	}
	if _, ok := admission.tryAcquire(); ok {
		t.Fatal("third concurrent job admitted")
	}

	// The release closure is deliberately idempotent so duplicated cleanup paths
	// cannot free another job's slot.
	releaseFirst()
	releaseFirst()
	releaseAfterFailure, ok := admission.tryAcquire()
	if !ok {
		t.Fatal("slot was not released after a failed job")
	}
	releaseAfterFailure()
	releaseSecond()
	if _, ok := admission.tryAcquire(); !ok {
		t.Fatal("new job rejected after all previous jobs ended")
	}
}

func TestParseMarkingPDFInfoDoesNotExposeCommandDetails(t *testing.T) {
	_, _, err := parseMarkingPDFInfo("pdfinfo: /private/student/path.pdf: syntax error")
	if err == nil || strings.Contains(err.Error(), "/private/") {
		t.Fatalf("unsafe error=%v", err)
	}
}

func syntheticMarkingPDF(pages, width, height int) []byte {
	var output bytes.Buffer
	output.WriteString("%PDF-1.4\n")
	offsets := make([]int, pages+3)
	writeObject := func(id int, body string) {
		offsets[id] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", id, body)
	}
	writeObject(1, "<< /Type /Catalog /Pages 2 0 R >>")
	kids := make([]string, pages)
	for page := range pages {
		kids[page] = fmt.Sprintf("%d 0 R", page+3)
	}
	writeObject(2, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), pages))
	for page := range pages {
		writeObject(page+3, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] >>", width, height))
	}
	xref := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for id := 1; id < len(offsets); id++ {
		fmt.Fprintf(&output, "%010d 00000 n \n", offsets[id])
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return output.Bytes()
}
