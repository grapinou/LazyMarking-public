package data

type MarkingRoutes struct {
	GenerationResults   string
	GenerationPDF       string
	ServePDF            string
	ProcessingMarking   string
	ProgressMarking     string
	SuccessURL          string
	ReviewURL           string
	ReviewApply         string
	ReviewCrop          string
	ArtifactsRegenerate string
}

var DefaultMarkingRoutes = MarkingRoutes{
	GenerationResults:   "/dashboard/marking/results",
	GenerationPDF:       "/dashboard/marking/results/pdf",
	ServePDF:            "/dashboard/marking/servePDF",
	ProcessingMarking:   "/dashboard/marking/processing",
	ProgressMarking:     "/dashboard/marking/progress",
	SuccessURL:          "/dashboard/marking/success",
	ReviewURL:           "/dashboard/marking/review",
	ReviewApply:         "/dashboard/marking/review/apply",
	ReviewCrop:          "/dashboard/marking/review/crop",
	ArtifactsRegenerate: "/dashboard/marking/artifacts/regenerate",
}

type MarkingPageData struct {
	SelectedGenerationID   int64
	SelectedGenerationName string
	Routes                 DashboardRoutes
	MarkingRoutes          MarkingRoutes
	PageTitle              string
	ExtraData              map[string]any
	RecentJobs             []MarkingJobHistoryView
}

type MarkingJobHistoryView struct {
	CompletedLabel string
	ResultURL      string
	JobID          int64
	ExamName       string
	ClassCodeName  string
	SourceFilename string
	StatusLabel    string
	Detail         string
	ActionURL      string
	ActionLabel    string
}

type MarkingResultPageData struct {
	GenerationURL string
	Routes        DashboardRoutes
	MarkingRoutes MarkingRoutes
	PageTitle     string
	JobID         int64
	Review        MarkingReviewStatusView
	Artifacts     MarkingArtifactLinksView
	NonCorrected  MarkingNonCorrectedSummaryView
	Notice        NoticeView
	Alert         NoticeView
}

type MarkingReviewStatusView struct {
	Status             string
	TotalCandidates    int64
	ReviewedCandidates int64
	PendingCandidates  int64
	ArtifactsCurrent   bool
	ReviewURL          string
	RevisitURL         string
}

type MarkingExamSummaryView struct {
	Total         int64
	Corrected     int64
	PendingReview int64
	Issues        int64
	NotSeen       int64
	Results       []MarkingExamResultView
}

type MarkingExamResultView struct {
	StudentName string
	StatusLabel string
	ScoreLabel  string
	HasScore    bool
	Pending     bool
	SourceJobID int64
}

// Both HTML and PDF consume this presentation of a generation's current state.
type MarkingGenerationPageData struct {
	Routes       DashboardRoutes
	PageTitle    string
	GenerationID int64
	ExamName     string
	ClassName    string
	AddCopiesURL string
	PDFURL       string
	Summary      MarkingExamSummaryView
	Pedagogy     MarkingPedagogicalSummaryView
	Imports      []MarkingJobHistoryView
}

type MarkingPedagogicalSummaryView struct {
	IncludedCopies int
	DetailedCopies int
	ExcludedCopies int
	ScoreGroups    []MarkingScoreStatisticsView
	Questions      []MarkingQuestionStatisticsView
	Skills         []MarkingSuccessRateView
	ThemeSkills    []MarkingSuccessRateView
}

type MarkingScoreStatisticsView struct {
	Count  int
	Total  int64
	Mean   string
	Median string
	StdDev string
}

type MarkingQuestionStatisticsView struct {
	Label   string
	Count   int
	Correct int
	Success string
}

type MarkingSuccessRateView struct {
	Label   string
	Success string
}

type MarkingArtifactLinksView struct {
	CorrectedPDFURL    string
	MarkTablePDFURL    string
	NonCorrectedPDFURL string
	RegenerateURL      string
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
	Routes               DashboardRoutes
	MarkingRoutes        MarkingRoutes
	PageTitle            string
	JobID                int64
	Position             int64
	Total                int64
	Remaining            int64
	JobRevision          int64
	AnswerReviewRevision *int64
	Candidate            MarkingReviewCandidateView
	ResultURL            string
	PreviousURL          string
	NextURL              string
	Notice               NoticeView
}

type MarkingReviewCandidateView struct {
	DetectionID        int64
	StudentDisplayName string
	QuestionNumber     int64
	AnswerLabel        string
	DetectedChecked    bool
	HasReview          bool
	ReviewedChecked    bool
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
