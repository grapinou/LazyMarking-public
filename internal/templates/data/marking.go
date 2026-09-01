package data

type MarkingRoutes struct {
	ServePDF          string
	ProcessingMarking string
	ProgressMarking   string
	SuccessURL        string
	ReviewCrop        string
}

var DefaultMarkingRoutes = MarkingRoutes{
	ServePDF:          "/dashboard/marking/servePDF",
	ProcessingMarking: "/dashboard/marking/processing",
	ProgressMarking:   "/dashboard/marking/progress",
	SuccessURL:        "/dashboard/marking/success",
	ReviewCrop:        "/dashboard/marking/review/crop",
}

type MarkingPageData struct {
	Routes        DashboardRoutes
	MarkingRoutes MarkingRoutes
	PageTitle     string
	ExtraData     map[string]any
}

type MarkingResultPageData struct {
	Routes        DashboardRoutes
	MarkingRoutes MarkingRoutes
	PageTitle     string
	JobID         int64
	Review        MarkingReviewStatusView
	Artifacts     MarkingArtifactLinksView
	NonCorrected  MarkingNonCorrectedSummaryView
	Notice        NoticeView
}

type MarkingReviewStatusView struct {
	Status             string
	TotalCandidates    int64
	ReviewedCandidates int64
	PendingCandidates  int64
	ArtifactsCurrent   bool
	ReviewURL          string
}

type MarkingArtifactLinksView struct {
	CorrectedPDFURL    string
	MarkTablePDFURL    string
	NonCorrectedPDFURL string
}

type MarkingNonCorrectedSummaryView struct {
	Incomplete int64
	Errors     int64
	NotSeen    int64
	Total      int64
}

type NoticeView struct {
	Title string
	Text  string
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
