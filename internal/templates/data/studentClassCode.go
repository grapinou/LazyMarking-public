package data

type StudentClassCodeRoutes struct {
	AddURL    string
	DeleteURL string
}

type StudentClassContext struct {
	ID        int64
	FirstName string
	LastName  string
}

type StudentClassListItem struct {
	ClassID   int64
	ClassName string
	DeleteURL string
}

type StudentClassListData struct {
	Student       StudentClassContext
	Items         []StudentClassListItem
	AddURL        string
	AllowedDelete bool
	NoClasses     bool
}

type StudentClassFormData struct {
	Student   StudentClassContext
	Classes   []StudentClassOption
	ReturnURL string
}

var DefaultStudentClassCodeRoutes = StudentClassCodeRoutes{
	AddURL:    "/dashboard/students-classcodes/add",
	DeleteURL: "/dashboard/students-classcodes/delete",
}

type StudentClassCodePageData struct {
	Routes                 DashboardRoutes
	StudentClassCodeRoutes StudentClassCodeRoutes
	PageTitle              string
	List                   StudentClassListData
	Form                   StudentClassFormData
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
