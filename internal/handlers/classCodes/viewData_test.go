package classcodes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestBuildClassCodeListPageData(t *testing.T) {
	page := buildClassCodeListPageData([]db.ClassCode{
		{ID: 3, Name: "4e A", UserID: 1},
		{ID: 8, Name: "3e B", UserID: 1},
	})
	if page.List.NoClasses || len(page.List.Items) != 2 {
		t.Fatalf("list=%+v", page.List)
	}
	for index, expected := range []struct {
		id   int64
		name string
	}{{3, "4e A"}, {8, "3e B"}} {
		item := page.List.Items[index]
		if item.ID != expected.id || item.Name != expected.name {
			t.Fatalf("item %d=%+v", index, item)
		}
		suffix := "?class_code_id=" + strconv.FormatInt(expected.id, 10)
		if item.EditURL != data.DefaultClassCodeRoutes.EditURL+suffix || item.DeleteURL != data.DefaultClassCodeRoutes.DeleteURL+suffix {
			t.Fatalf("item URLs=%+v", item)
		}
	}
}

func TestBuildClassCodePageDataRepresentsEmptyAndFormContexts(t *testing.T) {
	empty := buildClassCodeListPageData(nil)
	if !empty.List.NoClasses || empty.List.Items == nil || len(empty.List.Items) != 0 {
		t.Fatalf("empty list=%#v", empty.List)
	}
	add := buildClassCodePageData("add class code")
	if add.PageTitle != "add class code" || add.ClassCode != (data.ClassCodeContext{}) {
		t.Fatalf("add=%+v", add)
	}
	for _, title := range []string{"edit class code", "delete class code"} {
		page := buildClassCodeContextPageData(7, "6e C", title)
		if page.PageTitle != title || page.ClassCode != (data.ClassCodeContext{ID: 7, Name: "6e C"}) {
			t.Fatalf("context=%+v", page)
		}
	}
}

func TestClassCodeTemplatesRenderTypedDataAndPreserveContracts(t *testing.T) {
	list := buildClassCodeListPageData([]db.ClassCode{{ID: 3, Name: "4e A", UserID: 1}})
	edit := buildClassCodeContextPageData(3, "4e A", "edit")
	deletePage := buildClassCodeContextPageData(3, "4e A", "delete")
	tests := []struct {
		name     string
		render   func(http.ResponseWriter, data.ClassCodePageData)
		page     data.ClassCodePageData
		expected []string
	}{
		{"list", RenderTableClassCodePage, list, []string{"4e A", list.List.Items[0].EditURL, list.List.Items[0].DeleteURL}},
		{"add", RenderAddFormClassCodePage, buildClassCodePageData("add"), []string{data.DefaultClassCodeRoutes.AddURL, "name=\"class_code\""}},
		{"edit", RenderEditFormClassCodePage, edit, []string{data.DefaultClassCodeRoutes.EditURL, "name=\"class_code_id\" value=\"3\"", "name=\"new_class_code\"", "value=\"4e A\""}},
		{"delete", RenderDeleteFormClassCodePage, deletePage, []string{data.DefaultClassCodeRoutes.DeleteURL, "name=\"class_code_id\" value=\"3\"", "4e A"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := renderClassCodePageForTest(t, test.render, test.page)
			for _, expected := range test.expected {
				if !strings.Contains(body, expected) {
					t.Errorf("body does not contain %q", expected)
				}
			}
		})
	}
}

func TestClassCodeViewDataDoesNotUseExtraData(t *testing.T) {
	if _, exists := reflect.TypeFor[data.ClassCodePageData]().FieldByName("ExtraData"); exists {
		t.Fatal("ClassCodePageData exposes ExtraData")
	}
	dataSource, err := os.ReadFile("../../templates/data/classCode.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dataSource), "any") || strings.Contains(string(dataSource), "ActionURLs") {
		t.Fatal("class code view data exposes dynamic or parallel action data")
	}
	for _, path := range []string{
		"../../templates/classcodes/table_class_codes.html",
		"../../templates/classcodes/add_form_class_code.html",
		"../../templates/classcodes/edit_form_class_code.html",
		"../../templates/classcodes/delete_form_class_code.html",
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

func renderClassCodePageForTest(t *testing.T, render func(http.ResponseWriter, data.ClassCodePageData), page data.ClassCodePageData) string {
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
