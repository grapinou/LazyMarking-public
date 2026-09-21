package qcm

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/csrf"
	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	altanswers "github.com/grapinou/LazyMarking/internal/handlers/altAnswers"
	altimages "github.com/grapinou/LazyMarking/internal/handlers/altImages"
	altquestions "github.com/grapinou/LazyMarking/internal/handlers/altQuestions"
	"github.com/grapinou/LazyMarking/internal/handlers/answers"
	"github.com/grapinou/LazyMarking/internal/handlers/images"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	qcmpreview "github.com/grapinou/LazyMarking/internal/handlers/qcmPreview"
	qcmquestions "github.com/grapinou/LazyMarking/internal/handlers/qcmQuestions"
	"github.com/grapinou/LazyMarking/internal/handlers/questions"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/httpsecurity"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestSharedQCMAliceBobJourney(t *testing.T) {
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	// Snap-packaged Typst has a private /tmp. Keep this isolated runtime under
	// the repository's disposable assets/tmp so the real compiler can read it.
	base := filepath.Join(repo, "assets", "tmp")
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	workspace, err := os.MkdirTemp(base, "p62-http-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(workspace) })
	t.Chdir(workspace)
	if err := os.Symlink(filepath.Join(repo, "internal"), "internal"); err != nil {
		t.Fatal(err)
	}
	conn, _, err := db.OpenMigratedDB(context.Background(), "app.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	sql := func(query string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	scalar := func(query string, args ...any) int64 {
		t.Helper()
		var n int64
		if err := conn.QueryRow(query, args...).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	sql(`INSERT INTO users(id,username,email,hashpassword) VALUES(1,'Alice','a@example.test','hash'),(2,'Bob','b@example.test','hash')`)
	for _, table := range []string{"subjects", "themes", "year_levels", "skills", "difficulties"} {
		sql("INSERT INTO " + table + "(id,name,user_id) VALUES(1,'Physique',1)")
	}
	sql(`INSERT INTO points(id,point_value,user_id) VALUES(1,2,1);
INSERT INTO qcm(id,name,user_id) VALUES(1,'Contrôle énergie',1),(2,'QCM secret',1);
INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,content,instruction,user_id)
VALUES(1,1,1,1,1,1,1,'Privée A','Consigne privée A',1),(2,1,1,1,1,1,1,'Privée B','',1),(3,1,1,1,1,1,1,'Publiée C','',1),(4,1,1,1,1,1,1,'Hors composition','',1);
INSERT INTO answers(id,question_id,content,state,user_id) VALUES(1,1,'Réponse vraie',1,1),(2,1,'Réponse fausse',0,1),(3,2,'Réponse B',1,1),(4,3,'Réponse C',1,1);
INSERT INTO alt_questions(id,question_id,content,user_id) VALUES(1,1,'Variante privée A',1),(2,4,'Variante hors composition',1);
INSERT INTO alt_answers(alt_question_id,content,state,user_id) VALUES(1,'Bonne alternative',1,1);
INSERT INTO images(question_id,image_name,resize_percentage,user_id) VALUES(1,'main.png',50,1),(4,'outside.png',50,1);
INSERT INTO alt_images(alt_question_id,image_name,resize_percentage,user_id) VALUES(1,'alt.png',40,1),(2,'outside-alt.png',40,1);
INSERT INTO question_shares VALUES(3);
INSERT INTO qcm_questions(id,qcm_id,question_id,user_id,position) VALUES(1,1,2,1,1),(2,1,1,1,2),(3,1,3,1,3);`)
	if err := os.MkdirAll(config.ImageSavePath, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.png", "alt.png", "outside.png", "outside-alt.png"} {
		f, err := os.Create(filepath.Join(config.ImageSavePath, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	t.Setenv("SESSION_KEY", "sharing-test-key-with-32-bytes!!!")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	q := db.New(conn)
	mux := http.NewServeMux()
	RegisterSharingRoutes(mux, conn)
	RegisterRoutes(mux, q)
	questions.RegisterSharingRoutes(mux, conn)
	questions.RegisterRoutes(mux, q)
	qcmquestions.RegisterRoutes(mux, q, conn)
	qcmpreview.RegisterRoutes(mux, q)
	answers.RegisterRoutes(mux, q)
	altquestions.RegisterRoutes(mux, q)
	altanswers.RegisterRoutes(mux, q)
	images.RegisterRoutes(mux, q)
	altimages.RegisterRoutes(mux, q)
	mux.Handle("GET /static/images/{filename}", login.CheckAuth(tools.HandlerWithDB(tools.ServeUserImageHandler, q)))
	mux.HandleFunc("GET /test-csrf", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, csrf.TemplateField(r)) })
	protected := httpsecurity.NewCSRFMiddleware([]byte("0123456789abcdef0123456789abcdef"), false)(mux)
	tokenRE := regexp.MustCompile(`name="gorilla\.csrf\.Token" value="([^"]+)"`)
	request := func(user int64, method, path string, form url.Values, withCSRF bool) *httptest.ResponseRecorder {
		t.Helper()
		var cookies []*http.Cookie
		if method == "POST" && withCSRF {
			w := httptest.NewRecorder()
			protected.ServeHTTP(w, httptest.NewRequest("GET", "/test-csrf", nil))
			match := tokenRE.FindStringSubmatch(w.Body.String())
			if len(match) != 2 {
				t.Fatal("missing CSRF token")
			}
			form.Set("gorilla.csrf.Token", match[1])
			cookies = w.Result().Cookies()
		}
		r := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for _, c := range cookies {
			r.AddCookie(c)
		}
		if user != 0 {
			session, err := login.GetStore().Get(r, "session")
			if err != nil {
				t.Fatal(err)
			}
			session.Values["user_id"] = user
			session.Values["username"] = map[int64]string{1: "Alice", 2: "Bob"}[user]
			w := httptest.NewRecorder()
			if err := session.Save(r, w); err != nil {
				t.Fatal(err)
			}
			for _, c := range w.Result().Cookies() {
				r.AddCookie(c)
			}
		}
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, r)
		return w
	}
	get := func(user int64, path string, want int) string {
		t.Helper()
		w := request(user, "GET", path, nil, true)
		if w.Code != want {
			t.Fatalf("GET %s user %d: %d %s", path, user, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	post := func(user int64, path string, form url.Values, want int) {
		t.Helper()
		w := request(user, "POST", path, form, true)
		if w.Code != want {
			t.Fatalf("POST %s user %d: %d %s", path, user, w.Code, w.Body.String())
		}
	}
	sharing := func(state string) url.Values { return url.Values{"qcm_id": {"1"}, "shared": {state}} }
	copyForm := func() url.Values { return url.Values{"qcm_id": {"1"}, "user_id": {"1"}} }
	previewURL := "/dashboard/library/qcm/preview?qcm_id=1"
	imageURL := "/dashboard/library/qcm/image?qcm_id=1&question_id=1&variant_id="
	if body := get(2, "/dashboard/library/qcm", 200); !strings.Contains(body, "Aucun QCM partagé") || strings.Contains(body, "Contrôle énergie") {
		t.Fatal("QCM not private by default")
	}
	get(2, previewURL, 404)
	get(2, imageURL+"0", 404)
	post(2, "/dashboard/library/qcm/copy", copyForm(), 404)
	post(2, "/dashboard/qcm/sharing", sharing("1"), 404)
	for _, path := range []string{"/dashboard/qcm/sharing", "/dashboard/library/qcm/copy"} {
		if w := request(1, "POST", path, sharing("1"), false); w.Code != 403 {
			t.Fatalf("CSRF bypass %s: %d", path, w.Code)
		}
		get(1, path, 405)
	}
	post(1, "/dashboard/qcm/sharing", sharing("1"), 303)
	post(1, "/dashboard/qcm/sharing", sharing("1"), 303)
	if scalar("SELECT COUNT(*) FROM question_shares") != 1 {
		t.Fatal("implicit family publication")
	}
	if body := get(1, "/dashboard/qcm", 200); !strings.Contains(body, "Partagé") || !strings.Contains(body, "Retirer de la bibliothèque") {
		t.Fatal("missing personal sharing action")
	}
	if body := get(1, "/dashboard/library/qcm", 200); !strings.Contains(body, "Votre QCM") {
		t.Fatal("missing own QCM state")
	}
	if body := get(1, previewURL, 200); !strings.Contains(body, "Votre QCM") || strings.Contains(body, "Copier dans Mes QCM") || strings.Contains(body, "/qcm/copy") {
		t.Fatal("own QCM offers copying")
	}
	post(1, "/dashboard/library/qcm/copy", copyForm(), 409)
	for _, suffix := range []string{"", "?q=CONTROLE+ENERGIE", "?q=alice", "?subject=Physique&level=Physique"} {
		body := get(2, "/dashboard/library/qcm"+suffix, 200)
		for _, want := range []string{"Contrôle énergie", "Alice", "3 question(s)", "Physique"} {
			if !strings.Contains(body, want) {
				t.Fatalf("list missing %q", want)
			}
		}
		if strings.Contains(body, "QCM secret") || strings.Contains(body, "Votre QCM") {
			t.Fatal("private QCM or incorrect ownership")
		}
	}
	for _, suffix := range []string{"?q=inconnu", "?subject=inconnue", "?level=inconnu"} {
		if !strings.Contains(get(2, "/dashboard/library/qcm"+suffix, 200), "Aucun QCM ne correspond") {
			t.Fatal("filter failed")
		}
	}
	preview := get(2, previewURL, 200)
	for _, want := range []string{"Privée A", "Privée B", "Publiée C", "Consigne privée A", "Variante privée A", "Réponse vraie", "Réponse fausse", "Bonne alternative", "Lecture seule", "Copier dans Mes QCM", "gorilla.csrf.Token"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q", want)
		}
	}
	if strings.Index(preview, "Privée B") > strings.Index(preview, "Privée A") || strings.Index(preview, "Privée A") > strings.Index(preview, "Publiée C") ||
		strings.Contains(preview, "Hors composition") || strings.Contains(preview, "/questions/edit") {
		t.Fatal("preview order, membership or readonly boundary")
	}
	if body := get(2, "/dashboard/library", 200); strings.Contains(body, "Privée A") || strings.Contains(body, "Privée B") || !strings.Contains(body, "Publiée C") {
		t.Fatal("family library visibility changed")
	}
	get(2, "/dashboard/library/preview?question_id=1", 404)
	get(2, "/dashboard/library/image?question_id=1&variant_id=0", 404)
	post(2, "/dashboard/library/copy", url.Values{"question_id": {"1"}}, 404)
	get(2, "/dashboard/library/qcm/preview?qcm_id=2", 404)
	get(2, imageURL+"0", 200)
	get(2, imageURL+"1", 200)
	get(2, imageURL+"2", 404)
	get(2, "/dashboard/library/qcm/image?qcm_id=1&question_id=4&variant_id=0", 404)
	get(2, "/dashboard/library/qcm/image?qcm_id=2&question_id=1&variant_id=0", 404)
	get(2, "/static/images/main.png", 404)
	post(2, "/dashboard/qcm/sharing", sharing("0"), 404)
	post(1, "/dashboard/qcm/sharing", sharing("invalid"), 400)
	for _, id := range []string{"-1", "0", "invalid", "99999999999999999999", ""} {
		get(2, "/dashboard/library/qcm/preview?qcm_id="+id, 400)
		post(2, "/dashboard/library/qcm/copy", url.Values{"qcm_id": {id}}, 400)
	}
	// Sharing does not relax ANY normal owner-scoped reads or mutations.
	for _, path := range []string{
		"/dashboard/qcm/edit?qcm_id=1", "/dashboard/qcm/qcmquestion?qcm_id=1", "/dashboard/qcm/previewqcm?qcm_id=1",
		"/dashboard/questions/edit?question_id=1",
		"/dashboard/questions/answers/edit?question_id=1&answer_id=1",
		"/dashboard/questions/altquestions/edit?question_id=1&alt_question_id=1",
		"/dashboard/questions/altquestions/altanswers/edit?question_id=1&alt_question_id=1&alt_answer_id=1",
		"/dashboard/questions/images/edit?question_id=1",
		"/dashboard/questions/altquestions/altimages/edit?question_id=1&alt_question_id=1",
	} {
		get(2, path, 404)
	}
	for _, path := range []string{"/dashboard/qcm/edit", "/dashboard/qcm/delete", data.DefaultQCMQuestionRoutes.MoveDownURL, data.DefaultQCMQuestionRoutes.DeleteURL,
		"/dashboard/questions/edit", "/dashboard/questions/delete", "/dashboard/questions/answers/edit", "/dashboard/questions/answers/delete",
		"/dashboard/questions/altquestions/edit", "/dashboard/questions/altquestions/delete",
		"/dashboard/questions/altquestions/altanswers/edit", "/dashboard/questions/altquestions/altanswers/delete",
		"/dashboard/questions/images/edit", "/dashboard/questions/images/delete", "/dashboard/questions/altquestions/altimages/edit", "/dashboard/questions/altquestions/altimages/delete"} {
		post(2, path, url.Values{
			"qcm_id": {"1"}, "qcm_question_id": {"1"}, "new_qcm": {"Intrusion"},
			"question_id": {"1"}, "answer_id": {"1"}, "alt_question_id": {"1"}, "alt_answer_id": {"1"},
			"content": {"Intrusion"}, "new_content": {"Intrusion"}, "new_state": {"1"}, "width": {"50"},
			"subjectID": {"1"}, "themeID": {"1"}, "yearLevelID": {"1"}, "skillID": {"1"}, "difficultyID": {"1"}, "pointID": {"1"},
		}, 404)
	}
	post(2, "/dashboard/library/qcm/copy", copyForm(), 303)
	copyID := scalar("SELECT id FROM qcm WHERE user_id=2")
	copyText := strconv.FormatInt(copyID, 10)
	if body := get(2, "/dashboard/qcm?copied=1", 200); !strings.Contains(body, "Copié depuis un QCM de Alice") || !strings.Contains(body, "Privé") || !strings.Contains(body, "ont été copiés") {
		t.Fatal("missing private personal copy and provenance")
	}
	get(2, "/dashboard/qcm/edit?qcm_id="+copyText, 200)
	get(2, "/dashboard/qcm/qcmquestion?qcm_id="+copyText+"&reorder=1", 200)
	firstLink := scalar("SELECT id FROM qcm_questions WHERE qcm_id=? AND position=1", copyID)
	post(2, data.DefaultQCMQuestionRoutes.MoveDownURL, url.Values{"qcm_id": {copyText}, "qcm_question_id": {fmt.Sprint(firstLink)}, "reorder": {"1"}}, 303)
	if scalar("SELECT position FROM qcm_questions WHERE id=?", firstLink) != 2 || scalar("SELECT position FROM qcm_questions WHERE id=1") != 1 {
		t.Fatal("copied QCM cannot be independently reordered")
	}
	post(1, "/dashboard/qcm/sharing", sharing("0"), 303)
	get(2, previewURL, 404)
	get(2, imageURL+"0", 404)
	post(2, "/dashboard/library/qcm/copy", copyForm(), 404)
	get(2, "/dashboard/qcm/qcmquestion?qcm_id="+copyText, 200)
	if !strings.Contains(get(2, "/dashboard/library/qcm", 200), "Aucun QCM partagé") {
		t.Fatal("withdrawn QCM still listed")
	}
	if !strings.Contains(get(2, "/dashboard/library", 200), "Publiée C") {
		t.Fatal("withdrawal changed independent family share")
	}
	var imageName string
	if err := conn.QueryRow("SELECT image_name FROM images WHERE user_id=2").Scan(&imageName); err != nil {
		t.Fatal(err)
	}
	get(2, "/static/images/"+imageName, 200)
	get(1, "/static/images/"+imageName, 404)
	// Normal portrait AND landscape preview endpoints compile real PDFs of the copy.
	t.Run("copied QCM PDF previews", func(t *testing.T) {
		if _, err := exec.LookPath("typst"); err != nil {
			t.Skip("typst unavailable")
		}
		for _, path := range []string{data.DefaultQCMRoutes.PreviewURL, data.DefaultQCMRoutes.PreviewLandscapeURL} {
			w := request(2, "GET", path+"?qcm_id="+copyText, nil, true)
			if w.Code != 303 || !strings.Contains(w.Header().Get("Location"), "operation=") {
				t.Fatalf("preview compilation: %d %s", w.Code, w.Body.String())
			}
			pdf := get(2, w.Header().Get("Location"), 200)
			if !strings.HasPrefix(pdf, "%PDF-") {
				t.Fatal("preview not PDF")
			}
		}
	})
	post(1, "/dashboard/qcm/sharing", sharing("1"), 303)
	for _, path := range []string{"/dashboard/library/qcm", previewURL, imageURL + "0"} {
		if w := request(0, "GET", path, nil, true); w.Code != 302 {
			t.Fatalf("anonymous access: %s %d", path, w.Code)
		}
		if w := request(2, "POST", path, url.Values{}, true); w.Code != 405 {
			t.Fatalf("unexpected method accepted: %s %d", path, w.Code)
		}
	}
	if w := request(0, "POST", "/dashboard/library/qcm/copy", copyForm(), true); w.Code != 302 || w.Header().Get("Location") != "/login" {
		t.Fatalf("anonymous copy: %d %s", w.Code, w.Header().Get("Location"))
	}
	t.Run("copied QCM exam generation after source deletion", func(t *testing.T) {
		if _, err := exec.LookPath("typst"); err != nil {
			t.Skip("typst unavailable")
		}
		// Force the answerable variant so the actual generation validates it.
		mainID := scalar("SELECT question_id FROM question_copy_origins WHERE source_question_id=1")
		sql("DELETE FROM answers WHERE question_id=? AND user_id=2", mainID)
		sql("DELETE FROM qcm WHERE user_id=1; DELETE FROM questions WHERE user_id=1")
		for _, name := range []string{"main.png", "alt.png", "outside.png", "outside-alt.png"} {
			if err := os.Remove(filepath.Join(config.ImageSavePath, name)); err != nil {
				t.Fatal(err)
			}
		}
		for _, table := range []string{"class_codes", "periods", "years"} {
			sql("INSERT INTO " + table + "(id,name,user_id) VALUES(1,'Seconde',2)")
		}
		sql("INSERT INTO students(id,first_name,last_name,user_id) VALUES(1,'Jean-Baptiste','Dupont-Martin',2)")
		sql("INSERT INTO exams(id,name,qcm_id,class_code_id,period_id,year_id,user_id) VALUES(1,'Évaluation depuis la copie',?,1,1,1,2)", copyID)
		sql("INSERT INTO exams_generated(id,exam_id,total_students,user_id) VALUES(1,1,1,2)")
		exam, err := q.GetExamByID(context.Background(), db.GetExamByIDParams{ID: 1, UserID: 2})
		if err != nil {
			t.Fatal(err)
		}
		dir, ok := tools.CreateOperationTempDir("Bob", "exam-1")
		if !ok {
			t.Fatal("generation workspace")
		}
		generated, err := tools.BuildQcmStudentCtx(db.Student{ID: 1, FirstName: "Jean-Baptiste", LastName: "Dupont-Martin"}, exam, 1, 2, dir, "Bob", "Seconde", context.Background(), q)
		if err != nil || len(generated.Questions) != 3 {
			t.Fatalf("copied QCM generation: %+v %v", generated, err)
		}
		foundVariant := false
		for _, question := range generated.Questions {
			if question.Tags.MainQuestionID <= 4 || question.Circle.Radius <= 0 {
				t.Fatalf("foreign reference or missing detection: %+v", question)
			}
			if question.Tags.MainQuestionID == mainID {
				foundVariant = true
				if question.Content != "Variante privée A" || question.Instruction != "Consigne privée A" ||
					question.Image.Name == "alt.png" || question.Image.Name == "" {
					t.Fatalf("variant/instruction/image generation: %+v", question)
				}
			}
		}
		if !foundVariant {
			t.Fatal("variant missing")
		}
		stored, err := q.GetStudentContentExam(context.Background(), db.GetStudentContentExamParams{StudentExamID: 1, UserID: 2})
		if err != nil || stored.PageTot < 1 || !strings.Contains(stored.Content, "Variante privée A") {
			t.Fatalf("generated snapshot: %+v %v", stored, err)
		}
		scans, err := filepath.Glob(filepath.Join(dir, "qr_student-exam-*page-*.png"))
		if err != nil || len(scans) != int(stored.PageTot) {
			t.Fatalf("generated pages: %v %v", scans, err)
		}
		for _, scan := range scans {
			decoded, err := tools.DecodeWithGozxing(scan)
			if err != nil {
				t.Fatal(err)
			}
			var info config.QrCodeInfo
			if err := json.Unmarshal([]byte(decoded), &info); err != nil || info.StudentExamID != 1 {
				t.Fatalf("generated QR: %+v %v", info, err)
			}
		}
		t.Logf("independent exam generated after source deletion: %d pages, 3 personal families, illustrated variant, persisted snapshot and readable QR", stored.PageTot)
	})
}
