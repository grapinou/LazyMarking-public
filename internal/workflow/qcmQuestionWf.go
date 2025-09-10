package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func QcmQuestionWf(baseURL string) {
	/*
		worktool.QcmQuestionsFiller(
			"QCM Question",
			baseURL,
			data.DefaultQCMRoutes.AddQuestionURL,
			data.DefaultQCMQuestionRoutes.AddURL,
			"1",
			"question_ids",
			"1",
		)

		worktool.QcmQuestionsFiller(
			"QCM Question",
			baseURL,
			data.DefaultQCMRoutes.AddQuestionURL,
			data.DefaultQCMQuestionRoutes.AddURL,
			"1",
			"question_ids",
			"2",
		)

		worktool.QcmQuestionsFiller(
			"QCM Question",
			baseURL,
			data.DefaultQCMRoutes.AddQuestionURL,
			data.DefaultQCMQuestionRoutes.AddURL,
			"1",
			"question_ids",
			"3",
		)
	*/
	worktool.QcmQuestionsFiller(
		"QCM Question",
		baseURL,
		data.DefaultQCMRoutes.AddQuestionURL,
		data.DefaultQCMQuestionRoutes.AddURL,
		"1",
		"question_ids",
		"4",
	)
}
