package data

type MarkingRoutes struct {
	ProcessingMarking string
	ProgressMarking   string
	SuccessURL        string
}

var DefaultMarkingRoutes = MarkingRoutes{
	ProcessingMarking: "/dashboard/marking/processing",
	ProgressMarking:   "/dashboard/marking/progress",
	SuccessURL:        "/dashboard/marking/success",
}

type MarkingPageData struct {
	Routes        DashboardRoutes
	MarkingRoutes MarkingRoutes
	PageTitle     string
	ExtraData     map[string]any
}

type MarkingTemplateName struct {
	Progress string
	Table    string
}

var DefaultMarkingTemplateName = MarkingTemplateName{
	Progress: "progress_marking.html",
	Table:    "table_marking.html",
}

var DefaultMarkingPathTemplate = "internal/templates/marking/"
