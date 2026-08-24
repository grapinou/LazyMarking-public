package tools

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func TestCheckPdfFileReturnsClosableRewoundFile(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("pdffile", "exam.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("%PDF-test")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest("POST", "/marking", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	file, err := CheckPdfFile(request, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "%PDF-test" {
		t.Fatalf("content = %q", content)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
