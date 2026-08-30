package studentclasscode

import (
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func buildStudentClassListPageData(student db.Student, classCodes []config.ClassCode) data.StudentClassCodePageData {
	studentContext := buildStudentClassContext(student)
	items := make([]data.StudentClassListItem, 0, len(classCodes))
	for _, classCode := range classCodes {
		params := "?student_id=" + url.QueryEscape(strconv.FormatInt(student.ID, 10)) +
			"&class_code_id=" + url.QueryEscape(strconv.FormatInt(classCode.ID, 10))
		items = append(items, data.StudentClassListItem{
			ClassID:   classCode.ID,
			ClassName: classCode.Name,
			DeleteURL: data.DefaultStudentClassCodeRoutes.DeleteURL + params,
		})
	}

	addURL := data.DefaultStudentClassCodeRoutes.AddURL +
		"?student_id=" + url.QueryEscape(strconv.FormatInt(student.ID, 10))

	return data.StudentClassCodePageData{
		Routes:                 data.DefaultDashboardRoutes,
		StudentClassCodeRoutes: data.DefaultStudentClassCodeRoutes,
		PageTitle:              "student-classcodes",
		List: data.StudentClassListData{
			Student:       studentContext,
			Items:         items,
			AddURL:        addURL,
			AllowedDelete: len(items) > 1,
			NoClasses:     len(items) == 0,
		},
	}
}

func buildStudentClassFormPageData(student db.Student, classCodes []db.ListClassCodesNotAssignedToStudentRow) data.StudentClassCodePageData {
	classes := make([]data.StudentClassOption, 0, len(classCodes))
	for _, classCode := range classCodes {
		classes = append(classes, data.StudentClassOption{ID: classCode.ClassCodesID, Name: classCode.ClassCodesName})
	}
	returnURL := data.DefaultStudentRoutes.StudentClassCodesURL +
		"?student_id=" + url.QueryEscape(strconv.FormatInt(student.ID, 10))

	return data.StudentClassCodePageData{
		Routes:                 data.DefaultDashboardRoutes,
		StudentClassCodeRoutes: data.DefaultStudentClassCodeRoutes,
		PageTitle:              "add extra class code",
		Form: data.StudentClassFormData{
			Student:   buildStudentClassContext(student),
			Classes:   classes,
			ReturnURL: returnURL,
		},
	}
}

func buildStudentClassContext(student db.Student) data.StudentClassContext {
	return data.StudentClassContext{
		ID:        student.ID,
		FirstName: student.FirstName,
		LastName:  student.LastName,
	}
}

func buildStudentClassDeletePageData(student db.Student, classCodeID int64, classCodeName string, canDelete bool) data.StudentClassCodePageData {
	returnURL := data.DefaultStudentRoutes.StudentClassCodesURL +
		"?student_id=" + url.QueryEscape(strconv.FormatInt(student.ID, 10))

	return data.StudentClassCodePageData{
		Routes:                 data.DefaultDashboardRoutes,
		StudentClassCodeRoutes: data.DefaultStudentClassCodeRoutes,
		PageTitle:              "Retirer une classe de l’élève",
		Delete: data.StudentClassRelationDeleteData{
			Student:   buildStudentClassContext(student),
			Class:     data.StudentClassOption{ID: classCodeID, Name: classCodeName},
			ActionURL: data.DefaultStudentClassCodeRoutes.DeleteURL,
			ReturnURL: returnURL,
			CanDelete: canDelete,
		},
	}
}
