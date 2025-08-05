package config

const (
	ImageSavePath      = "assets/images"
	PublicImageBaseURL = "/static/images/"
)

type QuestionType string

const (
	MainQuestion QuestionType = "mainQuestion"
	AltQuestion  QuestionType = "altQuestion"
)
