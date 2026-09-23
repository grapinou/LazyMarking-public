package training

import (
	"context"
	"crypto/rand"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/marking"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/skip2/go-qrcode"
)

// Only collective exercise content is copied. Student snapshot metadata stays outside P5.
type cardContent struct {
	Instruction string       `json:"instruction,omitempty"`
	Question    string       `json:"question"`
	Answers     []cardAnswer `json:"answers"`
}
type cardAnswer struct {
	Text    string `json:"text"`
	Correct bool   `json:"correct"`
}
type renderedCard struct {
	Instruction string   `json:"instruction,omitempty"`
	Question    string   `json:"question"`
	Answers     []string `json:"answers"`
}
type cardView struct {
	ID       int64
	Position int64
	Selected bool
	Label    string
	Variant  string
}
type teacherPage struct {
	Routes      data.DashboardRoutes
	PageTitle   string
	Decks       []db.ListTrainingDecksRow
	Deck        db.TrainingDeck
	Cards       []cardView
	IsDetail    bool
	IsDraft     bool
	IsPublished bool
	PublicURL   string
	QR          template.URL
	Error       string
	Notice      string
}

func RegisterRoutes(mux *http.ServeMux, q *db.Queries, conn *sql.DB) {
	mux.Handle("GET /dashboard/training", login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { list(w, r, q) })))
	mux.Handle("POST /dashboard/training", login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { create(w, r, q, conn) })))
	mux.Handle("GET /dashboard/training/{id}", login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { detail(w, r, q) })))
	mux.Handle("POST /dashboard/training/{id}", login.CheckAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mutate(w, r, q, conn) })))
	mux.HandleFunc("GET /train/{token}", func(w http.ResponseWriter, r *http.Request) { public(w, r, q) })
	mux.HandleFunc("POST /train/{token}/check", func(w http.ResponseWriter, r *http.Request) { check(w, r, q) })
}

func teacherID(w http.ResponseWriter, r *http.Request, method string) (int64, bool) {
	id, _, ok := tools.CheckRequest(w, r, method)
	return id, ok
}
func deckID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, id > 0 && err == nil
}
func renderTeacher(w http.ResponseWriter, p teacherPage, name string) {
	p.Routes = data.DefaultDashboardRoutes
	tools.RenderMergeTemplate(w, p, data.DefaultDashboarPath, data.DefaultDashboardName, "internal/templates/training/", name)
}
func list(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	user, ok := teacherID(w, r, http.MethodGet)
	if !ok {
		return
	}
	decks, err := q.ListTrainingDecks(r.Context(), user)
	if err != nil {
		http.Error(w, "Entraînements indisponibles.", 500)
		return
	}
	renderTeacher(w, teacherPage{PageTitle: "Entraînements", Decks: decks}, "list.html")
}
func publicURL(token string) (string, error) {
	base := strings.TrimSpace(os.Getenv("APP_BASE_URL"))
	u, err := url.Parse(base)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme == "http" && u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") {
		return "", errors.New("Configurez APP_BASE_URL avec une URL HTTPS publique avant publication")
	}
	return strings.TrimRight(u.String(), "/") + "/train/" + token, nil
}
func token() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
func validContent(c cardContent) error {
	if strings.TrimSpace(c.Question) == "" || len(c.Answers) < 2 || len(c.Answers) > 12 {
		return errors.New("question ou choix incomplets")
	}
	correct := 0
	for _, a := range c.Answers {
		if strings.TrimSpace(a.Text) == "" {
			return errors.New("une proposition est vide")
		}
		if a.Correct {
			correct++
		}
	}
	if correct == 0 {
		return errors.New("aucune bonne réponse définie")
	}
	return nil
}
func fromQuestion(q config.Question) (cardContent, error) {
	if q.Image.Name != "" {
		return cardContent{}, errors.New("image non disponible dans l’entraînement mobile")
	}
	c := cardContent{Instruction: q.Instruction, Question: q.Content}
	for _, a := range q.Answers {
		c.Answers = append(c.Answers, cardAnswer{Text: a.Content, Correct: a.State == 1})
	}
	return c, validContent(c)
}
func create(w http.ResponseWriter, r *http.Request, q *db.Queries, conn *sql.DB) {
	user, ok := teacherID(w, r, http.MethodPost)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Requête invalide.", 400)
		return
	}
	generation, err := strconv.ParseInt(r.FormValue("exam_generated_id"), 10, 64)
	if err != nil || generation <= 0 {
		http.Error(w, "Évaluation invalide.", 400)
		return
	}
	source, err := q.GetMarkingGeneration(r.Context(), db.GetMarkingGenerationParams{UserID: user, GenerationID: generation})
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rows, err := q.ListCurrentExamResultsForGeneration(r.Context(), db.ListCurrentExamResultsForGenerationParams{UserID: user, GenerationID: generation})
	if err != nil {
		http.Error(w, "Analyse indisponible.", 500)
		return
	}
	candidates := marking.TrainingCandidates(rows)
	library, err := libraryCandidates(r.Context(), q, conn, user, candidates)
	if err != nil {
		http.Error(w, "Bibliothèque indisponible.", 500)
		return
	}
	candidates = append(candidates, library...)
	if len(candidates) == 0 {
		http.Error(w, "Aucune question corrigée utilisable pour créer un entraînement.", 422)
		return
	}
	tx, err := conn.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, "Création indisponible.", 500)
		return
	}
	defer tx.Rollback()
	tq := q.WithTx(tx)
	secret, err := token()
	if err != nil {
		http.Error(w, "Création indisponible.", 500)
		return
	}
	title := "Entraînement — " + source.ExamName
	if len([]rune(title)) > 160 {
		title = string([]rune(title)[:160])
	}
	id, err := tq.InsertTrainingDeck(r.Context(), db.InsertTrainingDeckParams{UserID: user, GenerationID: sql.NullInt64{Int64: generation, Valid: true}, Title: title, PublicToken: secret})
	if err != nil {
		http.Error(w, "Création indisponible.", 500)
		return
	}
	position := int64(0)
	selected := 0
	for _, candidate := range candidates {
		c, err := fromQuestion(candidate.Question)
		if err != nil {
			continue
		}
		raw, _ := json.Marshal(c)
		flag := int64(0)
		if candidate.Weak {
			flag = 1
			selected++
		}
		err = tq.InsertTrainingCard(r.Context(), db.InsertTrainingCardParams{DeckID: id, Position: position, Selected: flag, SourceQuestionID: sql.NullInt64{Int64: candidate.Question.Tags.MainQuestionID, Valid: candidate.Question.Tags.MainQuestionID > 0}, SourceVariantType: sql.NullString{String: string(candidate.Question.Tags.VariantType), Valid: candidate.Question.Tags.VariantType != ""}, SourceVariantID: sql.NullInt64{Int64: candidate.Question.Tags.VariantID, Valid: candidate.Question.Tags.VariantID > 0}, ContentJson: string(raw)})
		if err != nil {
			http.Error(w, "Création indisponible.", 500)
			return
		}
		position++
	}
	if position == 0 {
		http.Error(w, "Aucune question à choix textuelle utilisable.", 422)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "Création indisponible.", 500)
		return
	}
	target := fmt.Sprintf("/dashboard/training/%d", id)
	if selected == 0 {
		target += "?notice=none"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
func detail(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	user, ok := teacherID(w, r, http.MethodGet)
	if !ok {
		return
	}
	id, ok := deckID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	d, err := q.GetTrainingDeck(r.Context(), db.GetTrainingDeckParams{ID: id, UserID: user})
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cards, err := q.ListTrainingCards(r.Context(), id)
	if err != nil {
		http.Error(w, "Cartes indisponibles.", 500)
		return
	}
	p := teacherPage{PageTitle: d.Title, Deck: d, IsDetail: true, IsDraft: d.State == "draft", IsPublished: d.State == "published"}
	if r.URL.Query().Get("notice") == "none" {
		p.Notice = "P3 n’a détecté aucune question à retravailler. Sélectionnez manuellement les cartes utiles."
	}
	if r.URL.Query().Get("error") == "math" {
		p.Error = "Une formule ou une carte ne peut pas être publiée. Vérifiez les cartes sélectionnées."
	}
	for _, card := range cards {
		var content cardContent
		if json.Unmarshal([]byte(card.ContentJson), &content) != nil {
			continue
		}
		label := content.Question
		if len([]rune(label)) > 130 {
			label = string([]rune(label)[:130]) + "…"
		}
		v := "Question d’origine"
		if card.SourceVariantType.String == string(config.AltQuestion) {
			v = "Variante"
		}
		p.Cards = append(p.Cards, cardView{ID: card.ID, Position: card.Position + 1, Selected: card.Selected == 1, Label: label, Variant: v})
	}
	if d.State != "draft" {
		p.PublicURL, err = publicURL(d.PublicToken)
		if err == nil && d.State == "published" {
			png, e := qrcode.Encode(p.PublicURL, qrcode.Medium, 300)
			if e == nil {
				p.QR = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png))
			}
		}
	}
	renderTeacher(w, p, "detail.html")
}
func renderCard(ctx context.Context, c cardContent) (renderedCard, error) {
	if err := validContent(c); err != nil {
		return renderedCard{}, err
	}
	instruction, err := tools.RenderTrainingText(ctx, c.Instruction)
	if err != nil {
		return renderedCard{}, fmt.Errorf("consigne : %w", err)
	}
	question, err := tools.RenderTrainingText(ctx, c.Question)
	if err != nil {
		return renderedCard{}, fmt.Errorf("énoncé : %w", err)
	}
	result := renderedCard{Instruction: string(instruction), Question: string(question)}
	for i, a := range c.Answers {
		h, e := tools.RenderTrainingText(ctx, a.Text)
		if e != nil {
			return renderedCard{}, fmt.Errorf("réponse %d : %w", i+1, e)
		}
		result.Answers = append(result.Answers, string(h))
	}
	return result, nil
}
func mutate(w http.ResponseWriter, r *http.Request, q *db.Queries, conn *sql.DB) {
	user, ok := teacherID(w, r, http.MethodPost)
	if !ok {
		return
	}
	id, ok := deckID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Requête invalide.", 400)
		return
	}
	d, err := q.GetTrainingDeck(r.Context(), db.GetTrainingDeckParams{ID: id, UserID: user})
	if err != nil {
		http.NotFound(w, r)
		return
	}
	action := r.FormValue("action")
	if action == "close" || action == "reopen" {
		from, to := "published", "closed"
		if action == "reopen" {
			from, to = "closed", "published"
		}
		if d.State != from {
			http.Error(w, "Transition impossible.", 409)
			return
		}
		affected, err := q.SetTrainingState(r.Context(), db.SetTrainingStateParams{State: to, ID: id, UserID: user, State_2: from})
		if err != nil || affected != 1 {
			http.Error(w, "Transition impossible.", 409)
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/dashboard/training/%d", id), 303)
		return
	}
	if d.State != "draft" {
		http.Error(w, "Seul un brouillon peut être modifié.", 409)
		return
	}
	cards, err := q.ListTrainingCards(r.Context(), id)
	if err != nil {
		http.Error(w, "Cartes indisponibles.", 500)
		return
	}
	if action == "save" {
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" || len([]rune(title)) > 160 {
			http.Error(w, "Titre invalide.", 422)
			return
		}
		type positioned struct {
			card     db.TrainingCard
			order    int64
			selected int64
		}
		ordered := make([]positioned, 0, len(cards))
		for _, card := range cards {
			v, e := strconv.ParseInt(r.FormValue(fmt.Sprintf("order_%d", card.ID)), 10, 64)
			if e != nil || v < 1 || v > 10000 {
				http.Error(w, "Ordre invalide.", 422)
				return
			}
			s := int64(0)
			if r.FormValue(fmt.Sprintf("card_%d", card.ID)) == "on" {
				s = 1
			}
			ordered = append(ordered, positioned{card, v, s})
		}
		sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].order < ordered[j].order })
		tx, e := conn.BeginTx(r.Context(), nil)
		if e != nil {
			http.Error(w, "Enregistrement indisponible.", 500)
			return
		}
		defer tx.Rollback()
		tq := q.WithTx(tx)
		if n, e := tq.UpdateTrainingTitle(r.Context(), db.UpdateTrainingTitleParams{Title: title, ID: id, UserID: user}); e != nil || n != 1 {
			http.Error(w, "Enregistrement indisponible.", 500)
			return
		}
		for _, item := range ordered {
			if _, e := tq.UpdateTrainingCard(r.Context(), db.UpdateTrainingCardParams{Selected: item.selected, Position: 1000000000 + item.card.ID, ID: item.card.ID, DeckID: id, UserID: user}); e != nil {
				http.Error(w, "Enregistrement indisponible.", 500)
				return
			}
		}
		for i, item := range ordered {
			if _, e := tq.UpdateTrainingCard(r.Context(), db.UpdateTrainingCardParams{Selected: item.selected, Position: int64(i), ID: item.card.ID, DeckID: id, UserID: user}); e != nil {
				http.Error(w, "Enregistrement indisponible.", 500)
				return
			}
		}
		if e := tx.Commit(); e != nil {
			http.Error(w, "Enregistrement indisponible.", 500)
			return
		}
	} else if action == "publish" {
		if _, err := publicURL(d.PublicToken); err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		prepared := map[int64]string{}
		count := 0
		for _, card := range cards {
			if card.Selected != 1 {
				continue
			}
			count++
			var content cardContent
			if json.Unmarshal([]byte(card.ContentJson), &content) != nil {
				http.Error(w, "Carte invalide.", 422)
				return
			}
			rendered, e := renderCard(r.Context(), content)
			if e != nil {
				http.Error(w, "Carte "+strconv.Itoa(count)+" : "+e.Error(), 422)
				return
			}
			raw, _ := json.Marshal(rendered)
			prepared[card.ID] = string(raw)
		}
		if count == 0 {
			http.Error(w, "Sélectionnez au moins une carte avant publication.", 422)
			return
		}
		tx, e := conn.BeginTx(r.Context(), nil)
		if e != nil {
			http.Error(w, "Publication indisponible.", 500)
			return
		}
		defer tx.Rollback()
		tq := q.WithTx(tx)
		for _, card := range cards {
			if raw, ok := prepared[card.ID]; ok {
				n, e := tq.SetTrainingRenderedCard(r.Context(), db.SetTrainingRenderedCardParams{RenderedJson: sql.NullString{String: raw, Valid: true}, ID: card.ID, DeckID: id, UserID: user})
				if e != nil || n != 1 {
					http.Error(w, "Publication indisponible.", 500)
					return
				}
			}
		}
		n, e := tq.SetTrainingState(r.Context(), db.SetTrainingStateParams{State: "published", ID: id, UserID: user, State_2: "draft"})
		if e != nil || n != 1 {
			http.Error(w, "Publication indisponible.", 500)
			return
		}
		if e := tx.Commit(); e != nil {
			http.Error(w, "Publication indisponible.", 500)
			return
		}
	} else {
		http.Error(w, "Action invalide.", 400)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/dashboard/training/%d", id), 303)
}

//go:embed public.html
var publicTemplateSource string
var publicTemplate = template.Must(template.New("public").Parse(publicTemplateSource))

type publicCard struct {
	ID          int64    `json:"id"`
	Instruction string   `json:"instruction"`
	Question    string   `json:"question"`
	Answers     []string `json:"answers"`
	Multiple    bool     `json:"multiple"`
}
type publicPage struct {
	Title     string
	Count     int
	CardsJSON template.JS
	CSRF      string
}

func published(w http.ResponseWriter, r *http.Request, q *db.Queries) (db.GetPublishedTrainingDeckRow, []db.TrainingCard, bool) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	token := r.PathValue("token")
	if len(token) != 43 {
		http.NotFound(w, r)
		return db.GetPublishedTrainingDeckRow{}, nil, false
	}
	d, err := q.GetPublishedTrainingDeck(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return db.GetPublishedTrainingDeckRow{}, nil, false
	}
	cards, err := q.ListTrainingCards(r.Context(), d.ID)
	if err != nil {
		http.Error(w, "Entraînement indisponible.", 500)
		return db.GetPublishedTrainingDeckRow{}, nil, false
	}
	return d, cards, true
}
func public(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	d, cards, ok := published(w, r, q)
	if !ok {
		return
	}
	visible := make([]publicCard, 0)
	for _, card := range cards {
		if card.Selected != 1 || !card.RenderedJson.Valid {
			continue
		}
		var rendered renderedCard
		if json.Unmarshal([]byte(card.RenderedJson.String), &rendered) != nil {
			http.Error(w, "Entraînement indisponible.", 500)
			return
		}
		var content cardContent
		if json.Unmarshal([]byte(card.ContentJson), &content) != nil {
			http.Error(w, "Entraînement indisponible.", 500)
			return
		}
		multiple := false
		count := 0
		for _, a := range content.Answers {
			if a.Correct {
				count++
			}
		}
		multiple = count > 1
		visible = append(visible, publicCard{ID: card.ID, Instruction: rendered.Instruction, Question: rendered.Question, Answers: rendered.Answers, Multiple: multiple})
	}
	if len(visible) == 0 {
		http.NotFound(w, r)
		return
	}
	raw, _ := json.Marshal(visible)
	page := publicPage{Title: d.Title, Count: len(visible), CardsJSON: template.JS(raw), CSRF: csrf.Token(r)}
	if err := publicTemplate.Execute(w, page); err != nil {
		log.Printf("render public training: %v", err)
	}
}

type checkRequest struct {
	CardID   int64 `json:"card_id"`
	Selected []int `json:"selected"`
}
type checkResult struct {
	Correct bool  `json:"correct"`
	Answers []int `json:"answers"`
}

func evaluate(c cardContent, selected []int) (checkResult, error) {
	if err := validContent(c); err != nil {
		return checkResult{}, err
	}
	picked := map[int]bool{}
	for _, v := range selected {
		if v < 0 || v >= len(c.Answers) || picked[v] {
			return checkResult{}, errors.New("sélection invalide")
		}
		picked[v] = true
	}
	result := checkResult{Correct: true, Answers: []int{}}
	for i, a := range c.Answers {
		if a.Correct {
			result.Answers = append(result.Answers, i)
		}
		if picked[i] != a.Correct {
			result.Correct = false
		}
	}
	return result, nil
}
func check(w http.ResponseWriter, r *http.Request, q *db.Queries) {
	d, cards, ok := published(w, r, q)
	if !ok {
		return
	}
	_ = d
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	var request checkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Réponse invalide.", 400)
		return
	}
	for _, card := range cards {
		if card.ID != request.CardID || card.Selected != 1 || !card.RenderedJson.Valid {
			continue
		}
		var content cardContent
		if json.Unmarshal([]byte(card.ContentJson), &content) != nil {
			break
		}
		result, err := evaluate(content, request.Selected)
		if err != nil {
			http.Error(w, "Réponse invalide.", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(result)
		return
	}
	http.NotFound(w, r)
}
