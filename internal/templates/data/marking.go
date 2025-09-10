package data

type MarkingRoutes struct {
	ProcessingMarking string
	SuccessURL        string
}

var DefaultMarkingRoutes = MarkingRoutes{
	ProcessingMarking: "/dashboard/marking/processing",
	SuccessURL:        "/dashboard/marking/success",
}

type MarkingPageData struct {
	Routes        DashboardRoutes
	MarkingRoutes MarkingRoutes
	PageTitle     string
	ExtraData     map[string]any
}

type MarkingTemplateName struct {
	Processing string
	Table      string
}

var DefaultMarkingTemplateName = MarkingTemplateName{
	Processing: "processing_marking.html",
	Table:      "table_marking.html",
}

var DefaultMarkingPathTemplate = "internal/templates/marking/"
