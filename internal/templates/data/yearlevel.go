package data

import (
	"net/url"
	"strconv"
)

type YearLevelRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type YearLevelContext struct {
	ID   int64
	Name string
}

type YearLevelListItem struct {
	ID        int64
	Name      string
	EditURL   string
	DeleteURL string
}

func YearLevelURL(base string, yearLevelID int64) string {
	return base + "?yearlevel_id=" + url.QueryEscape(strconv.FormatInt(yearLevelID, 10))
}

var DefaultYearLevelRoutes = YearLevelRoutes{
	AddURL:    "/dashboard/questions/yearlevels/add",
	EditURL:   "/dashboard/questions/yearlevels/edit",
	DeleteURL: "/dashboard/questions/yearlevels/delete",
}

type YearLevelPageData struct {
	Routes           DashboardRoutes
	YearLevelRoutes  YearLevelRoutes
	YearLevelContext YearLevelContext
	YearLevelItems   []YearLevelListItem
	CancelURL        string
	PageTitle        string
}

type YearLevelTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultYearLevelTemplateName = YearLevelTemplateName{
	AddForm:    "add_form_yearlevel.html",
	EditForm:   "edit_form_yearlevel.html",
	DeleteForm: "delete_form_yearlevel.html",
	Table:      "table_yearlevels.html",
}

var DefaultYearLevelPathTemplate = "internal/templates/yearlevels/"
