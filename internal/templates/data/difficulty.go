package data

import (
	"net/url"
	"strconv"
)

type DifficultyRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type DifficultyContext struct {
	ID   int64
	Name string
}

type DifficultyListItem struct {
	ID        int64
	Name      string
	EditURL   string
	DeleteURL string
}

func DifficultyURL(base string, difficultyID int64) string {
	return base + "?difficulty_id=" + url.QueryEscape(strconv.FormatInt(difficultyID, 10))
}

var DefaultDifficultyRoutes = DifficultyRoutes{
	AddURL:    "/dashboard/questions/difficulties/add",
	EditURL:   "/dashboard/questions/difficulties/edit",
	DeleteURL: "/dashboard/questions/difficulties/delete",
}

type DifficultyPageData struct {
	Routes            DashboardRoutes
	DifficultyRoutes  DifficultyRoutes
	DifficultyContext DifficultyContext
	DifficultyItems   []DifficultyListItem
	CancelURL         string
	PageTitle         string
}

type DifficultyTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultDifficultyTemplateName = DifficultyTemplateName{
	AddForm:    "add_form_difficulty.html",
	EditForm:   "edit_form_difficulty.html",
	DeleteForm: "delete_form_difficulty.html",
	Table:      "table_difficulties.html",
}

var DefaultDifficultyPathTemplate = "internal/templates/difficulties/"
