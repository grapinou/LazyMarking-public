package training

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	htmlstd "html"
	stdpng "image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/httpsecurity"
	"github.com/makiuchi-d/gozxing"
	zxingqrcode "github.com/makiuchi-d/gozxing/qrcode"
)

var csrfField = regexp.MustCompile(`name="gorilla\.csrf\.Token" value="([^"]+)"`)

type trainingFixture struct {
	conn    *sql.DB
	q       *db.Queries
	handler http.Handler
	session *http.Cookie
	csrf    *http.Cookie
	token   string
}

func fixture(t *testing.T) *trainingFixture {
	t.Helper()
	t.Chdir("../../..")
	t.Setenv("SESSION_KEY", "p5-test-session-key-at-least-32-bytes")
	t.Setenv("SESSION_SECURE", "false")
	t.Setenv("APP_BASE_URL", "http://127.0.0.1:8080")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	conn, _, err := db.OpenMigratedDB(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	q := db.New(conn)
	for _, statement := range []string{
		`INSERT INTO users(id,username,email,hashpassword) VALUES(1,'teacher','t@example.test','x'),(2,'other','o@example.test','x')`,
		`INSERT INTO class_codes(id,name,user_id) VALUES(1,'Classe test',1),(2,'Classe autre',2)`,
		`INSERT INTO qcm(id,name,user_id) VALUES(1,'QCM test',1),(2,'QCM autre',2)`,
		`INSERT INTO years(id,name,user_id) VALUES(1,'2026',1),(2,'2027',2)`,
		`INSERT INTO periods(id,name,user_id) VALUES(1,'P1',1),(2,'P2',2)`,
		`INSERT INTO exams(id,name,qcm_id,class_code_id,period_id,year_id,user_id) VALUES(1,'Masse volumique',1,1,1,1,1),(2,'Autre sujet',2,2,2,2,2)`,
		`INSERT INTO exams_generated(id,exam_id,total_students,status,user_id) VALUES(10,1,2,'success',1),(20,2,0,'success',2)`,
		`INSERT INTO students(id,first_name,last_name,user_id) VALUES(1,'Alice','Privée',1),(2,'Bob','Privé',1)`,
		`INSERT INTO student_exam(id,exam_generated_id,student_id,user_id) VALUES(100,10,1,1),(101,10,2,1)`,
		`INSERT INTO subjects(id,name,user_id) VALUES(1,'Physique',1)`,
		`INSERT INTO themes(id,name,user_id) VALUES(1,'Masse volumique',1)`,
		`INSERT INTO year_levels(id,name,user_id) VALUES(1,'Quatrième',1)`,
		`INSERT INTO skills(id,name,user_id) VALUES(1,'Calculer',1)`,
		`INSERT INTO difficulties(id,name,user_id) VALUES(1,'Moyen',1)`,
		`INSERT INTO points(id,point_value,user_id) VALUES(1,1,1)`,
		`INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,content,user_id) VALUES(1,1,1,1,1,1,1,'Calculer la masse volumique',1),(2,1,1,1,1,1,1,'Calculer une énergie',1)`,
		`INSERT INTO alt_questions(id,question_id,content,user_id) VALUES(3,1,'Variante : masse de 78 g et volume de 60 mL ?',1)`,
		`INSERT INTO alt_answers(alt_question_id,state,content,user_id) VALUES(3,1,'60 mL',1),(3,0,'78 mL',1)`,
		`INSERT INTO marking_jobs(id,user_id,status,status_pdf,exam_generated_id) VALUES(50,1,'success','success',10)`,
		`INSERT INTO marking_copy_results(id,user_id,marking_job_id,student_exam_id,outcome,expected_pages,detected_pages,score_half_units,total_points) VALUES(500,1,50,100,'corrected',1,1,0,2),(501,1,50,101,'corrected',1,1,0,2)`,
		`INSERT INTO marking_question_results(id,copy_result_id,question_index,state,score_half_units,total_points) VALUES(5000,500,0,'incorrect',0,1),(5001,500,1,'incorrect',0,1),(5010,501,0,'incorrect',0,1),(5011,501,1,'incorrect',0,1)`,
	} {
		if _, err := conn.Exec(statement); err != nil {
			t.Fatalf("seed %q: %v", statement, err)
		}
	}
	first := config.Question{Tags: config.Tags{MainQuestionID: 1, VariantType: config.MainQuestion, VariantID: 1, Theme: config.Theme{ID: 1, Name: "Masse volumique"}, Skill: config.Skill{ID: 1, Name: "Calculer"}, Point: config.Point{PointValue: 1}}, Content: `Quelle relation ? $rho = m / V$`, Answers: []config.Answer{{Content: `$V = 60 "mL"$`, State: 1}, {Content: "Masse / volume", State: 1}, {Content: "Volume / masse", State: 0}}}
	second := config.Question{Tags: config.Tags{MainQuestionID: 2, VariantType: config.MainQuestion, VariantID: 2, Theme: config.Theme{ID: 1, Name: "Masse volumique"}, Skill: config.Skill{ID: 1, Name: "Calculer"}, Point: config.Point{PointValue: 1}}, Content: `Calculer $E_c = 1/2 m v^2$`, Answers: []config.Answer{{Content: `$sqrt(x)$`, State: 1}, {Content: "x²", State: 0}}}
	variant := first
	variant.Tags.VariantType = config.AltQuestion
	variant.Tags.VariantID = 3
	variant.Content = "Variante : masse de 78 g et volume de 60 mL ?"
	for i, questions := range [][]config.Question{{first, second}, {variant, second}} {
		raw, _ := json.Marshal(config.QCM{Name: "Masse volumique", Student: config.StudentQCM{FirstName: "Secret", LastName: "Élève"}, Questions: questions})
		if _, err := conn.Exec(`INSERT INTO student_exam_content(student_exam_id,page_tot,content,user_id) VALUES(?,1,?,1)`, 100+i, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, q, conn)
	protected := httpsecurity.NewCSRFMiddleware([]byte("0123456789abcdef0123456789abcdef"), false)(mux)
	f := &trainingFixture{conn: conn, q: q, handler: protected}
	request := httptest.NewRequest("GET", "/dashboard/training", nil)
	session, err := login.GetStore().Get(request, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	session.Values["username"] = "teacher"
	response := httptest.NewRecorder()
	if err := session.Save(request, response); err != nil {
		t.Fatal(err)
	}
	f.session = response.Result().Cookies()[0]
	page := f.req("GET", "/dashboard/training", nil, true)
	if page.Code != 200 {
		t.Fatalf("list status %d: %s", page.Code, page.Body.String())
	}
	match := csrfField.FindStringSubmatch(page.Body.String())
	if len(match) < 2 {
		t.Fatal("CSRF field missing")
	}
	f.token = match[1]
	for _, cookie := range page.Result().Cookies() {
		if cookie.Name == "_lazymarking_csrf" {
			f.csrf = cookie
		}
	}
	if f.csrf == nil {
		t.Fatal("CSRF cookie missing")
	}
	return f
}
func (f *trainingFixture) req(method, path string, body url.Values, teacher bool) *httptest.ResponseRecorder {
	var data *strings.Reader
	if body == nil {
		data = strings.NewReader("")
	} else {
		data = strings.NewReader(body.Encode())
	}
	r := httptest.NewRequest(method, path, data)
	if body != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if teacher && f.session != nil {
		r.AddCookie(f.session)
	}
	if f.csrf != nil {
		r.AddCookie(f.csrf)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}
func (f *trainingFixture) post(path string, form url.Values) *httptest.ResponseRecorder {
	form.Set("gorilla.csrf.Token", f.token)
	return f.req("POST", path, form, true)
}
func createDeck(t *testing.T, f *trainingFixture) (int64, string) {
	t.Helper()
	r := f.post("/dashboard/training", url.Values{"exam_generated_id": {"10"}})
	if r.Code != 303 {
		t.Fatalf("create %d: %s", r.Code, r.Body.String())
	}
	var id int64
	if _, err := fmt.Sscanf(r.Header().Get("Location"), "/dashboard/training/%d", &id); err != nil {
		t.Fatal(err)
	}
	d, err := f.q.GetTrainingDeck(context.Background(), db.GetTrainingDeckParams{ID: id, UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	return id, d.PublicToken
}

func TestCollectiveTrainingFlow(t *testing.T) {
	f := fixture(t)
	if r := f.post("/dashboard/training", url.Values{"exam_generated_id": {"20"}}); r.Code != 404 {
		t.Fatalf("foreign generation status %d", r.Code)
	}
	id, secret := createDeck(t, f)
	if len(secret) != 43 || secret == strconv.FormatInt(id, 10) {
		t.Fatalf("predictable token %q", secret)
	}
	cards, err := f.q.ListTrainingCards(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 3 {
		t.Fatalf("candidate variants=%d", len(cards))
	}
	for _, c := range cards {
		if c.Selected != 1 {
			t.Fatalf("P3 weak candidate not selected: %+v", c)
		}
	}
	publicPath := "/train/" + secret
	if r := f.req("GET", publicPath, nil, false); r.Code != 404 {
		t.Fatalf("draft public status=%d", r.Code)
	}
	save := url.Values{"action": {"save"}, "title": {"Entraînement — Masse volumique"}}
	for i, c := range cards {
		save.Set(fmt.Sprintf("order_%d", c.ID), strconv.Itoa(len(cards)-i))
		if i != 0 {
			save.Set(fmt.Sprintf("card_%d", c.ID), "on")
		}
	}
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), save); r.Code != 303 {
		t.Fatalf("save %d: %s", r.Code, r.Body.String())
	}
	cards, err = f.q.ListTrainingCards(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if cards[2].Selected != 0 {
		t.Fatalf("reordering or removal failed: %+v", cards)
	}
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), url.Values{"action": {"publish"}}); r.Code != 303 {
		t.Fatalf("publish %d: %s", r.Code, r.Body.String())
	}
	if _, err := f.conn.Exec(`UPDATE alt_questions SET content='Question modifiée' WHERE id=3`); err != nil {
		t.Fatal(err)
	}
	page := f.req("GET", publicPath, nil, false)
	if page.Code != 200 {
		t.Fatalf("public %d: %s", page.Code, page.Body.String())
	}
	html := page.Body.String()
	publicCSRF := regexp.MustCompile(`const csrf="([^"]+)"`).FindStringSubmatch(html)
	if len(publicCSRF) < 2 {
		t.Fatal("public page lacks CSRF token for answer validation")
	}
	f.token = publicCSRF[1]
	for _, forbidden := range []string{"Alice", "Privée", "Bob", "Élève", "student_exam_id", "copies/", "Question modifiée", "correct\":true"} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("public page leaks %q", forbidden)
		}
	}
	for _, wanted := range []string{"Entraînement — Masse volumique", "data:image/svg+xml;base64,", "Question ${index+1} /", "Recommencer", "Bonne réponse"} {
		if !strings.Contains(html, wanted) {
			t.Fatalf("public page lacks %q", wanted)
		}
	}
	for _, formula := range []string{`rho = m / V`, `E_c = 1/2 m v^2`, `V = 60`, `sqrt(x)`} {
		if !strings.Contains(html, formula) {
			t.Fatalf("rendered math missing %q", formula)
		}
	}
	detail := f.req("GET", fmt.Sprintf("/dashboard/training/%d", id), nil, true)
	if detail.Code != 200 {
		t.Fatalf("detail %d: %s", detail.Code, detail.Body.String())
	}
	link, _ := publicURL(secret)
	if !strings.Contains(detail.Body.String(), link) {
		t.Fatal("public link missing")
	}
	if !strings.Contains(detail.Body.String(), "data:image/png;base64,") {
		t.Fatalf("QR image missing from teacher page: %s", detail.Body.String())
	}
	encodedQR := regexp.MustCompile(`data:image/png;base64,([^"]+)`).FindStringSubmatch(detail.Body.String())
	if len(encodedQR) != 2 {
		t.Fatal("QR data missing")
	}
	pngData, err := base64.StdEncoding.DecodeString(htmlstd.UnescapeString(encodedQR[1]))
	if err != nil {
		t.Fatal(err)
	}
	qrImage, err := stdpng.Decode(bytes.NewReader(pngData))
	if err != nil {
		t.Fatal(err)
	}
	bitmap, err := gozxing.NewBinaryBitmapFromImage(qrImage)
	if err != nil {
		t.Fatal(err)
	}
	// This is an exact generated PNG. Avoid the perspective detector used for
	// scanned pages, which intermittently misses otherwise valid QR matrices.
	decoded, err := zxingqrcode.NewQRCodeReader().Decode(bitmap, map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_PURE_BARCODE: true,
	})
	if err != nil || decoded.String() != link {
		t.Fatalf("QR decode mismatch: %v", err)
	}
	var beforeChanges int64
	if err := f.conn.QueryRow(`SELECT total_changes()`).Scan(&beforeChanges); err != nil {
		t.Fatal(err)
	}
	selectedCard := cards[1]
	correct := postCheck(t, f, publicPath, selectedCard.ID, []int{0, 1})
	if !correct.Correct {
		t.Fatalf("correct answer scored wrong: %+v", correct)
	}
	if postCheck(t, f, publicPath, selectedCard.ID, []int{0}).Correct {
		t.Fatal("partial multiple selection accepted")
	}
	if postCheck(t, f, publicPath, selectedCard.ID, []int{0, 1, 2}).Correct {
		t.Fatal("extra wrong choice accepted")
	}
	var afterChanges int64
	if err := f.conn.QueryRow(`SELECT total_changes()`).Scan(&afterChanges); err != nil {
		t.Fatal(err)
	}
	if afterChanges != beforeChanges {
		t.Fatalf("public answers wrote %d database changes", afterChanges-beforeChanges)
	}
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), url.Values{"action": {"close"}}); r.Code != 303 {
		t.Fatalf("close %d", r.Code)
	}
	if r := f.req("GET", publicPath, nil, false); r.Code != 404 {
		t.Fatalf("closed public status=%d", r.Code)
	}
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), url.Values{"action": {"reopen"}}); r.Code != 303 {
		t.Fatalf("reopen %d", r.Code)
	}
	if r := f.req("GET", publicPath, nil, false); r.Code != 200 {
		t.Fatalf("reopened public status=%d", r.Code)
	}
	if r := f.req("GET", "/train/unknown", nil, false); r.Code != 404 {
		t.Fatalf("unknown token status=%d", r.Code)
	}
	var count int
	if err := f.conn.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name LIKE 'training_%'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("unexpected tracking tables=%d", count)
	}
}
func postCheck(t *testing.T, f *trainingFixture, path string, id int64, selected []int) checkResult {
	t.Helper()
	payload, _ := json.Marshal(checkRequest{CardID: id, Selected: selected})
	r := httptest.NewRequest("POST", path+"/check", bytes.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", f.token)
	r.AddCookie(f.csrf)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("check %d: %s", w.Code, w.Body.String())
	}
	var result checkResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func TestEvaluationAndRendering(t *testing.T) {
	c := cardContent{Question: "Choisir", Answers: []cardAnswer{{"A", true}, {"B", false}, {"C", true}}}
	for _, tc := range []struct {
		selected []int
		want     bool
	}{{[]int{0, 2}, true}, {[]int{0}, false}, {[]int{0, 1, 2}, false}, {[]int{1}, false}} {
		got, err := evaluate(c, tc.selected)
		if err != nil || got.Correct != tc.want {
			t.Fatalf("evaluate(%v)=%+v %v", tc.selected, got, err)
		}
	}
	unique := cardContent{Question: "Unique", Answers: []cardAnswer{{"A", true}, {"B", false}}}
	if got, _ := evaluate(unique, []int{0}); !got.Correct {
		t.Fatal("unique correct failed")
	}
	if got, _ := evaluate(unique, []int{1}); got.Correct {
		t.Fatal("unique incorrect accepted")
	}
	for _, s := range []string{`$rho = m / V$`, `$E_c = 1/2 m v^2$`, `$V = 60 "mL"$`, `$sqrt(x)$`} {
		html, err := tools.RenderTrainingText(context.Background(), "Calculer "+s)
		if err != nil || !strings.Contains(string(html), "data:image/svg+xml;base64,") {
			t.Fatalf("render %q: %v", s, err)
		}
	}
	if _, err := tools.RenderTrainingText(context.Background(), `$#read("secret")$`); err == nil {
		t.Fatal("unsafe math accepted")
	}
	if _, err := tools.RenderTrainingText(context.Background(), `$sqrt($`); err == nil {
		t.Fatal("unrenderable math accepted")
	}
}

func TestPublicTrainingInChromium(t *testing.T) {
	browser, err := exec.LookPath("chromium")
	if err != nil {
		t.Skip("Chromium unavailable")
	}
	f := fixture(t)
	id, secret := createDeck(t, f)
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), url.Values{"action": {"publish"}}); r.Code != 303 {
		t.Fatalf("publish status %d: %s", r.Code, r.Body.String())
	}
	inject := `<script>(async()=>{try{document.body.dataset.browserNarrow=String(innerWidth<=500);document.body.dataset.browserOverflow=String(document.documentElement.scrollWidth>innerWidth);document.getElementById('start').click();for(let i=0;i<cards.length;i++){const inputs=[...document.querySelectorAll('#choices input')];inputs[0].checked=true;if(cards[i].multiple)inputs[1].checked=true;document.getElementById('validate').click();let ready=false;for(let n=0;n<100;n++){await new Promise(resolve=>setTimeout(resolve,50));if(!document.getElementById('next').classList.contains('hidden')){ready=true;break}}if(!ready)throw Error('validation unavailable: '+document.getElementById('message').textContent);document.getElementById('next').click()}document.body.dataset.browserScore=document.getElementById('score').textContent;document.getElementById('restart').click();document.body.dataset.browserRestart=document.getElementById('progress').textContent}catch(e){document.body.dataset.browserError=String(e)}})()</script>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/train/"+secret {
			rec := httptest.NewRecorder()
			f.handler.ServeHTTP(rec, r)
			for key, values := range rec.Header() {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(rec.Code)
			body := bytes.Replace(rec.Body.Bytes(), []byte("</body>"), []byte(inject+"</body>"), 1)
			w.Write(body)
			return
		}
		f.handler.ServeHTTP(w, r)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, browser, "--headless", "--no-sandbox", "--no-proxy-server", "--disable-gpu", "--disable-dev-shm-usage", "--window-size=390,844", "--virtual-time-budget=12000", "--dump-dom", server.URL+"/train/"+secret).CombinedOutput()
	if err != nil {
		t.Fatalf("Chromium: %v: %s", err, output)
	}
	if !bytes.Contains(output, []byte(`data-browser-score="3 / 3 réponses correctes — 100 %"`)) || !bytes.Contains(output, []byte(`data-browser-restart="Question 1 / 3"`)) {
		t.Fatalf("browser flow did not finish: %s", output)
	}
	if !bytes.Contains(output, []byte(`data-browser-narrow="true"`)) || !bytes.Contains(output, []byte(`data-browser-overflow="false"`)) {
		t.Fatal("narrow browser viewport overflows or was not applied")
	}
}

func TestNoP3WeaknessAndPublishValidation(t *testing.T) {
	f := fixture(t)
	if _, err := f.conn.Exec(`UPDATE marking_question_results SET state='correct',score_half_units=2; UPDATE marking_copy_results SET score_half_units=4`); err != nil {
		t.Fatal(err)
	}
	id, _ := createDeck(t, f)
	cards, err := f.q.ListTrainingCards(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	for _, card := range cards {
		if card.Selected != 0 {
			t.Fatal("P3 has no weak family but card was selected")
		}
	}
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), url.Values{"action": {"publish"}}); r.Code != 422 {
		t.Fatalf("empty deck publish=%d", r.Code)
	}
	form := url.Values{"action": {"save"}, "title": {"Choix manuel"}}
	for i, card := range cards {
		form.Set(fmt.Sprintf("order_%d", card.ID), strconv.Itoa(i+1))
	}
	form.Set(fmt.Sprintf("card_%d", cards[0].ID), "on")
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), form); r.Code != 303 {
		t.Fatalf("manual selection=%d", r.Code)
	}
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), url.Values{"action": {"publish"}}); r.Code != 303 {
		t.Fatalf("manual publish=%d: %s", r.Code, r.Body.String())
	}
}
func TestInvalidMathCannotPublishAndTeacherIsolation(t *testing.T) {
	f := fixture(t)
	snapshot := ""
	if err := f.conn.QueryRow(`SELECT content FROM student_exam_content WHERE student_exam_id=100`).Scan(&snapshot); err != nil {
		t.Fatal(err)
	}
	var qcm config.QCM
	if err := json.Unmarshal([]byte(snapshot), &qcm); err != nil {
		t.Fatal(err)
	}
	qcm.Questions[0].Content = `$#read("private")$`
	raw, _ := json.Marshal(qcm)
	if _, err := f.conn.Exec(`UPDATE student_exam_content SET content=? WHERE student_exam_id=100`, string(raw)); err != nil {
		t.Fatal(err)
	}
	id, token := createDeck(t, f)
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), url.Values{"action": {"publish"}}); r.Code != 422 {
		t.Fatalf("invalid Typst published: %d", r.Code)
	}
	if r := f.req("GET", "/train/"+token, nil, false); r.Code != 404 {
		t.Fatalf("invalid deck public status=%d", r.Code)
	}
	otherRequest := httptest.NewRequest("GET", "/dashboard/training", nil)
	session, err := login.GetStore().Get(otherRequest, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(2)
	session.Values["username"] = "other"
	response := httptest.NewRecorder()
	if err := session.Save(otherRequest, response); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", fmt.Sprintf("/dashboard/training/%d", id), nil)
	request.AddCookie(response.Result().Cookies()[0])
	request.AddCookie(f.csrf)
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, request)
	if rec.Code != 404 {
		t.Fatalf("foreign deck status=%d", rec.Code)
	}
	noCSRF := httptest.NewRequest("POST", fmt.Sprintf("/dashboard/training/%d", id), strings.NewReader("action=publish"))
	noCSRF.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	noCSRF.AddCookie(f.session)
	noCSRF.AddCookie(f.csrf)
	rec = httptest.NewRecorder()
	f.handler.ServeHTTP(rec, noCSRF)
	if rec.Code != 403 {
		t.Fatalf("missing CSRF status=%d", rec.Code)
	}
}

func TestTeacherCanAddRelevantLibraryQuestionAndVariant(t *testing.T) {
	f := fixture(t)
	for _, statement := range []string{
		`INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,content,user_id) VALUES(3,1,1,1,1,1,1,'Convertir des unités',1)`,
		`INSERT INTO answers(question_id,state,content,user_id) VALUES(3,1,'1000 mL',1),(3,0,'10 mL',1)`,
		`INSERT INTO alt_questions(id,question_id,content,user_id) VALUES(4,3,'Variante : convertir 2 L',1)`,
		`INSERT INTO alt_answers(alt_question_id,state,content,user_id) VALUES(4,1,'2000 mL',1),(4,0,'20 mL',1)`,
	} {
		if _, err := f.conn.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	id, token := createDeck(t, f)
	cards, err := f.q.ListTrainingCards(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 5 {
		t.Fatalf("expected 3 observed + 2 library candidates, got %d", len(cards))
	}
	form := url.Values{"action": {"save"}, "title": {"Conversions et masse volumique"}}
	libraryCount := 0
	for i, card := range cards {
		form.Set(fmt.Sprintf("order_%d", card.ID), strconv.Itoa(i+1))
		if i < 3 {
			form.Set(fmt.Sprintf("card_%d", card.ID), "on")
		} else {
			if card.Selected != 0 {
				t.Fatal("library card was preselected by P3")
			}
			form.Set(fmt.Sprintf("card_%d", card.ID), "on")
			libraryCount++
		}
	}
	if libraryCount != 2 {
		t.Fatal("missing library variants")
	}
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), form); r.Code != 303 {
		t.Fatalf("save %d", r.Code)
	}
	if r := f.post(fmt.Sprintf("/dashboard/training/%d", id), url.Values{"action": {"publish"}}); r.Code != 303 {
		t.Fatalf("publish %d: %s", r.Code, r.Body.String())
	}
	if _, err := f.conn.Exec(`UPDATE questions SET content='Changement ultérieur' WHERE id=3; UPDATE alt_questions SET content='Autre changement' WHERE id=4`); err != nil {
		t.Fatal(err)
	}
	page := f.req("GET", "/train/"+token, nil, false)
	if page.Code != 200 {
		t.Fatalf("public=%d", page.Code)
	}
	body := page.Body.String()
	for _, want := range []string{"Convertir des unités", "Variante : convertir 2 L"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing added library card %q", want)
		}
	}
	for _, forbidden := range []string{"Changement ultérieur", "Autre changement"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("published snapshot changed: %q", forbidden)
		}
	}
}
