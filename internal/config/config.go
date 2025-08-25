package config

const (
	ImageSavePath        = "assets/images"
	PublicImageBaseURL   = "/static/images/"
	RefQCMTypst          = "internal/config/ref_qcm.txt"
	RefQCMLandscapeTypst = "internal/config/ref_qcm_landscape.txt"
	ImagePathTypst       = "../../images/" // typst gère par rapport au dossier où il compile le fichier.
)

type QCMType string

const (
	PreviewQuestion QCMType = "_question_preview.typ"
	PreviewQCM      QCMType = "_qcm_preview.typ"
	LandscapeQCM    QCMType = "_qcm_landscape.typ"
	ExamQCM         QCMType = "_.typ"
)

type QuestionType string

const (
	MainQuestion QuestionType = "mainQuestion"
	AltQuestion  QuestionType = "altQuestion"
)

type Position struct {
	X float64
	Y float64
}

type Answer struct {
	Symbol   string // Exemple : "▷" ou Unicode
	Content  string
	State    int64
	Position Position
}

type Image struct {
	Name  string
	Width string
}

type Subject struct {
	ID   int64
	Name string
}

type Theme struct {
	ID   int64
	Name string
}

type YearLevel struct {
	ID   int64
	Name string
}

type Skill struct {
	ID   int64
	Name string
}

type Difficulty struct {
	ID   int64
	Name string
}

type Point struct {
	ID         int64
	PointValue int64
}

type Tags struct {
	MainQuestionID int64
	Subject        Subject
	Theme          Theme
	YearLevel      YearLevel
	Skill          Skill
	Difficulty     Difficulty
	Point          Point
}

type Question struct {
	Tags     Tags
	Content  string
	Image    Image
	Position Position
	Answers  []Answer
}

type QCM struct {
	Name      string
	Student   StudentQCM
	Questions []Question
}

type ClassCode struct {
	ID   int64
	Name string
}
type Student struct {
	ID         int64
	FirstName  string
	LastName   string
	ClassCodes []ClassCode
}
type StudentQCM struct {
	ID         int64
	FirstName  string
	LastName   string
	ClassCodes ClassCode
}
