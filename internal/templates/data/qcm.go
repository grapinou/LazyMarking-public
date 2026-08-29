package data

import (
	"net/url"
	"strconv"
)

type QCMContext struct {
	ID   int64
	Name string
}

func QCMURL(base string, qcmID int64) string {
	return base + "?qcm_id=" + url.QueryEscape(strconv.FormatInt(qcmID, 10))
}

type QCMRoutes struct {
	AddURL              string
	EditURL             string
	DeleteURL           string
	AddQuestionURL      string
	PreviewURL          string
	PreviewLandscapeURL string
}

type QCMListItem struct {
	ID                  int64
	Name                string
	QuestionCount       int64
	CompositionURL      string
	PreviewURL          string
	PreviewLandscapeURL string
	EditURL             string
	DeleteURL           string
}

var DefaultQCMRoutes = QCMRoutes{
	AddURL:              "/dashboard/qcm/add",
	EditURL:             "/dashboard/qcm/edit",
	DeleteURL:           "/dashboard/qcm/delete",
	AddQuestionURL:      "/dashboard/qcm/qcmquestion",
	PreviewURL:          "/dashboard/qcm/previewqcm",
	PreviewLandscapeURL: "/dashboard/qcm/previewqcmlandscape",
}

type QCMPageData struct {
	Routes     DashboardRoutes
	QCMRoutes  QCMRoutes
	QCMContext QCMContext
	QCMItems   []QCMListItem
	PageTitle  string
}

type QCMTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultQCMTemplateName = QCMTemplateName{
	AddForm:    "add_form_qcm.html",
	EditForm:   "edit_form_qcm.html",
	DeleteForm: "delete_form_qcm.html",
	Table:      "table_qcm.html",
}

var DefaultQCMPathTemplate = "internal/templates/qcm/"
