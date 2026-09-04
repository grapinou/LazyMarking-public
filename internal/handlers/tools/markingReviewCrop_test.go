package tools

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestResolveHistoricalAnswerROIOffsets(t *testing.T) {
	snapshot := reviewCropSnapshot()
	pages := reviewCropPages()
	tests := []struct {
		name             string
		question, answer int64
		wantPage         int64
		wantX            int
		wantErr          bool
	}{
		{name: "question 1 answer 1", question: 0, answer: 0, wantPage: 1, wantX: 10},
		{name: "last answer question 1", question: 0, answer: 2, wantPage: 2, wantX: 30},
		{name: "first answer question 2", question: 1, answer: 0, wantPage: 2, wantX: 40},
		{name: "last answer on later page", question: 1, answer: 1, wantPage: 3, wantX: 50},
		{name: "invalid answer", question: 0, answer: 3, wantErr: true},
		{name: "invalid question", question: 2, answer: 0, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page, circle, err := resolveHistoricalAnswerROI(snapshot, pages, tc.question, tc.answer)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && (page != tc.wantPage || circle.Position.X != tc.wantX) {
				t.Fatalf("page=%d circle=%+v, want page=%d x=%d", page, circle, tc.wantPage, tc.wantX)
			}
		})
	}
}

func TestResolveMarkingReviewCandidateROIPreservesMeasurementGeometry(t *testing.T) {
	fixture := newReviewCropFixture(t, "success", "corrected")
	roi, err := ResolveMarkingReviewCandidateROI(t.Context(), fixture.queries, 1, "alice", 50, 703)
	if err != nil {
		t.Fatal(err)
	}
	if roi.PageExam != 2 || roi.CenterX != 40 || roi.CenterY != 40 || roi.Radius != 12 || roi.ImageWidth != 100 || roi.ImageHeight != 100 {
		t.Fatalf("ROI=%+v", roi)
	}
	wantRect := image.Rect(34, 34, 46, 46)
	if got := MarkingAnswerMeasurementRect(image.Pt(roi.CenterX, roi.CenterY), roi.Radius, image.Rect(0, 0, roi.ImageWidth, roi.ImageHeight)); got != wantRect {
		t.Fatalf("measurement rect=%v, want %v", got, wantRect)
	}
	detections, err := GetAnswerDetections(filepath.Dir(roi.AlignedPagePath), filepath.Base(roi.AlignedPagePath), []config.CircleValidated{{Position: config.Position{X: roi.CenterX, Y: roi.CenterY}, Radius: roi.Radius}})
	if err != nil || len(detections) != 1 {
		t.Fatalf("GetAnswerDetections=%+v err=%v", detections, err)
	}
	var persistedMean float64
	if err := fixture.conn.QueryRow(`SELECT mean_gray FROM marking_answer_detections WHERE id=703`).Scan(&persistedMean); err != nil {
		t.Fatal(err)
	}
	if detections[0].MeanGray != persistedMean {
		t.Fatalf("mapped MeanGray=%v, persisted=%v", detections[0].MeanGray, persistedMean)
	}
}

func TestBuildMarkingReviewCropClampsAndPreservesSource(t *testing.T) {
	path, original := writeReviewCropPNG(t, t.TempDir(), "source.png", 100, 80)
	tests := []struct {
		name       string
		x, y       int
		wantWidth  int
		wantHeight int
	}{
		{name: "normal", x: 50, y: 40, wantWidth: 60, wantHeight: 60},
		{name: "left", x: 2, y: 40, wantWidth: 32, wantHeight: 60},
		{name: "right", x: 98, y: 40, wantWidth: 32, wantHeight: 60},
		{name: "top", x: 50, y: 2, wantWidth: 60, wantHeight: 32},
		{name: "bottom", x: 50, y: 78, wantWidth: 60, wantHeight: 32},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output, err := BuildMarkingReviewCrop(MarkingReviewCandidateROI{AlignedPagePath: path, CenterX: tc.x, CenterY: tc.y, Radius: 10, ImageWidth: 100, ImageHeight: 80})
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := png.Decode(bytes.NewReader(output))
			if err != nil {
				t.Fatalf("decode crop: %v", err)
			}
			if decoded.Bounds().Dx() != tc.wantWidth || decoded.Bounds().Dy() != tc.wantHeight {
				t.Fatalf("crop dimensions=%v, want %dx%d", decoded.Bounds(), tc.wantWidth, tc.wantHeight)
			}
		})
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, original) {
		t.Fatal("durable source changed after crop")
	}
}

type reviewCropFixture struct {
	conn    *sql.DB
	queries *db.Queries
}

func newReviewCropFixture(t *testing.T, status, outcome string) reviewCropFixture {
	t.Helper()
	t.Chdir(t.TempDir())
	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY, username TEXT NOT NULL);
		CREATE TABLE marking_jobs(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,status TEXT NOT NULL,review_revision INTEGER NOT NULL,artifacts_revision INTEGER NOT NULL);
		CREATE TABLE student_exam_content(student_exam_id INTEGER NOT NULL,user_id INTEGER NOT NULL,content TEXT NOT NULL);
		CREATE TABLE student_exam_page_content(student_exam_id INTEGER NOT NULL,page INTEGER NOT NULL,content TEXT NOT NULL,user_id INTEGER NOT NULL);
		CREATE TABLE marking_copy_results(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,marking_job_id INTEGER NOT NULL,student_exam_id INTEGER NOT NULL,outcome TEXT NOT NULL);
		CREATE TABLE marking_question_results(id INTEGER PRIMARY KEY,copy_result_id INTEGER NOT NULL,question_index INTEGER NOT NULL,total_points INTEGER NOT NULL);
		CREATE TABLE marking_answer_detections(id INTEGER PRIMARY KEY,question_result_id INTEGER NOT NULL,answer_index INTEGER NOT NULL,detected_state INTEGER NOT NULL,mean_gray REAL NOT NULL,automatic_state INTEGER);
		CREATE TABLE marking_answer_reviews(id INTEGER PRIMARY KEY,answer_detection_id INTEGER,reviewed_state INTEGER,revision INTEGER);
		CREATE TABLE marking_aligned_pages(id INTEGER PRIMARY KEY,user_id INTEGER NOT NULL,copy_result_id INTEGER NOT NULL,page_exam INTEGER NOT NULL,storage_key TEXT NOT NULL,width INTEGER NOT NULL,height INTEGER NOT NULL,sha256 TEXT NOT NULL,created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		INSERT INTO users VALUES(1,'alice'),(2,'bob');
		INSERT INTO marking_jobs VALUES(50,1,?,0,0),(51,2,'success',0,0);
		INSERT INTO marking_copy_results VALUES(500,1,50,100,?);
		INSERT INTO marking_question_results VALUES(600,500,0,1),(601,500,1,1);
		INSERT INTO marking_answer_detections VALUES(700,600,0,0,230,NULL),(701,600,1,0,230,NULL),(702,600,2,0,230,NULL),(703,601,0,1,40,NULL),(704,601,1,0,230,NULL);
	`, status, outcome); err != nil {
		t.Fatal(err)
	}
	snapshotJSON, _ := json.Marshal(reviewCropSnapshot())
	if _, err := conn.Exec(`INSERT INTO student_exam_content VALUES(100,1,?)`, string(snapshotJSON)); err != nil {
		t.Fatal(err)
	}
	for _, page := range reviewCropPages() {
		contentJSON, _ := json.Marshal(page.Content)
		if _, err := conn.Exec(`INSERT INTO student_exam_page_content VALUES(100,?,?,1)`, page.Page, string(contentJSON)); err != nil {
			t.Fatal(err)
		}
	}
	for page := int64(1); page <= 3; page++ {
		path, content := writeReviewCropPNG(t, filepath.Join("assets", "tmp", "alice", "marking-50", "aligned", "student-exam-100"), "page-"+string(rune('0'+page))+".png", 100, 100)
		digest := sha256.Sum256(content)
		key := "aligned/student-exam-100/" + filepath.Base(path)
		if _, err := conn.Exec(`INSERT INTO marking_aligned_pages(id,user_id,copy_result_id,page_exam,storage_key,width,height,sha256) VALUES(?,1,500,?,?,100,100,?)`, 800+page, page, key, hex.EncodeToString(digest[:])); err != nil {
			t.Fatal(err)
		}
	}
	return reviewCropFixture{conn: conn, queries: db.New(conn)}
}

func reviewCropSnapshot() config.QCM {
	return config.QCM{Questions: []config.Question{
		{Answers: make([]config.Answer, 3)},
		{Answers: make([]config.Answer, 2)},
	}}
}

func reviewCropPages() []markingReviewPageSnapshot {
	circle := func(x int) config.CircleValidated {
		return config.CircleValidated{Position: config.Position{X: x, Y: 40}, Radius: 12}
	}
	return []markingReviewPageSnapshot{
		{Page: 1, Content: config.PageContent{Answers: []config.CircleValidated{circle(10), circle(20)}}},
		{Page: 2, Content: config.PageContent{Answers: []config.CircleValidated{circle(30), circle(40)}}},
		{Page: 3, Content: config.PageContent{Answers: []config.CircleValidated{circle(50)}}},
	}
}

func writeReviewCropPNG(t *testing.T, directory, name string, width, height int) (string, []byte) {
	t.Helper()
	if err := os.MkdirAll(directory, 0o750); err != nil {
		t.Fatal(err)
	}
	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := uint8(230)
			if x >= 34 && x < 46 && y >= 34 && y < 46 {
				value = 40
			}
			img.SetGray(x, y, color.Gray{Y: value})
		}
	}
	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, output.Bytes(), 0o640); err != nil {
		t.Fatal(err)
	}
	return path, output.Bytes()
}
