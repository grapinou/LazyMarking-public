package tools

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadImageConfigRejectsFinalSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation commonly requires additional privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target.png")
	if err := os.WriteFile(target, []byte("not relevant"), 0o640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "linked.png")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := ReadImageConfig(link); err == nil {
		t.Fatal("ReadImageConfig followed final symlink")
	}
}

func encodedTestImage(t *testing.T, format string, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	var err error
	if format == "png" {
		err = png.Encode(&encoded, img)
	} else {
		err = jpeg.Encode(&encoded, img, nil)
	}
	if err != nil {
		t.Fatalf("encode test image: %v", err)
	}
	return encoded.Bytes()
}

func imageMultipartRequest(t *testing.T, filename string, contents []byte, fieldsFirst bool) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if fieldsFirst {
		if err := writer.WriteField("question_id", "1"); err != nil {
			t.Fatal(err)
		}
		if err := writer.WriteField("width", "100"); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/images/add", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestCheckImageFileAcceptsValidImagesAndRewinds(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		format    string
		signature []byte
	}{
		{name: "png", filename: "picture.png", format: "png", signature: []byte("\x89PNG\r\n\x1a\n")},
		{name: "jpeg", filename: "picture.jpeg", format: "jpeg", signature: []byte{0xff, 0xd8, 0xff}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := imageMultipartRequest(t, tc.filename, encodedTestImage(t, tc.format, 20, 10), false)
			file, _, cfg, err := CheckImageFile(httptest.NewRecorder(), req)
			if err != nil {
				t.Fatalf("CheckImageFile: %v", err)
			}
			defer file.Close()
			defer req.MultipartForm.RemoveAll()
			if cfg.Width != 20 || cfg.Height != 10 {
				t.Fatalf("dimensions = %dx%d", cfg.Width, cfg.Height)
			}
			got := make([]byte, len(tc.signature))
			if _, err := io.ReadFull(file, got); err != nil {
				t.Fatalf("read rewound file: %v", err)
			}
			if !bytes.Equal(got, tc.signature) {
				t.Fatalf("signature = %x, want %x", got, tc.signature)
			}
		})
	}
}

func TestCheckImageFileRejectsInvalidContentAndExtensions(t *testing.T) {
	pngData := encodedTestImage(t, "png", 10, 10)
	jpegData := encodedTestImage(t, "jpeg", 10, 10)
	tests := []struct {
		name     string
		filename string
		contents []byte
	}{
		{name: "fake png", filename: "fake.png", contents: []byte("not an image")},
		{name: "fake jpeg", filename: "fake.jpg", contents: []byte("not an image")},
		{name: "png with jpeg extension", filename: "wrong.jpg", contents: pngData},
		{name: "jpeg with png extension", filename: "wrong.png", contents: jpegData},
		{name: "unsupported extension", filename: "image.gif", contents: pngData},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := imageMultipartRequest(t, tc.filename, tc.contents, false)
			if _, _, _, err := CheckImageFile(httptest.NewRecorder(), req); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestCheckImageFileRejectsFileOverConfiguredLimit(t *testing.T) {
	data := encodedTestImage(t, "png", 10, 10)
	req := imageMultipartRequest(t, "image.png", data, false)
	limits := imageFileLimits{fileBytes: int64(len(data) - 1), requestBytes: 1 << 20, memoryBytes: 64}
	if _, _, _, err := checkImageFile(httptest.NewRecorder(), req, limits); err == nil {
		t.Fatal("expected file size error")
	}
}

func TestCheckImageFileBoundsRequestBeforeReadingTextFields(t *testing.T) {
	req := imageMultipartRequest(t, "image.png", encodedTestImage(t, "png", 30, 30), true)
	limits := imageFileLimits{fileBytes: 1 << 20, requestBytes: 128, memoryBytes: 64}
	if _, _, _, err := checkImageFile(httptest.NewRecorder(), req, limits); err == nil {
		t.Fatal("expected request size error")
	}
	if req.MultipartForm != nil && (req.MultipartForm.Value["question_id"] != nil || req.MultipartForm.Value["width"] != nil) {
		t.Fatal("oversized multipart request exposed text fields")
	}
}

func TestValidateImageDimensions(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		wantErr       bool
	}{
		{name: "valid", width: 5000, height: 5000},
		{name: "zero width", width: 0, height: 1, wantErr: true},
		{name: "negative height", width: 1, height: -1, wantErr: true},
		{name: "width too large", width: 8001, height: 1, wantErr: true},
		{name: "height too large", width: 1, height: 8001, wantErr: true},
		{name: "too many pixels", width: 5001, height: 5000, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateImageDimensions(tc.width, tc.height)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateImageResize(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		scale         float64
		wantWidth     int
		wantHeight    int
		wantErr       bool
	}{
		{name: "nan", width: 100, height: 100, scale: math.NaN(), wantErr: true},
		{name: "positive infinity", width: 100, height: 100, scale: math.Inf(1), wantErr: true},
		{name: "negative infinity", width: 100, height: 100, scale: math.Inf(-1), wantErr: true},
		{name: "zero", width: 100, height: 100, scale: 0, wantErr: true},
		{name: "negative", width: 100, height: 100, scale: -1, wantErr: true},
		{name: "width too large", width: 8000, height: 1, scale: 101, wantErr: true},
		{name: "height too large", width: 1, height: 8000, scale: 101, wantErr: true},
		{name: "too many pixels", width: 5000, height: 5000, scale: 101, wantErr: true},
		{name: "valid", width: 1000, height: 500, scale: 200, wantWidth: 2000, wantHeight: 1000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			width, height, err := ValidateImageResize(tc.width, tc.height, tc.scale)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && (width != tc.wantWidth || height != tc.wantHeight) {
				t.Fatalf("dimensions = %dx%d, want %dx%d", width, height, tc.wantWidth, tc.wantHeight)
			}
		})
	}
}

func TestCheckImageFileErrorDoesNotRetainMultipartTemporaryFiles(t *testing.T) {
	contents := []byte(strings.Repeat("not-image", 100))
	req := imageMultipartRequest(t, "fake.png", contents, false)
	limits := imageFileLimits{fileBytes: 1 << 20, requestBytes: 1 << 20, memoryBytes: 1}
	if _, _, _, err := checkImageFile(httptest.NewRecorder(), req, limits); err == nil {
		t.Fatal("expected validation error")
	}
	if req.MultipartForm != nil {
		for _, headers := range req.MultipartForm.File {
			for _, header := range headers {
				file, err := header.Open()
				if err == nil {
					file.Close()
					t.Fatal("multipart temporary file still exists after validation error")
				}
			}
		}
	}
}
