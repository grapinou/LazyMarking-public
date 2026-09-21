package questions

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/httpsecurity"
)

func TestSharedLibraryAliceBobJourney(t *testing.T) {
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if err := os.Symlink(filepath.Join(repo, "internal"), "internal"); err != nil {
		t.Fatal(err)
	}
	conn, _, err := db.OpenMigratedDB(context.Background(), "app.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := conn.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,username,email,hashpassword) VALUES(1,'Alice','alice@example.test','hash'),(2,'Bob','bob@example.test','hash')`)
	for _, table := range []string{"subjects", "themes", "year_levels", "skills", "difficulties"} {
		exec(fmt.Sprintf("INSERT INTO %s(id,name,user_id) VALUES(1,'Physique',1)", table))
	}
	exec(`INSERT INTO points(id,point_value,user_id) VALUES(1,2,1);
INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,content,instruction,user_id) VALUES(1,1,1,1,1,1,1,'Énergie Alice','Justifier chaque choix.',1),(2,1,1,1,1,1,1,'Question secrète','',1);
INSERT INTO answers(id,question_id,content,state,user_id) VALUES(1,1,'Réponse vraie',1,1),(2,1,'Réponse fausse',0,1);
INSERT INTO alt_questions(id,question_id,content,user_id) VALUES(1,1,'Circuit alternatif',1),(2,2,'Variante secrète',1);
INSERT INTO alt_answers(id,alt_question_id,content,state,user_id) VALUES(1,1,'Bonne alternative',1,1);
INSERT INTO images(question_id,image_name,resize_percentage,user_id) VALUES(1,'main.png',50,1),(2,'secret.png',50,1);
INSERT INTO alt_images(alt_question_id,image_name,resize_percentage,user_id) VALUES(1,'alt.png',40,1),(2,'secret-alt.png',40,1);`)
	if err := os.MkdirAll(config.ImageSavePath, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.png", "alt.png", "secret.png", "secret-alt.png"} {
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
	mux := http.NewServeMux()
	RegisterSharingRoutes(mux, conn)
	q := db.New(conn)
	RegisterRoutes(mux, q)
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
			r := httptest.NewRequest("GET", "/test-csrf", nil)
			w := httptest.NewRecorder()
			protected.ServeHTTP(w, r)
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
			session.Values["username"] = fmt.Sprint("user-", user)
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
	sharing := func(state string) url.Values { return url.Values{"question_id": {"1"}, "shared": {state}} }
	if body := get(2, "/dashboard/library", 200); !strings.Contains(body, "Aucune question partagée") || strings.Contains(body, "Énergie Alice") {
		t.Fatal("not private by default")
	}
	get(2, "/dashboard/library/preview?question_id=1", 404)
	get(2, "/dashboard/library/image?question_id=1&variant_id=0", 404)
	post(2, "/dashboard/library/copy", url.Values{"question_id": {"1"}}, 404)
	post(2, "/dashboard/questions/sharing", sharing("1"), 404)
	for _, path := range []string{"/dashboard/questions/sharing", "/dashboard/library/copy"} {
		if w := request(1, "POST", path, sharing("1"), false); w.Code != 403 {
			t.Fatalf("CSRF bypass: %s %d", path, w.Code)
		}
		get(1, path, 405)
	}
	post(1, "/dashboard/questions/sharing", sharing("1"), 303)
	// Idempotent publication and sharing only explicit resources.
	post(1, "/dashboard/questions/sharing", sharing("1"), 303)
	if body := get(1, "/dashboard/library", 200); !strings.Contains(body, "Votre question") {
		t.Fatal("missing own question marker")
	}
	if body := get(1, "/dashboard/library/preview?question_id=1", 200); !strings.Contains(body, "Votre question") || !strings.Contains(body, "Retour à Mes questions") || strings.Contains(body, "Copier dans Mes questions") || strings.Contains(body, "/dashboard/library/copy") {
		t.Fatal("own question offers copying")
	}
	post(1, "/dashboard/library/copy", url.Values{"question_id": {"1"}}, 409)
	for _, suffix := range []string{"", "?q=ENERGIE", "?q=circuit", "?q=justifier", "?subject=Physique&level=Physique&theme=Physique"} {
		body := get(2, "/dashboard/library"+suffix, 200)
		for _, want := range []string{"Énergie Alice", "Physique", "Alice", "1 variante"} {
			if !strings.Contains(body, want) {
				t.Fatalf("list missing %q", want)
			}
		}
		if strings.Contains(body, "Question secrète") || strings.Contains(body, "Variante secrète") {
			t.Fatal("private data leaked")
		}
	}
	if !strings.Contains(get(2, "/dashboard/library?q=inconnue", 200), "Aucune question ne correspond") {
		t.Fatal("filter failed")
	}
	preview := get(2, "/dashboard/library/preview?question_id=1", 200)
	for _, want := range []string{"Justifier chaque choix.", "Réponse vraie", "Réponse fausse", "Bonne alternative", "Circuit alternatif", "Lecture seule", "Copier dans Mes questions", "gorilla.csrf.Token"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q", want)
		}
	}
	if strings.Contains(preview, "/questions/edit") {
		t.Fatal("edit link in shared preview")
	}
	for _, variant := range []string{"0", "1"} {
		get(2, "/dashboard/library/image?question_id=1&variant_id="+variant, 200)
	}
	get(2, "/dashboard/library/image?question_id=1&variant_id=2", 404)
	get(2, "/dashboard/library/preview?question_id=2", 404)
	get(2, "/static/images/main.png", 404)
	post(2, "/dashboard/questions/sharing", sharing("0"), 404)
	post(1, "/dashboard/questions/sharing", sharing("invalid"), 400)
	post(1, "/dashboard/library/copy", url.Values{"question_id": {"-1"}}, 400)
	// All original editing endpoints keep their owner checks after publication.
	for _, path := range []string{
		"/dashboard/questions/edit?question_id=1",
		"/dashboard/questions/answers/edit?question_id=1&answer_id=1",
		"/dashboard/questions/altquestions/edit?question_id=1&alt_question_id=1",
		"/dashboard/questions/altquestions/altanswers/edit?question_id=1&alt_question_id=1&alt_answer_id=1",
		"/dashboard/questions/images/edit?question_id=1",
		"/dashboard/questions/altquestions/altimages/edit?question_id=1&alt_question_id=1",
	} {
		get(2, path, 404)
	}
	for _, path := range []string{"/dashboard/questions/delete", "/dashboard/questions/answers/delete", "/dashboard/questions/altquestions/delete", "/dashboard/questions/altquestions/altanswers/delete", "/dashboard/questions/images/delete", "/dashboard/questions/altquestions/altimages/delete"} {
		post(2, path, url.Values{"question_id": {"1"}, "answer_id": {"1"}, "alt_question_id": {"1"}, "alt_answer_id": {"1"}}, 404)
	}
	for _, path := range []string{"/dashboard/questions/edit", "/dashboard/questions/answers/edit", "/dashboard/questions/altquestions/edit", "/dashboard/questions/altquestions/altanswers/edit", "/dashboard/questions/images/edit", "/dashboard/questions/altquestions/altimages/edit"} {
		post(2, path, url.Values{
			"question_id": {"1"}, "answer_id": {"1"}, "alt_question_id": {"1"}, "alt_answer_id": {"1"},
			"content": {"Intrusion"}, "new_content": {"Intrusion"}, "new_state": {"1"}, "width": {"50"},
			"subjectID": {"1"}, "themeID": {"1"}, "yearLevelID": {"1"}, "skillID": {"1"}, "difficultyID": {"1"}, "pointID": {"1"},
		}, 404)
	}
	post(2, "/dashboard/library/copy", url.Values{"question_id": {"1"}, "user_id": {"1"}}, 303)
	var copyID int64
	if err := conn.QueryRow("SELECT id FROM questions WHERE user_id=2").Scan(&copyID); err != nil {
		t.Fatal(err)
	}
	copyText := strconv.FormatInt(copyID, 10)
	if body := get(2, "/dashboard/questions", 200); !strings.Contains(body, "Énergie Alice") || !strings.Contains(body, "Copiée depuis une question de Alice") || !strings.Contains(body, "Privée") {
		t.Fatal("copy not in personal library")
	}
	get(2, "/dashboard/questions/edit?question_id="+copyText, 200)
	// Bob can edit his new family through the normal form. It also remains
	// usable by the existing generation builders and ordered QCM composition.
	copyRow, err := q.GetQuestionByID(context.Background(), db.GetQuestionByIDParams{ID: copyID, UserID: 2})
	if err != nil {
		t.Fatal(err)
	}
	post(2, "/dashboard/questions/edit", url.Values{
		"question_id": {copyText}, "content": {"Version Bob"}, "instruction": {copyRow.Instruction},
		"subjectID": {fmt.Sprint(copyRow.SubjectID)}, "themeID": {fmt.Sprint(copyRow.ThemeID)},
		"yearLevelID": {fmt.Sprint(copyRow.YearLevelID)}, "skillID": {fmt.Sprint(copyRow.SkillID)},
		"difficultyID": {fmt.Sprint(copyRow.DifficultyID)}, "pointID": {fmt.Sprint(copyRow.PointID)},
	}, 303)
	generated, err := tools.GetQuestionAnswerCtx(2, copyID, q, context.Background())
	if err != nil || generated.Content != "Version Bob" || generated.Instruction != "Justifier chaque choix." || len(generated.Answers) != 2 || generated.Image.Name == "main.png" {
		t.Fatalf("copy generation: %+v %v", generated, err)
	}
	exec("INSERT INTO qcm(id,name,user_id) VALUES(1,'QCM Bob',2)")
	exec("INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(1,?,2,1)", copyID)
	if !strings.Contains(get(2, "/dashboard/library/preview?question_id=1", 200), "Énergie Alice") {
		t.Fatal("copy affected original")
	}
	post(1, "/dashboard/questions/sharing", sharing("0"), 303)
	if !strings.Contains(get(2, "/dashboard/library", 200), "Aucune question partagée") {
		t.Fatal("withdrawal failed")
	}
	get(2, "/dashboard/library/preview?question_id=1", 404)
	get(2, "/dashboard/library/image?question_id=1&variant_id=0", 404)
	post(2, "/dashboard/library/copy", url.Values{"question_id": {"1"}}, 404)
	if !strings.Contains(get(2, "/dashboard/questions", 200), "Version Bob") {
		t.Fatal("withdrawal lost copy")
	}
	var copiedImage string
	if err := conn.QueryRow("SELECT image_name FROM images WHERE question_id=?", copyID).Scan(&copiedImage); err != nil {
		t.Fatal(err)
	}
	get(2, "/static/images/"+copiedImage, 200)
	get(1, "/static/images/"+copiedImage, 404)
	// No unauthenticated access to shared resources, even after publication.
	post(1, "/dashboard/questions/sharing", sharing("1"), 303)
	for _, path := range []string{"/dashboard/library", "/dashboard/library/preview?question_id=1", "/dashboard/library/image?question_id=1&variant_id=0"} {
		w := request(0, "GET", path, nil, true)
		if w.Code == 200 {
			t.Fatalf("anonymous access: %s", path)
		}
	}
	post(1, "/dashboard/questions/delete", url.Values{"question_id": {"1"}}, 303)
	get(2, "/static/images/"+copiedImage, 200)
	if _, err := os.Stat(filepath.Join(config.ImageSavePath, "main.png")); !os.IsNotExist(err) {
		t.Fatalf("original file not deleted: %v", err)
	}
}
