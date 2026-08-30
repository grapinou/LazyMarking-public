package studentclasscode

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestBuildStudentClassListPageDataKeepsContextItemsURLsAndAllowedDelete(t *testing.T) {
	student := db.Student{ID: 7, FirstName: "Marie", LastName: "Curie", UserID: 1}
	classes := []config.ClassCode{{ID: 3, Name: "4e A"}, {ID: 8, Name: "Club sciences"}}
	page := buildStudentClassListPageData(student, classes)
	if page.List.Student != (data.StudentClassContext{ID: 7, FirstName: "Marie", LastName: "Curie"}) ||
		len(page.List.Items) != 2 || !page.List.AllowedDelete || page.List.NoClasses {
		t.Fatalf("list=%+v", page.List)
	}
	if page.List.AddURL != data.DefaultStudentClassCodeRoutes.AddURL+"?student_id=7" {
		t.Fatalf("add URL=%q", page.List.AddURL)
	}
	for index, expected := range []struct {
		id   int64
		name string
		url  string
	}{
		{3, "4e A", data.DefaultStudentClassCodeRoutes.DeleteURL + "?student_id=7&class_code_id=3"},
		{8, "Club sciences", data.DefaultStudentClassCodeRoutes.DeleteURL + "?student_id=7&class_code_id=8"},
	} {
		item := page.List.Items[index]
		if item.ClassID != expected.id || item.ClassName != expected.name || item.DeleteURL != expected.url {
			t.Fatalf("item %d=%+v", index, item)
		}
	}
}

func TestBuildStudentClassListPageDataPreservesDeleteRule(t *testing.T) {
	student := db.Student{ID: 7, FirstName: "Marie", LastName: "Curie", UserID: 1}
	one := buildStudentClassListPageData(student, []config.ClassCode{{ID: 3, Name: "4e A"}})
	if one.List.AllowedDelete || one.List.NoClasses {
		t.Fatalf("one class=%+v", one.List)
	}
	empty := buildStudentClassListPageData(student, nil)
	if empty.List.AllowedDelete || !empty.List.NoClasses || empty.List.Items == nil || len(empty.List.Items) != 0 {
		t.Fatalf("empty classes=%#v", empty.List)
	}
}

func TestBuildStudentClassFormPageData(t *testing.T) {
	student := db.Student{ID: 7, FirstName: "Marie", LastName: "Curie", UserID: 1}
	rows := []db.ListClassCodesNotAssignedToStudentRow{{ClassCodesID: 3, ClassCodesName: "4e A"}, {ClassCodesID: 8, ClassCodesName: "Club sciences"}}
	page := buildStudentClassFormPageData(student, rows)
	if page.Form.Student != (data.StudentClassContext{ID: 7, FirstName: "Marie", LastName: "Curie"}) ||
		len(page.Form.Classes) != 2 ||
		page.Form.Classes[0] != (data.StudentClassOption{ID: 3, Name: "4e A"}) ||
		page.Form.Classes[1] != (data.StudentClassOption{ID: 8, Name: "Club sciences"}) ||
		page.Form.ReturnURL != data.DefaultStudentRoutes.StudentClassCodesURL+"?student_id=7" {
		t.Fatalf("form=%+v", page.Form)
	}
	empty := buildStudentClassFormPageData(student, nil)
	if empty.Form.Classes == nil || len(empty.Form.Classes) != 0 {
		t.Fatalf("empty form classes=%#v", empty.Form.Classes)
	}
}

func TestStudentClassTemplatesRenderTypedDataAndPreserveContracts(t *testing.T) {
	student := db.Student{ID: 7, FirstName: "Marie", LastName: "Curie", UserID: 1}
	list := buildStudentClassListPageData(student, []config.ClassCode{{ID: 3, Name: "4e A"}, {ID: 8, Name: "Club sciences"}})
	form := buildStudentClassFormPageData(student, []db.ListClassCodesNotAssignedToStudentRow{{ClassCodesID: 5, ClassCodesName: "5e B"}})
	tests := []struct {
		name     string
		render   func(http.ResponseWriter, data.StudentClassCodePageData)
		page     data.StudentClassCodePageData
		expected []string
	}{
		{"list", RenderTableStudentClassCodesPage, list, []string{"Classes de l’élève", "Marie Curie", "4e A", "Club sciences", list.List.AddURL, data.DefaultDashboardRoutes.StudentURL, "Retour aux élèves", "Ajouter une classe", "Retirer", strings.ReplaceAll(list.List.Items[0].DeleteURL, "&", "&amp;"), strings.ReplaceAll(list.List.Items[1].DeleteURL, "&", "&amp;")}},
		{"add", RenderAddFormStudentClassCodePage, form, []string{"Ajouter une classe à l’élève", "Marie Curie", "action=\"" + data.DefaultStudentClassCodeRoutes.AddURL + "\" method=\"post\"", "name=\"student_id\" value=\"7\"", "name=\"class_code_id\"", "value=\"5\"", "5e B", "Ajouter la classe", strings.ReplaceAll(form.Form.ReturnURL, "&", "&amp;"), "Annuler"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := renderStudentClassPageForTest(t, test.render, test.page)
			for _, expected := range test.expected {
				if !strings.Contains(body, expected) {
					t.Errorf("body does not contain %q", expected)
				}
			}
			for _, obsolete := range []string{"Gestion des classes d'un élève", "Back to students", ">Sup<", "<p> - </p>"} {
				if strings.Contains(body, obsolete) {
					t.Errorf("body contains obsolete content %q", obsolete)
				}
			}
		})
	}
	oneClassBody := renderStudentClassPageForTest(t, RenderTableStudentClassCodesPage, buildStudentClassListPageData(student, []config.ClassCode{{ID: 3, Name: "4e A"}}))
	if strings.Contains(oneClassBody, data.DefaultStudentClassCodeRoutes.DeleteURL) {
		t.Fatal("single-class student exposes relation deletion")
	}
	for _, expected := range []string{"Classe obligatoire", "Impossible de retirer la dernière classe de l’élève."} {
		if !strings.Contains(oneClassBody, expected) {
			t.Errorf("single-class state does not contain %q", expected)
		}
	}
}

func TestStudentClassTemplatesRenderDefensiveEmptyStates(t *testing.T) {
	student := db.Student{ID: 7, FirstName: "Marie", LastName: "Curie", UserID: 1}
	emptyList := buildStudentClassListPageData(student, nil)
	listBody := renderStudentClassPageForTest(t, RenderTableStudentClassCodesPage, emptyList)
	for _, expected := range []string{"Aucune classe associée", "n’est actuellement rattaché à aucune classe", emptyList.List.AddURL, "Ajouter une classe"} {
		if !strings.Contains(listBody, expected) {
			t.Errorf("empty list does not contain %q", expected)
		}
	}

	emptyForm := buildStudentClassFormPageData(student, nil)
	formBody := renderStudentClassPageForTest(t, RenderAddFormStudentClassCodePage, emptyForm)
	for _, expected := range []string{"Marie Curie", "Toutes les classes disponibles sont déjà associées à cet élève.", "Retour aux classes de l’élève", strings.ReplaceAll(emptyForm.Form.ReturnURL, "&", "&amp;")} {
		if !strings.Contains(formBody, expected) {
			t.Errorf("empty form does not contain %q", expected)
		}
	}
	for _, unexpected := range []string{"name=\"class_code_id\"", "Ajouter la classe", "action=\"" + data.DefaultStudentClassCodeRoutes.AddURL + "\""} {
		if strings.Contains(formBody, unexpected) {
			t.Errorf("empty form unexpectedly contains %q", unexpected)
		}
	}
}

func TestStudentClassViewDataDoesNotUseExtraData(t *testing.T) {
	if _, exists := reflect.TypeFor[data.StudentClassCodePageData]().FieldByName("ExtraData"); exists {
		t.Fatal("StudentClassCodePageData exposes ExtraData")
	}
	dataSource, err := os.ReadFile("../../templates/data/studentClassCode.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dataSource), "any") || strings.Contains(string(dataSource), "ActionURLs") {
		t.Fatal("student class view data exposes dynamic or parallel action data")
	}
	for _, path := range []string{
		"../../templates/studentClassCodes/table_student_class_codes.html",
		"../../templates/studentClassCodes/add_form_student_class_code.html",
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "ExtraData") || strings.Contains(string(content), "index $.") {
			t.Fatalf("%s contains dynamic or parallel view data", path)
		}
	}
}

func renderStudentClassPageForTest(t *testing.T, render func(http.ResponseWriter, data.StudentClassCodePageData), page data.StudentClassCodePageData) string {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../../.."); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(current) }()
	response := httptest.NewRecorder()
	render(response, page)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	return response.Body.String()
}
