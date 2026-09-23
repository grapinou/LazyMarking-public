package marking

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/csrf"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"sort"
)

const studentCopyCookie = "student_copy_session"

func couponKey() []byte { return []byte(os.Getenv("SESSION_KEY")) }

func couponCode(token string) string {
	mac := hmac.New(sha256.New, couponKey())
	mac.Write([]byte("student-copy-code-v1:" + token))
	code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(mac.Sum(nil))[:12]
	return code[:6] + "-" + code[6:]
}

func randomPublicToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func ensureStudentAccess(ctx context.Context, q *db.Queries, userID, generationID int64) error {
	if len(couponKey()) < 32 {
		return errors.New("SESSION_KEY unavailable for coupons")
	}
	if _, err := q.GetMarkingGeneration(ctx, db.GetMarkingGenerationParams{UserID: userID, GenerationID: generationID}); err != nil {
		return err
	}
	ids, err := q.ListStudentExamsWithoutAccess(ctx, userID, generationID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		token, err := randomPublicToken()
		if err != nil {
			return err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(strings.ReplaceAll(couponCode(token), "-", "")), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := q.InsertStudentCopyAccess(ctx, userID, generationID, id, token, string(hash)); err != nil {
			return err
		}
	}
	return nil
}

func publicCopyURL(token string) (string, error) {
	base := strings.TrimSpace(os.Getenv("APP_BASE_URL"))
	u, err := url.Parse(base)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme == "http" && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") {
		return "", errors.New("APP_BASE_URL must be an absolute HTTP URL for coupons")
	}
	return strings.TrimRight(u.String(), "/") + "/copies/" + token, nil
}

func StudentCouponsPDFHandler(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		return
	}
	id, ok := parsePositiveReviewFormInt(r.URL.Query().Get("exam_generated_id"))
	if !ok {
		http.Error(w, "Évaluation invalide.", 400)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if err := ensureStudentAccess(r.Context(), q, userID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
		} else {
			log.Printf("prepare coupons: %v", err)
			http.Error(w, "Coupons indisponibles.", 500)
		}
		return
	}
	rows, err := q.ListStudentCopyAccess(r.Context(), userID, id)
	if err != nil {
		http.Error(w, "Coupons indisponibles.", 500)
		return
	}
	order := collate.New(language.French, collate.IgnoreCase)
	sort.SliceStable(rows, func(i, j int) bool {
		if c := order.CompareString(strings.TrimSpace(rows[i].LastName), strings.TrimSpace(rows[j].LastName)); c != 0 {
			return c < 0
		}
		if c := order.CompareString(strings.TrimSpace(rows[i].FirstName), strings.TrimSpace(rows[j].FirstName)); c != 0 {
			return c < 0
		}
		return rows[i].StudentExamID < rows[j].StudentExamID
	})
	coupons := make([]tools.StudentCoupon, 0, len(rows))
	for _, a := range rows {
		link, err := publicCopyURL(a.Token)
		if err != nil {
			http.Error(w, "Configurez APP_BASE_URL pour imprimer les coupons.", 500)
			return
		}
		code := couponCode(a.Token)
		if bcrypt.CompareHashAndPassword([]byte(a.CodeHash), []byte(strings.ReplaceAll(code, "-", ""))) != nil {
			http.Error(w, "Clé des coupons modifiée : anciens codes conservés, réimpression indisponible.", 409)
			return
		}
		coupons = append(coupons, tools.StudentCoupon{Name: strings.TrimSpace(a.LastName + " " + a.FirstName), Exam: a.ExamName, Class: a.ClassName, URL: link, Code: code})
	}
	content, err := tools.BuildStudentCouponsPDF(r.Context(), username, coupons)
	if err != nil {
		log.Printf("build student coupons: %v", err)
		http.Error(w, "PDF des coupons indisponible.", 500)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "coupons-copies-corrigees.pdf"}))
	_, _ = w.Write(content)
}

func publishFinalStudentCopies(ctx context.Context, q *db.Queries, userID, generationID int64, now time.Time) (int, error) {
	if err := ensureStudentAccess(ctx, q, userID, generationID); err != nil {
		return 0, err
	}
	rows, err := q.ListCurrentExamResultsForGeneration(ctx, db.ListCurrentExamResultsForGenerationParams{UserID: userID, GenerationID: generationID})
	if err != nil {
		return 0, err
	}
	count := 0
	for _, row := range rows {
		if !hasFinalMarkingScore(row) {
			continue
		}
		affected, err := q.PublishStudentCopyAccess(ctx, userID, generationID, row.StudentExamID, now)
		if err != nil {
			return count, err
		}
		count += int(affected)
	}
	return count, nil
}

func StudentAccessActionHandler(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Requête invalide.", 400)
		return
	}
	id, ok := parsePositiveReviewFormInt(r.FormValue("exam_generated_id"))
	if !ok {
		http.Error(w, "Évaluation invalide.", 400)
		return
	}
	if _, err := q.GetMarkingGeneration(r.Context(), db.GetMarkingGenerationParams{UserID: userID, GenerationID: id}); err != nil {
		http.NotFound(w, r)
		return
	}
	action := r.FormValue("action")
	if action == "publish" {
		if _, err := publishFinalStudentCopies(r.Context(), q, userID, id, time.Now().UTC()); err != nil {
			log.Printf("publish student access: %v", err)
			http.Error(w, "Publication impossible.", 500)
			return
		}
	} else if action == "revoke" || action == "reopen" {
		copyID, ok := parsePositiveReviewFormInt(r.FormValue("student_exam_id"))
		if !ok {
			http.Error(w, "Copie invalide.", 400)
			return
		}
		state := "revoked"
		if action == "reopen" {
			state = "published"
			rows, err := q.ListCurrentExamResultsForGeneration(r.Context(), db.ListCurrentExamResultsForGenerationParams{UserID: userID, GenerationID: id})
			if err != nil {
				http.Error(w, "Accès indisponible.", 500)
				return
			}
			final := false
			for _, row := range rows {
				if row.StudentExamID == copyID && hasFinalMarkingScore(row) {
					final = true
					break
				}
			}
			if !final {
				http.Error(w, "Cette copie n'est pas finalisée.", 409)
				return
			}
		}
		changed, err := q.ChangeStudentCopyAccess(r.Context(), userID, id, copyID, state, time.Now().UTC())
		if err != nil {
			http.Error(w, "Modification impossible.", 500)
			return
		}
		if changed == 0 {
			http.NotFound(w, r)
			return
		}
	} else {
		http.Error(w, "Action invalide.", 400)
		return
	}
	http.Redirect(w, r, markingGenerationURL(id), http.StatusSeeOther)
}

func copySessionSignature(token string, expires int64) string {
	mac := hmac.New(sha256.New, couponKey())
	mac.Write([]byte(fmt.Sprintf("student-copy-session-v1:%s:%d", token, expires)))
	return hex.EncodeToString(mac.Sum(nil))
}

func authorizedStudentCopy(r *http.Request, token string, now time.Time) bool {
	if len(couponKey()) < 32 {
		return false
	}
	cookie, err := r.Cookie(studentCopyCookie)
	if err != nil {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return false
	}
	expires, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || now.Unix() >= expires || expires > now.Add(30*time.Minute).Unix() {
		return false
	}
	want := copySessionSignature(token, expires)
	return hmac.Equal([]byte(parts[1]), []byte(want))
}

var publicCopyPage = template.Must(template.New("copy").Parse(`<!doctype html><html lang="fr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Ma copie corrigée</title><style>body{font:1.1rem system-ui,sans-serif;margin:0;background:#f5f7fb;color:#172030}main{max-width:34rem;margin:7vh auto;padding:2rem;background:white;border-radius:1rem;box-shadow:0 1px 12px #ccd2df}h1{font-size:1.6rem}label,input,button{display:block;width:100%;box-sizing:border-box}input{font:inherit;padding:.75rem;margin:.6rem 0 1rem}button,a.button{font:inherit;display:inline-block;background:#174ea6;color:white;border:0;border-radius:.5rem;padding:.8rem 1rem;text-decoration:none;cursor:pointer}.muted{color:#465269}.error{color:#9b1c1c}</style></head><body><main><h1>{{.Title}}</h1>{{if .Name}}<p>{{.Name}}</p>{{end}}{{if .Message}}<p class="{{if .Error}}error{{else}}muted{{end}}">{{.Message}}</p>{{end}}{{if .Ask}}<form method="post"><input type="hidden" name="gorilla.csrf.Token" value="{{.CSRF}}"><label for="code">Code personnel</label><input id="code" name="code" autocomplete="one-time-code" inputmode="text" maxlength="20" required><button>Voir ma correction</button></form>{{end}}{{if .Download}}<p><a class="button" href="{{.Download}}">Télécharger ma copie corrigée</a></p>{{end}}</main></body></html>`))

type studentCopyPage struct {
	Title, Name, Message, CSRF, Download string
	Ask, Error                           bool
}

func renderStudentCopyPage(w http.ResponseWriter, r *http.Request, page studentCopyPage) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Referrer-Policy", "no-referrer")
	page.CSRF = csrf.Token(r)
	_ = publicCopyPage.Execute(w, page)
}

func StudentCopyPageHandler(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if len(couponKey()) < 32 {
		http.Error(w, "Accès temporairement indisponible.", http.StatusServiceUnavailable)
		return
	}
	token := r.PathValue("token")
	if len(token) != 43 {
		http.NotFound(w, r)
		return
	}
	a, err := q.GetStudentCopyAccessByToken(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	now := time.Now().UTC()
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Requête invalide.", 400)
			return
		}
		code := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(r.FormValue("code")), "-", ""))
		allowed, err := q.UseStudentCopyCodeAttempt(r.Context(), a.StudentExamID, now)
		if err != nil {
			http.Error(w, "Accès indisponible.", 500)
			return
		}
		if !allowed || len(code) != 12 || bcrypt.CompareHashAndPassword([]byte(a.CodeHash), []byte(code)) != nil {
			renderStudentCopyPage(w, r, studentCopyPage{Title: "Ma copie corrigée", Message: "Code incorrect ou accès temporairement limité. Réessayez dans 15 minutes si nécessaire.", Ask: true, Error: true})
			return
		}
		if err := q.ClearStudentCopyCodeFailures(r.Context(), a.StudentExamID); err != nil {
			http.Error(w, "Accès indisponible.", 500)
			return
		}
		expires := now.Add(20 * time.Minute).Unix()
		http.SetCookie(w, &http.Cookie{Name: studentCopyCookie, Value: fmt.Sprintf("%d.%s", expires, copySessionSignature(token, expires)), Path: "/copies/", HttpOnly: true, Secure: r.TLS != nil || strings.HasPrefix(os.Getenv("APP_BASE_URL"), "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 1200})
	} else if !authorizedStudentCopy(r, token, now) {
		renderStudentCopyPage(w, r, studentCopyPage{Title: "Ma copie corrigée", Message: "Saisissez le code indiqué sur votre coupon.", Ask: true})
		return
	}
	if a.State == "revoked" {
		renderStudentCopyPage(w, r, studentCopyPage{Title: "Ma copie corrigée", Message: "Cet accès à la copie corrigée est fermé. Contactez votre professeur."})
		return
	}
	if a.State == "published" && (!a.ExpiresAt.Valid || !now.Before(a.ExpiresAt.Time)) {
		renderStudentCopyPage(w, r, studentCopyPage{Title: "Ma copie corrigée", Message: "Cet accès à la copie corrigée a expiré. Contactez votre professeur si vous souhaitez revoir votre correction."})
		return
	}
	if a.State != "published" {
		renderStudentCopyPage(w, r, studentCopyPage{Title: "Ma copie corrigée", Message: "La correction de cette évaluation n'est pas encore disponible."})
		return
	}
	if _, ok := currentFinalCopy(r.Context(), q, a); !ok {
		renderStudentCopyPage(w, r, studentCopyPage{Title: "Ma copie corrigée", Message: "La correction de cette évaluation n'est pas encore disponible."})
		return
	}
	renderStudentCopyPage(w, r, studentCopyPage{Title: a.ExamName, Name: strings.TrimSpace(a.FirstName + " " + a.LastName), Message: "Votre copie corrigée est disponible jusqu'au " + a.ExpiresAt.Time.Format("02/01/2006") + ".", Download: "/copies/" + token + "/pdf"})
}

func currentFinalCopy(ctx context.Context, q *db.Queries, a db.StudentCopyAccess) (db.ListCurrentExamResultsForGenerationRow, bool) {
	rows, err := q.ListCurrentExamResultsForGeneration(ctx, db.ListCurrentExamResultsForGenerationParams{UserID: a.UserID, GenerationID: a.GenerationID})
	if err != nil {
		return db.ListCurrentExamResultsForGenerationRow{}, false
	}
	for _, row := range rows {
		if row.StudentExamID == a.StudentExamID && hasFinalMarkingScore(row) {
			return row, true
		}
	}
	return db.ListCurrentExamResultsForGenerationRow{}, false
}

func StudentCopyPDFHandler(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Referrer-Policy", "no-referrer")
	token := r.PathValue("token")
	if len(token) != 43 || !authorizedStudentCopy(r, token, time.Now().UTC()) {
		http.NotFound(w, r)
		return
	}
	a, err := q.GetStudentCopyAccessByToken(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if a.State != "published" || !a.ExpiresAt.Valid || !time.Now().UTC().Before(a.ExpiresAt.Time) {
		http.NotFound(w, r)
		return
	}
	row, ok := currentFinalCopy(r.Context(), q, a)
	if !ok {
		http.NotFound(w, r)
		return
	}
	content, err := tools.BuildStudentCorrectedPDF(r.Context(), q, a.UserID, a.Username, row.MarkingJobID, row.CopyResultID, a.StudentExamID)
	if err != nil {
		log.Printf("build individual corrected copy: %v", err)
		http.Error(w, "Copie temporairement indisponible.", 503)
		return
	}
	// A revocation, expiry, or renewed review during rendering must still win.
	latest, err := q.GetStudentCopyAccessByToken(r.Context(), token)
	if err != nil || latest.State != "published" || !latest.ExpiresAt.Valid || !time.Now().UTC().Before(latest.ExpiresAt.Time) {
		http.NotFound(w, r)
		return
	}
	current, ok := currentFinalCopy(r.Context(), q, latest)
	if !ok || current.CopyResultID != row.CopyResultID || current.MarkingJobID != row.MarkingJobID {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "copie-corrigee.pdf"}))
	_, _ = w.Write(content)
}
