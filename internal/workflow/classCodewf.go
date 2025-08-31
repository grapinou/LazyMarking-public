package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func ClassCodesWf(baseURL string) {
	worktool.ClassCodeFiller(
		"Class Codes",
		baseURL,
		data.DefaultStudentRoutes.ClassCodesURL,
		data.DefaultClassCodeRoutes.AddURL,
		"class_code",
		"test",
	)

	/*
		worktool.ClassCodeFiller(
			"Class Codes",
			baseURL,
			data.DefaultStudentRoutes.ClassCodesURL,
			data.DefaultClassCodeRoutes.AddURL,
			"class_code",
			"6è1",
		)

		worktool.ClassCodeFiller(
			"Class Codes",
			baseURL,
			data.DefaultStudentRoutes.ClassCodesURL,
			data.DefaultClassCodeRoutes.AddURL,
			"class_code",
			"6è2",
		)

		worktool.ClassCodeFiller(
			"Class Codes",
			baseURL,
			data.DefaultStudentRoutes.ClassCodesURL,
			data.DefaultClassCodeRoutes.AddURL,
			"class_code",
			"6è3",
		)
	*/
}
