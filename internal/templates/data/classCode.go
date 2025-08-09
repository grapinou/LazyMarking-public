package data

type ClassCodeRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type ClassCodeActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultClassCodeRoutes = ClassCodeRoutes{
	AddURL:    "/dashboard/students/classcodes/add",
	EditURL:   "/dashboard/students/classcodes/edit",
	DeleteURL: "/dashboard/students/classcodes/delete",
}

type ClassCodePageData struct {
	Routes          DashboardRoutes
	ClassCodeRoutes ClassCodeRoutes
	PageTitle       string
	ExtraData       map[string]any
}

type ClassCodeTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultClassCodeTemplateName = ClassCodeTemplateName{
	AddForm:    "add_form_class_code.html",
	EditForm:   "edit_form_class_code.html",
	DeleteForm: "delete_form_class_code.html",
	Table:      "table_class_codes.html",
}

var DefaultClassCodePathTemplate = "internal/templates/classcodes/"
