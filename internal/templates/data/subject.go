package data

import (
	"net/url"
	"strconv"
)

type SubjectRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type SubjectContext struct {
	ID   int64
	Name string
}

type SubjectListItem struct {
	ID        int64
	Name      string
	EditURL   string
	DeleteURL string
}

func SubjectURL(base string, subjectID int64) string {
	return base + "?subject_id=" + url.QueryEscape(strconv.FormatInt(subjectID, 10))
}

var DefaultSubjectRoutes = SubjectRoutes{
	AddURL:    "/dashboard/questions/subjects/add",
	EditURL:   "/dashboard/questions/subjects/edit",
	DeleteURL: "/dashboard/questions/subjects/delete",
}

type SubjectPageData struct {
	Routes         DashboardRoutes
	SubjectRoutes  SubjectRoutes
	SubjectContext SubjectContext
	SubjectItems   []SubjectListItem
	CancelURL      string
	PageTitle      string
}

type SubjectTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultSubjectTemplateName = SubjectTemplateName{
	AddForm:    "add_form_subject.html",
	EditForm:   "edit_form_subject.html",
	DeleteForm: "delete_form_subject.html",
	Table:      "table_subjects.html",
}

var DefaultSubjectPathTemplate = "internal/templates/subjects/"
