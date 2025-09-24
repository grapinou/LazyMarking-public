package data

type MarkingRoutes struct {
	ServePDF          string
	ProcessingMarking string
	ProgressMarking   string
	SuccessURL        string
}

var DefaultMarkingRoutes = MarkingRoutes{
	ServePDF:          "/dashboard/marking/servePDF",
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
	Success  string
	Progress string
	Table    string
}

var DefaultMarkingTemplateName = MarkingTemplateName{
	Success:  "success_marking_processing.html",
	Progress: "progress_marking.html",
	Table:    "table_marking.html",
}

var DefaultMarkingPathTemplate = "internal/templates/marking/"
