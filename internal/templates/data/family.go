package data

import (
	"net/url"
	"strconv"
)

type QuestionContext struct {
	ID      int64
	Content string
}

type VariantContext struct {
	ID      int64
	Content string
}

func QuestionURL(base string, questionID int64) string {
	return base + "?question_id=" + url.QueryEscape(strconv.FormatInt(questionID, 10))
}

func VariantURL(base string, questionID, variantID int64) string {
	return QuestionURL(base, questionID) + "&alt_question_id=" + url.QueryEscape(strconv.FormatInt(variantID, 10))
}
