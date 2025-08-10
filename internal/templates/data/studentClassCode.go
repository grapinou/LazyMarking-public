package data

type StudentClassCodeRoutes struct {
	AddURL    string
	DeleteURL string
}

type StudentClassCodeActionURLs struct {
	DeleteURL string
}

var DefaultStudentClassCodeRoutes = StudentClassCodeRoutes{
	AddURL:    "/dashboard/students-classcodes/add",
	DeleteURL: "/dashboard/students-classcodes/delete",
}

type StudentClassCodePageData struct {
	Routes                 DashboardRoutes
	StudentClassCodeRoutes StudentClassCodeRoutes
	PageTitle              string
	ExtraData              map[string]any
}

type StudentClassCodeTemplateName struct {
	AddForm    string
	DeleteForm string
	Table      string
}

var DefaultStudentClassCodeTemplateName = StudentClassCodeTemplateName{
	AddForm:    "add_form_student_class_code.html",
	DeleteForm: "delete_form_student_class_code.html",
	Table:      "table_student_class_codes.html",
}

var DefaultStudentClassCodePathTemplate = "internal/templates/studentClassCodes/"
