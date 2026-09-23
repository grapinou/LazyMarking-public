package tools

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestStudentCouponsPDFRealFourPerPage(t *testing.T) {
	for _, binary := range []string{"typst", "pdftotext", "pdfinfo", "pdfimages"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("%s unavailable", binary)
		}
	}
	t.Chdir("../../..")
	username := "coupons-test-" + uuid.NewString()
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join("assets", "tmp", username)) })
	items := []StudentCoupon{
		{Name: "DUPONT Alice", Exam: "Masse volumique", Class: "2nde 3", URL: "https://school.example/copies/alpha", Code: "J7K4P2-ABCDEF"},
		{Name: "DUPONT Martin", Exam: "Masse volumique", Class: "2nde 3", URL: "https://school.example/copies/bravo", Code: "J7K4P2-GHIJKL"},
		{Name: "ÉCLAIR Chloé", Exam: "Masse volumique", Class: "2nde 3", URL: "https://school.example/copies/charlie", Code: "J7K4P2-MNOPQR"},
		{Name: "MARTIN Léa", Exam: "Masse volumique", Class: "2nde 3", URL: "https://school.example/copies/delta", Code: "J7K4P2-STUVWX"},
		{Name: "PETIT Lucas avec un nom de famille extrêmement long et accentué", Exam: "Masse volumique", Class: "2nde 3", URL: "https://school.example/copies/echo", Code: "J7K4P2-YZ1234"},
	}
	pdf, err := BuildStudentCouponsPDF(t.Context(), username, items)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("invalid PDF")
	}
	cmd := exec.Command("pdftotext", "-layout", "-", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	text, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	last := -1
	for _, item := range items {
		name := item.Name
		if strings.HasPrefix(name, "PETIT") {
			name = "PETIT Lucas"
		}
		index := strings.Index(string(text), name)
		if index <= last {
			t.Fatalf("coupon order or missing name %q: %s", item.Name, text)
		}
		last = index
		if !strings.Contains(string(text), item.Code) || !strings.Contains(string(text), item.URL) {
			t.Fatalf("coupon code/url missing for %s: %s", item.Name, text)
		}
	}
	cmd = exec.Command("pdfinfo", "-")
	cmd.Stdin = bytes.NewReader(pdf)
	info, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`(?m)^Pages:\s+2\s*$`).Match(info) {
		t.Fatalf("want two pages: %s\nTEXT PAGES: %q", info, strings.Split(string(text), "\f"))
	}
	imageDir := t.TempDir()
	pdfPath := filepath.Join(imageDir, "coupons.pdf")
	if err := os.WriteFile(pdfPath, pdf, 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("pdfimages", "-png", pdfPath, filepath.Join(imageDir, "qr")).CombinedOutput(); err != nil {
		t.Fatalf("extract QR images: %v %s", err, output)
	}
	qrFiles, err := filepath.Glob(filepath.Join(imageDir, "qr-*.png"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(qrFiles)
	if len(qrFiles) != len(items) {
		t.Fatalf("embedded QR count=%d, want %d", len(qrFiles), len(items))
	}
	for i, path := range qrFiles {
		decoded, err := DecodeWithGozxing(path)
		if err != nil {
			t.Fatal(err)
		}
		if decoded != items[i].URL {
			t.Fatalf("QR %d=%q, want %q", i, decoded, items[i].URL)
		}
	}
}

func TestStudentCorrectedPDFUsesOnlySelectedAnnotatedCopy(t *testing.T) {
	for _, binary := range []string{"pdfunite", "pdftoppm", "pdfinfo"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("%s unavailable", binary)
		}
	}
	t.Chdir(t.TempDir())
	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	_, err = conn.Exec(`
 CREATE TABLE users(id INTEGER PRIMARY KEY,username TEXT NOT NULL);
 CREATE TABLE marking_jobs(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,status TEXT NOT NULL);
 CREATE TABLE student_exam(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL);
 CREATE TABLE student_exam_content(student_exam_id INTEGER,user_id INTEGER,content TEXT);
 CREATE TABLE student_exam_page_content(student_exam_id INTEGER,page INTEGER,user_id INTEGER,content TEXT);
 CREATE TABLE marking_copy_results(id INTEGER PRIMARY KEY,user_id INTEGER,marking_job_id INTEGER,student_exam_id INTEGER,outcome TEXT,expected_pages INTEGER,detected_pages INTEGER,score_half_units INTEGER,total_points INTEGER);
 CREATE TABLE marking_question_results(id INTEGER PRIMARY KEY,copy_result_id INTEGER,question_index INTEGER,state TEXT,score_half_units INTEGER,total_points INTEGER);
 CREATE TABLE marking_answer_detections(id INTEGER PRIMARY KEY,question_result_id INTEGER,answer_index INTEGER,detected_state INTEGER,automatic_state INTEGER);
 CREATE TABLE marking_answer_reviews(answer_detection_id INTEGER,reviewed_state INTEGER);
 CREATE TABLE marking_aligned_pages(id INTEGER PRIMARY KEY,user_id INTEGER,copy_result_id INTEGER,page_exam INTEGER,storage_key TEXT,width INTEGER,height INTEGER,sha256 TEXT,created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
 INSERT INTO users VALUES(1,'alice'); INSERT INTO marking_jobs VALUES(50,1,'success');
 INSERT INTO student_exam VALUES(100,1),(101,1);
 INSERT INTO marking_copy_results VALUES(500,1,50,100,'corrected',1,1,2,1),(501,1,50,101,'corrected',1,1,0,1);
 INSERT INTO marking_question_results VALUES(600,500,0,'correct',2,1),(601,501,0,'incorrect',0,1);
 INSERT INTO marking_answer_detections VALUES(700,600,0,1,1),(701,601,0,0,0);
 `)
	if err != nil {
		t.Fatal(err)
	}
	workspace, ok := CreateOperationTempDir("alice", "marking-50")
	if !ok {
		t.Fatal("workspace")
	}
	for _, item := range []struct {
		id, copy, question int64
		gray               uint8
		name               string
	}{{100, 500, 600, 235, "Alice Unique"}, {101, 501, 601, 170, "Bob Secret"}} {
		qcm := config.QCM{Name: "Test", Student: config.StudentQCM{FirstName: item.name, ClassCodes: config.ClassCode{Name: "2nde 3"}}, Questions: []config.Question{{Content: "Question unique", Tags: config.Tags{Point: config.Point{PointValue: 1}}, Answers: []config.Answer{{Content: "Réponse", State: 1}}}}}
		snapshot, err := json.Marshal(qcm)
		if err != nil {
			t.Fatal(err)
		}
		page, err := json.Marshal(config.PageContent{Questions: []config.CircleValidated{{Position: config.Position{X: 140, Y: 80}, Radius: 12}}, Answers: []config.CircleValidated{{Position: config.Position{X: 200, Y: 150}, Radius: 12}}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(`INSERT INTO student_exam_content VALUES(?,?,?); INSERT INTO student_exam_page_content VALUES(?,?,?,?)`, item.id, 1, string(snapshot), item.id, 1, 1, string(page)); err != nil {
			t.Fatal(err)
		}
		img := image.NewGray(image.Rect(0, 0, 320, 460))
		for y := 0; y < 460; y++ {
			for x := 0; x < 320; x++ {
				img.SetGray(x, y, color.Gray{Y: item.gray})
			}
		}
		key := fmt.Sprintf("aligned/student-exam-%d/page-1.png", item.id)
		path := filepath.Join(workspace, filepath.FromSlash(key))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, img); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(bytes)
		if _, err := conn.Exec(`INSERT INTO marking_aligned_pages(user_id,copy_result_id,page_exam,storage_key,width,height,sha256) VALUES(1,?,1,?,320,460,?)`, item.copy, key, fmt.Sprintf("%x", digest)); err != nil {
			t.Fatal(err)
		}
	}
	queries := db.New(conn)
	if _, err := BuildStudentCorrectedPDF(t.Context(), queries, 1, "alice", 50, 501, 100); err != sql.ErrNoRows {
		t.Fatalf("cross-copy mismatch: %v", err)
	}
	var rasters [][]byte
	for _, item := range []struct{ copy, student int64 }{{500, 100}, {501, 101}} {
		pdf, err := BuildStudentCorrectedPDF(t.Context(), queries, 1, "alice", 50, item.copy, item.student)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
			t.Fatal("not a PDF")
		}
		cmd := exec.Command("pdfinfo", "-")
		cmd.Stdin = bytes.NewReader(pdf)
		info, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if !regexp.MustCompile(`(?m)^Pages:\s+1\s*$`).Match(info) {
			t.Fatalf("individual PDF included other pages: %s", info)
		}
		pdfPath := filepath.Join(t.TempDir(), "student.pdf")
		if err := os.WriteFile(pdfPath, pdf, 0o600); err != nil {
			t.Fatal(err)
		}
		prefix := strings.TrimSuffix(pdfPath, ".pdf")
		cmd = exec.Command("pdftoppm", "-f", "1", "-l", "1", "-singlefile", "-png", "-r", "72", pdfPath, prefix)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("rasterize: %v %s", err, output)
		}
		raster, err := os.ReadFile(prefix + ".png")
		if err != nil {
			t.Fatal(err)
		}
		rasters = append(rasters, raster)
	}
	if bytes.Equal(rasters[0], rasters[1]) {
		t.Fatalf("two distinct student scans produced the same PDF page (%d bytes each)", len(rasters[0]))
	}
	for i, raster := range rasters {
		img, err := png.Decode(bytes.NewReader(raster))
		if err != nil {
			t.Fatal(err)
		}
		colored := false
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y && !colored; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				if i == 0 && g > r+8000 && g > b+8000 {
					colored = true
					break
				}
				if i == 1 && r > g+8000 && r > b+8000 {
					colored = true
					break
				}
			}
		}
		if !colored {
			t.Fatalf("copy %d is missing expected correction annotation", i)
		}
	}
}
