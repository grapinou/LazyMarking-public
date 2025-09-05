package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func ImageWf(baseURL string) {
	image := worktool.ImageStructWf{
		QuestionID:    "2",
		AltQuestionID: "",
		ImagePath:     "assets/banque_images/FullMoon.jpg",
		Width:         "50",
	}

	worktool.PostImageTesterWF(baseURL,
		data.DefaultImageRoutes.AddURL, image)

	image = worktool.ImageStructWf{
		QuestionID:    "3",
		AltQuestionID: "",
		ImagePath:     "assets/banque_images/carre.png",
		Width:         "50",
	}

	worktool.PostImageTesterWF(baseURL,
		data.DefaultImageRoutes.AddURL, image)

	image = worktool.ImageStructWf{
		QuestionID:    "2",
		AltQuestionID: "4",
		ImagePath:     "assets/banque_images/Tour_Eiffel.jpg",
		Width:         "50",
	}

	worktool.PostImageTesterWF(baseURL,
		data.DefaultAltImageRoutes.AddURL, image)

	image = worktool.ImageStructWf{
		QuestionID:    "2",
		AltQuestionID: "5",
		ImagePath:     "assets/banque_images/The_Blue_Marble.jpg",
		Width:         "55",
	}

	worktool.PostImageTesterWF(baseURL,
		data.DefaultAltImageRoutes.AddURL, image)

	image = worktool.ImageStructWf{
		QuestionID:    "2",
		AltQuestionID: "6",
		ImagePath:     "assets/banque_images/Burj_Khalifa.jpg",
		Width:         "66",
	}

	worktool.PostImageTesterWF(baseURL,
		data.DefaultAltImageRoutes.AddURL, image)

	image = worktool.ImageStructWf{
		QuestionID:    "3",
		AltQuestionID: "7",
		ImagePath:     "assets/banque_images/cercle.png",
		Width:         "77",
	}

	worktool.PostImageTesterWF(baseURL,
		data.DefaultAltImageRoutes.AddURL, image)

	image = worktool.ImageStructWf{
		QuestionID:    "3",
		AltQuestionID: "8",
		ImagePath:     "assets/banque_images/Triangle.png",
		Width:         "88",
	}

	worktool.PostImageTesterWF(baseURL,
		data.DefaultAltImageRoutes.AddURL, image)

	image = worktool.ImageStructWf{
		QuestionID:    "3",
		AltQuestionID: "9",
		ImagePath:     "assets/banque_images/Rectangle.png",
		Width:         "99",
	}

	worktool.PostImageTesterWF(baseURL,
		data.DefaultAltImageRoutes.AddURL, image)
}
