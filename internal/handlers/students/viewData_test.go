package students

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestBuildStudentListPageDataGroupsClassesAndActionURLs(t *testing.T) {
	rows := []db.GetStudentsWithClassesRow{
		{StudentID: 1, StudentFirstName: "Ada", StudentLastName: "Lovelace", ClassID: sql.NullInt64{Int64: 10, Valid: true}, ClassName: sql.NullString{String: "4e A", Valid: true}},
		{StudentID: 2, StudentFirstName: "Alan", StudentLastName: "Turing", ClassID: sql.NullInt64{Int64: 10, Valid: true}, ClassName: sql.NullString{String: "4e A", Valid: true}},
		{StudentID: 1, StudentFirstName: "Ada", StudentLastName: "Lovelace", ClassID: sql.NullInt64{Int64: 11, Valid: true}, ClassName: sql.NullString{String: "Club sciences", Valid: true}},
		{StudentID: 3, StudentFirstName: "Grace", StudentLastName: "Hopper"},
	}
	page := buildStudentListPageData(rows, []db.ListClassCodesByUserRow{{ID: 10, Name: "4e A"}, {ID: 11, Name: "Club sciences"}}, "4e A")

	if page.List.NoStudents || page.List.NoClasses || page.List.CurrentClassFilter != "4e A" || len(page.List.Items) != 3 || len(page.List.Classes) != 2 {
		t.Fatalf("list=%+v", page.List)
	}
	ada := page.List.Items[0]
	if ada.ID != 1 || ada.FirstName != "Ada" || ada.LastName != "Lovelace" || len(ada.Classes) != 2 || ada.Classes[0].Name != "4e A" || ada.Classes[1].Name != "Club sciences" {
		t.Fatalf("Ada item=%+v", ada)
	}
	if ada.EditURL != data.DefaultStudentRoutes.EditURL+"?student_id=1" ||
		ada.DeleteURL != data.DefaultStudentRoutes.DeleteURL+"?student_id=1" ||
		ada.StudentClassCodesURL != data.DefaultStudentRoutes.StudentClassCodesURL+"?student_id=1" {
		t.Fatalf("Ada URLs=%+v", ada)
	}
	if got := page.List.Items[2]; got.ID != 3 || got.Classes == nil || len(got.Classes) != 0 {
		t.Fatalf("student without class=%+v, want typed empty class slice", got)
	}
}

func TestBuildStudentListPageDataRepresentsEmptyStates(t *testing.T) {
	page := buildStudentListPageData(nil, nil, "")
	if !page.List.NoStudents || !page.List.NoClasses || page.List.Items == nil || page.List.Classes == nil || len(page.List.Items) != 0 || len(page.List.Classes) != 0 {
		t.Fatalf("empty list=%#v", page.List)
	}
}

func TestBuildStudentFormAndContextPageData(t *testing.T) {
	form := buildStudentFormPageData([]db.ClassCode{{ID: 5, Name: "5e B", UserID: 1}}, "add student")
	if form.PageTitle != "add student" || len(form.Form.Classes) != 1 || form.Form.Classes[0] != (data.StudentClassOption{ID: 5, Name: "5e B"}) {
		t.Fatalf("form=%+v", form)
	}
	emptyForm := buildStudentFormPageData(nil, "add student")
	if emptyForm.Form.Classes == nil || len(emptyForm.Form.Classes) != 0 {
		t.Fatalf("empty form classes=%#v, want typed empty slice", emptyForm.Form.Classes)
	}

	student := db.Student{ID: 7, FirstName: "Marie", LastName: "Curie", UserID: 1}
	for _, title := range []string{"edit student", "delete student"} {
		page := buildStudentContextPageData(student, title)
		if page.PageTitle != title || page.Student != (data.StudentContext{ID: 7, FirstName: "Marie", LastName: "Curie"}) {
			t.Fatalf("context page=%+v", page)
		}
	}

	classDelete := buildStudentClassDeletePageData(9, "3e C")
	if classDelete.ClassDelete != (data.StudentClassDeleteData{ID: 9, Name: "3e C"}) {
		t.Fatalf("class delete=%+v", classDelete)
	}
}

func TestStudentTemplatesRenderTypedPageData(t *testing.T) {
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(current) })

	student := data.StudentContext{ID: 7, FirstName: "Marie", LastName: "Curie"}
	class := data.StudentClassOption{ID: 5, Name: "5e B"}
	pages := []struct {
		name   string
		render func(http.ResponseWriter, data.StudentPageData)
		page   data.StudentPageData
	}{
		{"list", RenderTableStudentPage, data.StudentPageData{Routes: data.DefaultDashboardRoutes, StudentRoutes: data.DefaultStudentRoutes, PageTitle: "students", List: data.StudentListData{Items: []data.StudentListItem{{ID: 7, FirstName: "Marie", LastName: "Curie", Classes: []data.StudentClassOption{class}, EditURL: "/edit", DeleteURL: "/delete", StudentClassCodesURL: "/classes"}}, Classes: []data.StudentClassOption{class}}}},
		{"add", RenderAddFormStudentPage, data.StudentPageData{Routes: data.DefaultDashboardRoutes, StudentRoutes: data.DefaultStudentRoutes, PageTitle: "add", Form: data.StudentFormData{Classes: []data.StudentClassOption{class}}}},
		{"edit", RenderEditFormStudentPage, data.StudentPageData{Routes: data.DefaultDashboardRoutes, StudentRoutes: data.DefaultStudentRoutes, PageTitle: "edit", Student: student}},
		{"delete", RenderDeleteFormStudentPage, data.StudentPageData{Routes: data.DefaultDashboardRoutes, StudentRoutes: data.DefaultStudentRoutes, PageTitle: "delete", Student: student}},
		{"csv", RenderAddCSVFormStudentPage, data.StudentPageData{Routes: data.DefaultDashboardRoutes, StudentRoutes: data.DefaultStudentRoutes, PageTitle: "csv", Form: data.StudentFormData{Classes: []data.StudentClassOption{class}}}},
		{"delete class", RenderDeleteFormAllStudentsPage, data.StudentPageData{Routes: data.DefaultDashboardRoutes, StudentRoutes: data.DefaultStudentRoutes, PageTitle: "delete class", ClassDelete: data.StudentClassDeleteData{ID: 5, Name: "5e B"}}},
	}
	for _, test := range pages {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			test.render(response, test.page)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
			}
		})
	}
}

func TestStudentViewDataAndTemplatesDoNotUseExtraData(t *testing.T) {
	if _, exists := reflect.TypeFor[data.StudentPageData]().FieldByName("ExtraData"); exists {
		t.Fatal("StudentPageData exposes ExtraData")
	}
	for _, path := range []string{
		"../../templates/students/table_students.html",
		"../../templates/students/add_form_student.html",
		"../../templates/students/edit_form_student.html",
		"../../templates/students/delete_form_student.html",
		"../../templates/students/add_csv_form_student.html",
		"../../templates/students/delete_form_all_students.html",
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "ExtraData") {
			t.Fatalf("%s still references ExtraData", path)
		}
	}
}
