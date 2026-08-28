package data

type ImageRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

var DefaultImageRoutes = ImageRoutes{
	AddURL:    "/dashboard/questions/images/add",
	EditURL:   "/dashboard/questions/images/edit",
	DeleteURL: "/dashboard/questions/images/delete",
}

type ImagePageData struct {
	Routes          DashboardRoutes
	ImageRoutes     ImageRoutes
	QuestionContext QuestionContext
	PageTitle       string
	ExtraData       map[string]any
}

type ImageTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultImageTemplateName = ImageTemplateName{
	AddForm:    "add_form_image.html",
	EditForm:   "edit_form_image.html",
	DeleteForm: "delete_form_image.html",
	Table:      "table_image.html",
}

var DefaultImagePathTemplate = "internal/templates/images/"
