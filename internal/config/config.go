package config

const (
	ImageSavePath      = "assets/images"
	PublicImageBaseURL = "/static/images/"
	RefQCMTypst        = "internal/config/ref_qcm.txt"
	ImagePathTypst     = "../../images/" // typst gère par rapport au dossier où il compile le fichier.
)

type QCMType string

const (
	PreviewQuestion QCMType = "_question_preview.typ"
)

type QuestionType string

const (
	MainQuestion QuestionType = "mainQuestion"
	AltQuestion  QuestionType = "altQuestion"
)

type Answer struct {
	Symbol  string // Exemple : "▷" ou Unicode
	Content string
}

type Image struct {
	Name  string
	Width string
}
type Question struct {
	Content string
	Image   Image
	Answers []Answer
}

type QCM struct {
	Student   string
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
