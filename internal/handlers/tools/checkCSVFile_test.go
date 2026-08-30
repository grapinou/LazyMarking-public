package tools

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckCSVFileAcceptsMultipartRequestBelowLimit(t *testing.T) {
	request := newCSVMultipartRequest(t, "1", "Jean;Gabin\n")
	response := httptest.NewRecorder()

	file, err := CheckCSVFile(response, request, MaxCSVRequestBytes)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	defer request.MultipartForm.RemoveAll()

	if got := request.MultipartForm.Value["class_code_id"]; len(got) != 1 || got[0] != "1" {
		t.Fatalf("class_code_id=%v, want [1]", got)
	}
}

func TestCheckCSVFileRejectsCompleteMultipartRequestAboveLimit(t *testing.T) {
	request := newCSVMultipartRequest(t, "1", strings.Repeat("x", int(MaxCSVRequestBytes)))
	if request.ContentLength <= MaxCSVRequestBytes {
		t.Fatalf("multipart request size=%d, want more than %d", request.ContentLength, MaxCSVRequestBytes)
	}
	response := httptest.NewRecorder()

	file, err := CheckCSVFile(response, request, MaxCSVRequestBytes)
	if file != nil {
		file.Close()
		t.Fatal("unexpected file for oversized request")
	}
	var maxBytesError *http.MaxBytesError
	if !errors.As(err, &maxBytesError) {
		t.Fatalf("error=%v, want http.MaxBytesError", err)
	}
}

func TestCheckCSVFileRejectsMalformedMultipartRequest(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not multipart"))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	response := httptest.NewRecorder()

	if file, err := CheckCSVFile(response, request, MaxCSVRequestBytes); err == nil || file != nil {
		t.Fatalf("file=%v error=%v, want parsing error", file, err)
	}
}

func newCSVMultipartRequest(t *testing.T, classCodeID, csvContent string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("class_code_id", classCodeID); err != nil {
		t.Fatal(err)
	}
	file, err := writer.CreateFormFile("csvfile", "students.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte(csvContent)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
