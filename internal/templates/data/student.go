package data

type StudentRoutes struct {
	ClassCodesURL        string // gère le nom de la classe TERM1, 1ERE3, 5è1...
	AddCSVURL            string
	AddURL               string
	EditURL              string
	DeleteURL            string
	StudentClassCodesURL string
	DeleteAllStudentURL  string
}

var DefaultStudentRoutes = StudentRoutes{
	ClassCodesURL:        "/dashboard/students/classcodes",
	AddCSVURL:            "/dashboard/students/addcsv",
	AddURL:               "/dashboard/students/add",
	EditURL:              "/dashboard/students/edit",
	DeleteURL:            "/dashboard/students/delete",
	StudentClassCodesURL: "/dashboard/student-classcodes",
	DeleteAllStudentURL:  "/dashboard/students/delete-all-students",
}

type StudentClassOption struct {
	ID   int64
	Name string
}

type StudentListItem struct {
	ID        int64
	FirstName string
	LastName  string
	Classes   []StudentClassOption

	EditURL              string
	DeleteURL            string
	StudentClassCodesURL string
}

type StudentListData struct {
	Items              []StudentListItem
	Classes            []StudentClassOption
	CurrentClassFilter string
	NoStudents         bool
	NoClasses          bool
}

type StudentFormData struct {
	Classes []StudentClassOption
}

type StudentContext struct {
	ID        int64
	FirstName string
	LastName  string
}

type StudentClassDeleteData struct {
	ID   int64
	Name string
}

type StudentPageData struct {
	Routes        DashboardRoutes
	StudentRoutes StudentRoutes
	PageTitle     string
	List          StudentListData
	Form          StudentFormData
	Student       StudentContext
	ClassDelete   StudentClassDeleteData
}

type StudentTemplateName struct {
	AddForm              string
	EditForm             string
	DeleteForm           string
	Table                string
	AddCSVForm           string
	DeleteFormAllStudent string
}

var DefaultStudentTemplateName = StudentTemplateName{
	AddForm:              "add_form_student.html",
	EditForm:             "edit_form_student.html",
	DeleteForm:           "delete_form_student.html",
	Table:                "table_students.html",
	AddCSVForm:           "add_csv_form_student.html",
	DeleteFormAllStudent: "delete_form_all_students.html",
}

var DefaultStudentPathTemplate = "internal/templates/students/"
