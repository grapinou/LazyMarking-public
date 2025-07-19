package data

var (
	ErrorMessageURL   = "/dashboard/errorsmessages"
	ErrorPathTemplate = "internal/templates/errors/"
	ErrorTemplateName = "question_feature_error.html"
)

type ErrorPageData struct {
	Routes    DashboardRoutes
	PageTitle string
	ExtraData map[string]any
}
