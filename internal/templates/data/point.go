package data

import (
	"net/url"
	"strconv"
)

type PointRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type PointContext struct {
	ID    int64
	Value int64
}

type PointListItem struct {
	ID        int64
	Value     int64
	EditURL   string
	DeleteURL string
}

type PointFormData struct {
	ID           int64
	CurrentValue int64
}

func PointURL(base string, pointID int64) string {
	return base + "?point_id=" + url.QueryEscape(strconv.FormatInt(pointID, 10))
}

var DefaultPointRoutes = PointRoutes{
	AddURL:    "/dashboard/questions/points/add",
	EditURL:   "/dashboard/questions/points/edit",
	DeleteURL: "/dashboard/questions/points/delete",
}

type PointPageData struct {
	Routes       DashboardRoutes
	PointRoutes  PointRoutes
	PointContext PointContext
	PointItems   []PointListItem
	Form         PointFormData
	CancelURL    string
	PageTitle    string
}

type PointTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultPointTemplateName = PointTemplateName{
	AddForm:    "add_form_point.html",
	EditForm:   "edit_form_point.html",
	DeleteForm: "delete_form_point.html",
	Table:      "table_points.html",
}

var DefaultPointPathTemplate = "internal/templates/points/"
