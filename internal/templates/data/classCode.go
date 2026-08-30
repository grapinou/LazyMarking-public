package data

type ClassCodeRoutes struct {
	AddURL    string
	EditURL   string
	DeleteURL string
}

type ClassCodeContext struct {
	ID   int64
	Name string
}

type ClassCodeListItem struct {
	ID        int64
	Name      string
	EditURL   string
	DeleteURL string
}

type ClassCodeListData struct {
	Items     []ClassCodeListItem
	NoClasses bool
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
	List            ClassCodeListData
	ClassCode       ClassCodeContext
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
