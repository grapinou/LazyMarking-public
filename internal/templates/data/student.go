package data

type StudentRoutes struct {
	ClassCodesURL string // gère le nom de la classe TERM1, 1ERE3, 5è1...
	AddCSVURL     string
	AddURL        string
	EditURL       string
	DeleteURL     string
}

type StudentActionURLs struct {
	EditURL   string
	DeleteURL string
}

var DefaultStudentRoutes = StudentRoutes{
	ClassCodesURL: "/dashboard/students/classcodes",
	AddCSVURL:     "/dashboard/students/addcsv",
	AddURL:        "/dashboard/students/add",
	EditURL:       "/dashboard/students/add",
	DeleteURL:     "/dashboard/students/add",
}

type StudentPageData struct {
	Routes        DashboardRoutes
	StudentRoutes StudentRoutes
	PageTitle     string
	ExtraData     map[string]any
}

type StudentTemplateName struct {
	AddForm    string
	EditForm   string
	DeleteForm string
	Table      string
}

var DefaultStudentTemplateName = StudentTemplateName{
	AddForm:    "add_form_student.html",
	EditForm:   "edit_form_student.html",
	DeleteForm: "delete_form_student.html",
	Table:      "table_students.html",
}

var DefaultStudentPathTemplate = "internal/templates/students/"
