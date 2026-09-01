package data

type MarkingRoutes struct {
	ServePDF          string
	ProcessingMarking string
	ProgressMarking   string
	SuccessURL        string
	ReviewURL         string
	ReviewCrop        string
}

var DefaultMarkingRoutes = MarkingRoutes{
	ServePDF:          "/dashboard/marking/servePDF",
	ProcessingMarking: "/dashboard/marking/processing",
	ProgressMarking:   "/dashboard/marking/progress",
	SuccessURL:        "/dashboard/marking/success",
	ReviewURL:         "/dashboard/marking/review",
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

type MarkingReviewPageData struct {
	Routes        DashboardRoutes
	MarkingRoutes MarkingRoutes
	PageTitle     string
	JobID         int64
	Position      int64
	Total         int64
	Remaining     int64
	JobRevision   int64
	Candidate     MarkingReviewCandidateView
	ResultURL     string
}

type MarkingReviewCandidateView struct {
	DetectionID        int64
	StudentDisplayName string
	QuestionNumber     int64
	AnswerLabel        string
	DetectedChecked    bool
	CropURL            string
}

type MarkingTemplateName struct {
	Success  string
	Progress string
	Table    string
	Review   string
}

var DefaultMarkingTemplateName = MarkingTemplateName{
	Success:  "success_marking_processing.html",
	Progress: "progress_marking.html",
	Table:    "table_marking.html",
	Review:   "review.html",
}

var DefaultMarkingPathTemplate = "internal/templates/marking/"
