package workflow

import (
	"github.com/grapinou/LazyMarking/internal/templates/data"
	"github.com/grapinou/LazyMarking/internal/workflow/worktool"
)

func LoginWf(baseURL string) {

	urlTested := data.DefaultHomeRoutes.LoginURL

	worktool.GetTester(baseURL, urlTested, "Connectez-vous :")

	fields := map[string]string{
		"username": "Sighto",
		"password": "aa",
	}

	worktool.PostTesterWF(baseURL, urlTested, fields)
}
