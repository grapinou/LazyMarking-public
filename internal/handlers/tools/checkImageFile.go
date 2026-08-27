package tools

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	MaxImageFileBytes    int64 = 20 << 20
	MaxImageRequestBytes int64 = 24 << 20
	MaxImageDimension          = 8000
	MaxImagePixels       int64 = 25_000_000

	imageMultipartMemory int64 = 1 << 20
)

type imageFileLimits struct {
	fileBytes    int64
	requestBytes int64
	memoryBytes  int64
}

var defaultImageFileLimits = imageFileLimits{
	fileBytes:    MaxImageFileBytes,
	requestBytes: MaxImageRequestBytes,
	memoryBytes:  imageMultipartMemory,
}

// CheckImageFile bounds and parses the multipart request, validates the image
// content and dimensions, and returns the image rewound to its first byte.
// The caller must close the file and remove r.MultipartForm's temporary files.
func CheckImageFile(w http.ResponseWriter, r *http.Request) (multipart.File, *multipart.FileHeader, image.Config, error) {
	return checkImageFile(w, r, defaultImageFileLimits)
}

func checkImageFile(w http.ResponseWriter, r *http.Request, limits imageFileLimits) (file multipart.File, header *multipart.FileHeader, cfg image.Config, err error) {
	r.Body = http.MaxBytesReader(w, r.Body, limits.requestBytes)
	if err = r.ParseMultipartForm(limits.memoryBytes); err != nil {
		removeMultipartFiles(r)
		return nil, nil, image.Config{}, fmt.Errorf("parse image multipart form: %w", err)
	}

	failed := true
	defer func() {
		if failed {
			if file != nil {
				_ = file.Close()
			}
			removeMultipartFiles(r)
		}
	}()

	file, header, err = r.FormFile("image")
	if err != nil {
		return nil, nil, image.Config{}, fmt.Errorf("read image multipart field: %w", err)
	}

	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, nil, image.Config{}, fmt.Errorf("measure image: %w", err)
	}
	if size <= 0 || size > limits.fileBytes {
		return nil, nil, image.Config{}, fmt.Errorf("image size %d is outside the allowed range", size)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, nil, image.Config{}, fmt.Errorf("rewind image before inspection: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return nil, nil, image.Config{}, fmt.Errorf("unsupported image extension %q", ext)
	}

	var sniff [512]byte
	n, readErr := io.ReadFull(file, sniff[:])
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		return nil, nil, image.Config{}, fmt.Errorf("inspect image content: %w", readErr)
	}
	contentType := http.DetectContentType(sniff[:n])
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, nil, image.Config{}, fmt.Errorf("rewind image before decoding: %w", err)
	}

	cfg, format, err := image.DecodeConfig(file)
	if err != nil {
		return nil, nil, image.Config{}, fmt.Errorf("decode image configuration: %w", err)
	}
	if !imageFormatMatches(ext, contentType, format) {
		return nil, nil, image.Config{}, fmt.Errorf("image extension %q does not match content format %q", ext, format)
	}
	if err = validateImageDimensions(cfg.Width, cfg.Height); err != nil {
		return nil, nil, image.Config{}, err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, nil, image.Config{}, fmt.Errorf("rewind validated image: %w", err)
	}

	failed = false
	return file, header, cfg, nil
}

func imageFormatMatches(ext, contentType, format string) bool {
	switch ext {
	case ".png":
		return contentType == "image/png" && format == "png"
	case ".jpg", ".jpeg":
		return contentType == "image/jpeg" && format == "jpeg"
	default:
		return false
	}
}

func removeMultipartFiles(r *http.Request) {
	if r.MultipartForm != nil {
		_ = r.MultipartForm.RemoveAll()
	}
}

func validateImageDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid image dimensions %dx%d", width, height)
	}
	if width > MaxImageDimension || height > MaxImageDimension {
		return fmt.Errorf("image dimensions %dx%d exceed %d pixels", width, height, MaxImageDimension)
	}
	if int64(width)*int64(height) > MaxImagePixels {
		return fmt.Errorf("image dimensions %dx%d exceed %d pixels total", width, height, MaxImagePixels)
	}
	return nil
}

// ValidateImageResize validates a percentage scale before any float-to-int
// conversion and returns the effective dimensions used by OpenCV.
func ValidateImageResize(width, height int, scale float64) (int, int, error) {
	if err := validateImageDimensions(width, height); err != nil {
		return 0, 0, err
	}
	if math.IsNaN(scale) || math.IsInf(scale, 0) || scale < 1 {
		return 0, 0, fmt.Errorf("invalid image resize percentage %v", scale)
	}

	effectiveWidth := float64(width) * (scale / 100)
	effectiveHeight := float64(height) * (scale / 100)
	if math.IsNaN(effectiveWidth) || math.IsInf(effectiveWidth, 0) ||
		math.IsNaN(effectiveHeight) || math.IsInf(effectiveHeight, 0) {
		return 0, 0, fmt.Errorf("non-finite resized image dimensions")
	}
	if effectiveWidth < 1 || effectiveHeight < 1 ||
		effectiveWidth > MaxImageDimension || effectiveHeight > MaxImageDimension {
		return 0, 0, fmt.Errorf("resized image dimensions %.2fx%.2f are outside the allowed range", effectiveWidth, effectiveHeight)
	}
	if effectiveWidth*effectiveHeight > float64(MaxImagePixels) {
		return 0, 0, fmt.Errorf("resized image dimensions %.2fx%.2f exceed %d pixels total", effectiveWidth, effectiveHeight, MaxImagePixels)
	}

	newWidth := int(effectiveWidth)
	newHeight := int(effectiveHeight)
	if err := validateImageDimensions(newWidth, newHeight); err != nil {
		return 0, 0, fmt.Errorf("invalid resized image: %w", err)
	}
	return newWidth, newHeight, nil
}

// ReadImageConfig reads and validates existing image dimensions without
// decoding all pixels.
func ReadImageConfig(path string) (image.Config, error) {
	validatedPath, lstatInfo, err := validateRegularFile(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return image.Config{}, err
	}
	file, err := os.Open(validatedPath)
	if err != nil {
		return image.Config{}, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !os.SameFile(lstatInfo, openedInfo) {
		return image.Config{}, fmt.Errorf("stored image changed during open")
	}

	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return image.Config{}, fmt.Errorf("decode existing image configuration: %w", err)
	}
	if err := validateImageDimensions(cfg.Width, cfg.Height); err != nil {
		return image.Config{}, err
	}
	return cfg, nil
}
