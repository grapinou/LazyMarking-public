package marking

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func publicCopyRequest(mux *http.ServeMux, method, path, code string, cookie *http.Cookie) *httptest.ResponseRecorder {
	var body *strings.Reader
	if method == http.MethodPost {
		body = strings.NewReader(url.Values{"code": {code}}.Encode())
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestStudentCouponsRouteUsesEachPersistedAccess(t *testing.T) {
	for _, binary := range []string{"typst", "pdftotext", "pdfinfo"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("%s unavailable", binary)
		}
	}
	t.Setenv("SESSION_KEY", "p4-test-session-key-with-more-than-32-characters")
	t.Setenv("APP_BASE_URL", "https://school.example")
	f, mux := newCumulativeFixture(t)
	doc := cumulativeGET(t, mux, "/dashboard/marking/coupons/pdf?exam_generated_id=10")
	if doc.Code != 200 || !bytes.HasPrefix(doc.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("coupon PDF %d: %s", doc.Code, doc.Body.String())
	}
	if doc.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("coupon PDF cacheable")
	}
	cmd := exec.Command("pdftotext", "-layout", "-", "-")
	cmd.Stdin = bytes.NewReader(doc.Body.Bytes())
	body, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, name := range []string{"Albert Charlie", "Dupont David", "Éclair Bob", "Éclair Eva", "Zola Alice"} {
		if !strings.Contains(text, name) {
			t.Fatalf("missing coupon for %s", name)
		}
	}
	rows, err := f.queries.ListStudentCopyAccess(t.Context(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range rows {
		if !strings.Contains(text, "https://school.example/copies/"+a.Token) || !strings.Contains(text, couponCode(a.Token)) {
			t.Fatalf("coupon mismatch for copy %d", a.StudentExamID)
		}
	}
	if got := cumulativeGET(t, mux, "/dashboard/marking/coupons/pdf?exam_generated_id=20"); got.Code != 404 {
		t.Fatalf("foreign coupons status=%d", got.Code)
	}
}

func TestStudentCopyAccessLifecycleAndCatchUp(t *testing.T) {
	t.Setenv("SESSION_KEY", "p4-test-session-key-with-more-than-32-characters")
	t.Setenv("APP_BASE_URL", "https://school.example")
	f, mux := newCumulativeFixture(t)
	if err := ensureStudentAccess(t.Context(), f.queries, 1, 10); err != nil {
		t.Fatal(err)
	}
	accesses, err := f.queries.ListStudentCopyAccess(t.Context(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureStudentAccess(t.Context(), f.queries, 1, 10); err != nil {
		t.Fatal(err)
	}
	reprinted, err := f.queries.ListStudentCopyAccess(t.Context(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	for i := range accesses {
		if accesses[i].Token != reprinted[i].Token {
			t.Fatal("coupon regeneration changed a public token")
		}
	}
	if len(accesses) != 5 {
		t.Fatalf("access count=%d", len(accesses))
	}
	var alice, absent db.StudentCopyAccess
	for _, a := range accesses {
		if a.StudentExamID == 100 {
			alice = a
		}
		if a.StudentExamID == 106 {
			absent = a
		}
	}
	if len(alice.Token) != 43 || alice.Token == absent.Token || strings.Contains(alice.Token, "100") {
		t.Fatal("token not opaque and individual")
	}
	if alice.CodeHash == couponCode(alice.Token) {
		t.Fatal("plaintext code persisted")
	}
	if got := publicCopyRequest(mux, "GET", "/copies/unknown", "", nil); got.Code != 404 {
		t.Fatalf("unknown token %d", got.Code)
	}
	path := "/copies/" + alice.Token
	if got := publicCopyRequest(mux, "GET", path, "", nil); got.Code != 200 || !strings.Contains(got.Body.String(), "Code personnel") || strings.Contains(got.Body.String(), "Alice") {
		t.Fatal("first scan leaked identity or did not challenge")
	}
	if got := publicCopyRequest(mux, "POST", path, "WRONG-WRONG", nil); got.Code != 200 || !strings.Contains(got.Body.String(), "Code incorrect") {
		t.Fatal("bad code accepted")
	}
	code := couponCode(alice.Token)
	got := publicCopyRequest(mux, "POST", path, code, nil)
	if got.Code != 200 || !strings.Contains(got.Body.String(), "pas encore disponible") {
		t.Fatalf("unpublished response: %d %s", got.Code, got.Body.String())
	}
	var session *http.Cookie
	for _, c := range got.Result().Cookies() {
		if c.Name == studentCopyCookie {
			session = c
		}
	}
	if session == nil {
		t.Fatal("correct code did not establish scoped session")
	}
	if got := publicCopyRequest(mux, "GET", path+"/pdf", "", session); got.Code != 404 {
		t.Fatalf("unpublished PDF status=%d", got.Code)
	}
	if got := publicCopyRequest(mux, "GET", "/copies/"+absent.Token+"/pdf", "", session); got.Code != 404 {
		t.Fatal("session crossed into another copy")
	}
	begin := time.Now().UTC().Truncate(time.Second)
	n, err := publishFinalStudentCopies(t.Context(), f.queries, 1, 10, begin)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("published %d, want 3 corrected copies", n)
	}
	alice, err = f.queries.GetStudentCopyAccessByToken(t.Context(), alice.Token)
	if err != nil {
		t.Fatal(err)
	}
	if alice.State != "published" || !alice.ExpiresAt.Time.Equal(begin.Add(14*24*time.Hour)) {
		t.Fatalf("publication=%+v", alice)
	}
	if got := publicCopyRequest(mux, "GET", path, "", session); !strings.Contains(got.Body.String(), "disponible jusqu&#39;au") {
		t.Fatalf("published page missing: %s", got.Body.String())
	}
	if got := publicCopyRequest(mux, "GET", "/copies/"+absent.Token, "", nil); strings.Contains(got.Body.String(), "Eva") {
		t.Fatal("absent identity disclosed")
	}
	// The absent pupil's initial coupon remains valid. Their window starts only
	// after the later import is corrected and explicitly published.
	if _, err := f.conn.Exec(`INSERT INTO marking_jobs(id,user_id,status,status_pdf,review_revision,artifacts_revision,exam_generated_id) VALUES(60,1,'success','success',0,0,10);
 INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,expected_pages,detected_pages,score_half_units,total_points)
 VALUES(6000,1,60,106,'corrected',1,1,2,1)`); err != nil {
		t.Fatal(err)
	}
	later := begin.Add(20 * 24 * time.Hour)
	n, err = publishFinalStudentCopies(t.Context(), f.queries, 1, 10, later)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("catch-up published %d", n)
	}
	absent, err = f.queries.GetStudentCopyAccessByToken(t.Context(), absent.Token)
	if err != nil {
		t.Fatal(err)
	}
	if !absent.ExpiresAt.Time.Equal(later.Add(14 * 24 * time.Hour)) {
		t.Fatalf("catch-up expiry=%v", absent.ExpiresAt)
	}
	if n, err := f.queries.ChangeStudentCopyAccess(t.Context(), 1, 10, 100, "revoked", later); err != nil || n != 1 {
		t.Fatalf("revoke %d %v", n, err)
	}
	if got := publicCopyRequest(mux, "GET", path+"/pdf", "", session); got.Code != 404 {
		t.Fatal("revoked PDF accessible")
	}
	if n, err := f.queries.ChangeStudentCopyAccess(t.Context(), 1, 10, 100, "published", later); err != nil || n != 1 {
		t.Fatalf("reopen %d %v", n, err)
	}
	alice, err = f.queries.GetStudentCopyAccessByToken(t.Context(), alice.Token)
	if err != nil {
		t.Fatal(err)
	}
	if !alice.ExpiresAt.Time.Equal(later.Add(14 * 24 * time.Hour)) {
		t.Fatal("reopened window incorrect")
	}
	if _, err := f.conn.Exec(`UPDATE student_copy_access SET expires_at=? WHERE student_exam_id=100`, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if got := publicCopyRequest(mux, "GET", path+"/pdf", "", session); got.Code != 404 {
		t.Fatal("exact expiry failed closed")
	}
	if got := publicCopyRequest(mux, "GET", path, "", session); !strings.Contains(got.Body.String(), "a expiré") {
		t.Fatal("expiration message missing")
	}
	if n, err := f.queries.ChangeStudentCopyAccess(t.Context(), 1, 10, 100, "published", later.Add(30*24*time.Hour)); err != nil || n != 1 {
		t.Fatalf("reopen expired access: %d %v", n, err)
	}
	if n, err := f.queries.ChangeStudentCopyAccess(t.Context(), 2, 10, 100, "revoked", later); err != nil || n != 0 {
		t.Fatal("foreign teacher changed access")
	}
}

func TestStudentCopyCodeAttemptLimit(t *testing.T) {
	t.Setenv("SESSION_KEY", "p4-test-session-key-with-more-than-32-characters")
	f, mux := newCumulativeFixture(t)
	if err := ensureStudentAccess(t.Context(), f.queries, 1, 10); err != nil {
		t.Fatal(err)
	}
	rows, err := f.queries.ListStudentCopyAccess(t.Context(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	path := "/copies/" + rows[0].Token
	for i := 0; i < 5; i++ {
		publicCopyRequest(mux, "POST", path, "BADCODEBADCODE", nil)
	}
	if got := publicCopyRequest(mux, "POST", path, couponCode(rows[0].Token), nil); strings.Contains(got.Body.String(), "pas encore disponible") {
		t.Fatal("correct code bypassed lock")
	}
	if _, err := f.conn.Exec(`UPDATE student_copy_access SET locked_until=? WHERE student_exam_id=?`, time.Now().Add(-time.Second), rows[0].StudentExamID); err != nil {
		t.Fatal(err)
	}
	if got := publicCopyRequest(mux, "POST", path, couponCode(rows[0].Token), nil); !strings.Contains(got.Body.String(), "pas encore disponible") {
		t.Fatalf("unlock failed: %s", got.Body.String())
	}
}

func TestStudentCopyPublicationRequiresFinalAndOwner(t *testing.T) {
	t.Setenv("SESSION_KEY", "p4-test-session-key-with-more-than-32-characters")
	f, _ := newCumulativeFixture(t)
	if err := ensureStudentAccess(context.Background(), f.queries, 1, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := publishFinalStudentCopies(t.Context(), f.queries, 2, 10, time.Now()); err == nil {
		t.Fatal("foreign teacher published")
	}
	if _, err := publishFinalStudentCopies(t.Context(), f.queries, 1, 10, time.Now()); err != nil {
		t.Fatal(err)
	}
	rows, err := f.queries.ListStudentCopyAccess(t.Context(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range rows {
		if a.StudentExamID == 104 || a.StudentExamID == 106 {
			if a.State != "unpublished" {
				t.Fatalf("nonfinal copy %d published", a.StudentExamID)
			}
		}
	}
	if n, err := f.queries.ChangeStudentCopyAccess(t.Context(), 1, 10, 104, "revoked", time.Now()); err != nil || n != 1 {
		t.Fatalf("revoke unpublished copy: %d %v", n, err)
	}
	if n, err := publishFinalStudentCopies(t.Context(), f.queries, 1, 10, time.Now()); err != nil || n != 0 {
		t.Fatalf("bulk publication changed revoked/nonfinal copies: %d %v", n, err)
	}
}

func TestStudentCopyTeacherActionsStayInGeneration(t *testing.T) {
	t.Setenv("SESSION_KEY", "p4-test-session-key-with-more-than-32-characters")
	f, mux := newCumulativeFixture(t)
	page := cumulativeGET(t, mux, "/dashboard/marking/results?exam_generated_id=10")
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Rendre les copies corrigées disponibles") || !strings.Contains(page.Body.String(), "Télécharger les coupons élèves") {
		t.Fatal("teacher access controls missing")
	}
	post := func(values url.Values) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/dashboard/marking/student-access", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(markingSessionCookie(t, req))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	if got := post(url.Values{"exam_generated_id": {"20"}, "action": {"publish"}}); got.Code != 404 {
		t.Fatalf("foreign publish=%d", got.Code)
	}
	if got := post(url.Values{"exam_generated_id": {"10"}, "action": {"publish"}}); got.Code != http.StatusSeeOther {
		t.Fatalf("publish=%d: %s", got.Code, got.Body.String())
	}
	rows, err := f.queries.ListStudentCopyAccess(t.Context(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	var published, nonfinal int64
	for _, a := range rows {
		if a.StudentExamID == 100 {
			published = a.StudentExamID
			if a.State != "published" {
				t.Fatal("final copy not published")
			}
		}
		if a.StudentExamID == 104 {
			nonfinal = a.StudentExamID
			if a.State != "unpublished" {
				t.Fatal("nonfinal copy published")
			}
		}
	}
	if got := post(url.Values{"exam_generated_id": {"10"}, "student_exam_id": {fmt.Sprint(nonfinal)}, "action": {"reopen"}}); got.Code != 409 {
		t.Fatalf("nonfinal reopen=%d", got.Code)
	}
	if got := post(url.Values{"exam_generated_id": {"10"}, "student_exam_id": {fmt.Sprint(published)}, "action": {"revoke"}}); got.Code != http.StatusSeeOther {
		t.Fatalf("revoke=%d", got.Code)
	}
	if got := post(url.Values{"exam_generated_id": {"10"}, "student_exam_id": {fmt.Sprint(published)}, "action": {"reopen"}}); got.Code != http.StatusSeeOther {
		t.Fatalf("reopen=%d", got.Code)
	}
}

func TestStudentCopyPublicPDFRequiresOwnPublishedCode(t *testing.T) {
	if _, err := exec.LookPath("pdfunite"); err != nil {
		t.Skip("pdfunite unavailable")
	}
	t.Setenv("SESSION_KEY", "p4-test-session-key-with-more-than-32-characters")
	f, mux := newCumulativeFixture(t)
	t.Chdir(t.TempDir())
	if _, err := f.conn.Exec(`DROP TABLE marking_aligned_pages;
 CREATE TABLE marking_aligned_pages(id INTEGER PRIMARY KEY,user_id INTEGER,copy_result_id INTEGER,page_exam INTEGER,storage_key TEXT,width INTEGER,height INTEGER,sha256 TEXT,created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
 INSERT INTO marking_question_results(id,copy_result_id,question_index,state,score_half_units,total_points) VALUES(630,503,0,'correct',2,1);
 INSERT INTO marking_answer_detections(id,question_result_id,answer_index,detected_state,mean_gray,automatic_state) VALUES(730,630,0,1,220,1)`); err != nil {
		t.Fatal(err)
	}
	snapshot := config.QCM{Name: "Masse volumique", Student: config.StudentQCM{FirstName: "Charlie", LastName: "Albert", ClassCodes: config.ClassCode{Name: "2nde 3"}}, Questions: []config.Question{{Content: "Réponse individuelle", Tags: config.Tags{Point: config.Point{PointValue: 1}}, Answers: []config.Answer{{Content: "Exact", State: 1}}}}}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	page := config.PageContent{Questions: []config.CircleValidated{{Position: config.Position{X: 150, Y: 80}, Radius: 12}}, Answers: []config.CircleValidated{{Position: config.Position{X: 220, Y: 150}, Radius: 12}}}
	pageJSON, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.conn.Exec(`INSERT INTO student_exam_content VALUES(103,1,?);INSERT INTO student_exam_page_content VALUES(103,1,?,1)`, string(snapshotJSON), string(pageJSON)); err != nil {
		t.Fatal(err)
	}
	workspace, ok := tools.CreateOperationTempDir("teacherone", "marking-50")
	if !ok {
		t.Fatal("marking workspace")
	}
	key := "aligned/student-exam-103/page-1.png"
	path := filepath.Join(workspace, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	img := image.NewGray(image.Rect(0, 0, 320, 460))
	for y := 0; y < 460; y++ {
		for x := 0; x < 320; x++ {
			img.SetGray(x, y, color.Gray{Y: 230})
		}
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
	imageBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(imageBytes)
	if _, err := f.conn.Exec(`INSERT INTO marking_aligned_pages(user_id,copy_result_id,page_exam,storage_key,width,height,sha256) VALUES(1,503,1,?,320,460,?)`, key, fmt.Sprintf("%x", digest)); err != nil {
		t.Fatal(err)
	}
	if err := ensureStudentAccess(t.Context(), f.queries, 1, 10); err != nil {
		t.Fatal(err)
	}
	accesses, err := f.queries.ListStudentCopyAccess(t.Context(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	var target, other db.StudentCopyAccess
	for _, a := range accesses {
		if a.StudentExamID == 103 {
			target = a
		}
		if a.StudentExamID == 100 {
			other = a
		}
	}
	if _, err := publishFinalStudentCopies(t.Context(), f.queries, 1, 10, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	pathURL := "/copies/" + target.Token
	if got := publicCopyRequest(mux, "GET", pathURL+"/pdf", "", nil); got.Code != 404 {
		t.Fatal("unauthenticated PDF accessible")
	}
	auth := publicCopyRequest(mux, "POST", pathURL, couponCode(target.Token), nil)
	var session *http.Cookie
	for _, c := range auth.Result().Cookies() {
		if c.Name == studentCopyCookie {
			session = c
		}
	}
	if session == nil {
		t.Fatal("no session")
	}
	if got := publicCopyRequest(mux, "GET", "/copies/"+other.Token+"/pdf", "", session); got.Code != 404 {
		t.Fatal("cross-copy PDF accessible")
	}
	download := publicCopyRequest(mux, "GET", pathURL+"/pdf", "", session)
	if download.Code != 200 || !bytes.HasPrefix(download.Body.Bytes(), []byte("%PDF-")) {
		t.Fatalf("PDF %d: %s", download.Code, download.Body.String())
	}
	if download.Header().Get("Cache-Control") != "no-store, private" {
		t.Fatal("personal PDF cacheable")
	}
	if _, err := f.queries.ChangeStudentCopyAccess(t.Context(), 1, 10, 103, "revoked", time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := publicCopyRequest(mux, "GET", pathURL+"/pdf", "", session); got.Code != 404 {
		t.Fatal("revoked PDF accessible")
	}
}
