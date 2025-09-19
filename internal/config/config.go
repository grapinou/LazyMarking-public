package config

// Limite globale de connexions DB
var DBSemaphore = make(chan struct{}, 20)

const (
	ImageSavePath        = "assets/images"
	PublicImageBaseURL   = "/static/images/"
	RefQCMTypst          = "internal/config/ref_qcm.txt"
	RefQCMLandscapeTypst = "internal/config/ref_qcm_landscape.txt"
	ImagePathTypst       = "../../images/" // typst gère par rapport au dossier où il compile le fichier.
)

type QCMType string

const (
	PreviewQuestion     QCMType = "_question_preview.typ"
	PreviewQCM          QCMType = "_qcm_preview.typ"
	PreviewLandscapeQCM QCMType = "_qcm_landscape.typ"
	ExamQCM             QCMType = "_.typ"
	MiniQCM             QCMType = "_miniqcm_landscape.typ"
)

type QuestionType string

const (
	MainQuestion QuestionType = "mainQuestion"
	AltQuestion  QuestionType = "altQuestion"
)

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Answer struct {
	Symbol  string          `json:"symbol"` // Exemple : "▷" ou Unicode
	Content string          `json:"content"`
	State   int64           `json:"state"`
	Circle  CircleValidated `json:"circle"`
}

type Image struct {
	Name  string `json:"name"`
	Width string `json:"width"`
}

type Subject struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Theme struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type YearLevel struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Skill struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Difficulty struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Point struct {
	ID         int64 `json:"id"`
	PointValue int64 `json:"name"`
}

type Tags struct {
	MainQuestionID int64      `json:"main_question_id"`
	Subject        Subject    `json:"subject"`
	Theme          Theme      `json:"theme"`
	YearLevel      YearLevel  `json:"year_level"`
	Skill          Skill      `json:"skill"`
	Difficulty     Difficulty `json:"difficulty"`
	Point          Point      `json:"point"`
}

type Question struct {
	Tags    Tags            `json:"tags"`
	Content string          `json:"content"`
	Image   Image           `json:"image"`
	Circle  CircleValidated `json:"circle_validated"`
	Answers []Answer        `json:"answers"`
}

type QCM struct {
	Name      string     `json:"name"`
	Student   StudentQCM `json:"student_qcm"`
	Questions []Question `json:"questions"`
}

type ClassCode struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Student struct {
	ID         int64
	FirstName  string
	LastName   string
	ClassCodes []ClassCode
}

type StudentQCM struct {
	ID         int64     `json:"id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	ClassCodes ClassCode `json:"class_codes"`
}

type QrCodeInfo struct {
	StudentExamID int64  `json:"student_exam_id"`
	PageExam      int    `json:"page_exam"`
	PageName      string `json:"page_name"`
}

type CircleValidated struct {
	Position Position `json:"position"`
	Radius   int      `json:"radius"`
}

type PageContent struct {
	Questions []CircleValidated `json:"questions"`
	Answers   []CircleValidated `json:"answers"`
}

type Page struct {
	Number int
	Name   string
}
type Exam struct {
	StudentExamID int64
	Pages         []Page
}

type QuestionState int

const (
	Incorrect QuestionState = iota // 0
	Partial                        // 1
	Correct                        // 2
)

type QuestionMark struct {
	Score float64
	Total int64
	State QuestionState
}

type HomoPage struct {
	Name    string
	Content PageContent
}

type CounterTag struct {
	Name  string
	Score float64
	Total int64
}
