package data

import (
	"net/url"
	"strconv"
)

type ThemeRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type ThemeContext struct {
	ID   int64
	Name string
}

type ThemeListItem struct {
	ID        int64
	Name      string
	EditURL   string
	DeleteURL string
}

func ThemeURL(base string, themeID int64) string {
	return base + "?theme_id=" + url.QueryEscape(strconv.FormatInt(themeID, 10))
}

var DefaultThemeRoutes = ThemeRoutes{
	AddURL:    "/dashboard/questions/themes/add",
	EditURL:   "/dashboard/questions/themes/edit",
	DeleteURL: "/dashboard/questions/themes/delete",
}

type ThemePageData struct {
	Routes       DashboardRoutes
	ThemeRoutes  ThemeRoutes
	ThemeContext ThemeContext
	ThemeItems   []ThemeListItem
	CancelURL    string
	PageTitle    string
}

type ThemeTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultThemeTemplateName = ThemeTemplateName{
	AddForm:    "add_form_theme.html",
	EditForm:   "edit_form_theme.html",
	DeleteForm: "delete_form_theme.html",
	Table:      "table_themes.html",
}

var DefaultThemePathTemplate = "internal/templates/themes/"
