package students

import (
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func buildStudentListPageData(rows []db.GetStudentsWithClassesRow, classes []db.ListClassCodesByUserRow, currentFilter string) data.StudentPageData {
	items := buildStudentListItems(rows, data.DefaultStudentRoutes)
	classOptions := make([]data.StudentClassOption, 0, len(classes))
	for _, class := range classes {
		classOptions = append(classOptions, data.StudentClassOption{ID: class.ID, Name: class.Name})
	}
	return data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     "students",
		List: data.StudentListData{
			Items:              items,
			Classes:            classOptions,
			CurrentClassFilter: currentFilter,
			NoStudents:         len(items) == 0,
			NoClasses:          len(classOptions) == 0,
		},
	}
}

func buildStudentListItems(rows []db.GetStudentsWithClassesRow, routes data.StudentRoutes) []data.StudentListItem {
	items := make([]data.StudentListItem, 0)
	itemIndexes := make(map[int64]int)
	for _, row := range rows {
		index, exists := itemIndexes[row.StudentID]
		if !exists {
			params := "?student_id=" + url.QueryEscape(strconv.FormatInt(row.StudentID, 10))
			items = append(items, data.StudentListItem{
				ID:                   row.StudentID,
				FirstName:            row.StudentFirstName,
				LastName:             row.StudentLastName,
				Classes:              make([]data.StudentClassOption, 0),
				EditURL:              routes.EditURL + params,
				DeleteURL:            routes.DeleteURL + params,
				StudentClassCodesURL: routes.StudentClassCodesURL + params,
			})
			index = len(items) - 1
			itemIndexes[row.StudentID] = index
		}
		if row.ClassID.Valid && row.ClassName.Valid {
			items[index].Classes = append(items[index].Classes, data.StudentClassOption{
				ID:   row.ClassID.Int64,
				Name: row.ClassName.String,
			})
		}
	}
	return items
}

func buildStudentFormPageData(classes []db.ClassCode, pageTitle string) data.StudentPageData {
	classOptions := make([]data.StudentClassOption, 0, len(classes))
	for _, class := range classes {
		classOptions = append(classOptions, data.StudentClassOption{ID: class.ID, Name: class.Name})
	}
	return data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     pageTitle,
		Form:          data.StudentFormData{Classes: classOptions},
	}
}

func buildStudentContextPageData(student db.Student, pageTitle string) data.StudentPageData {
	return data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     pageTitle,
		Student: data.StudentContext{
			ID:        student.ID,
			FirstName: student.FirstName,
			LastName:  student.LastName,
		},
	}
}

func buildStudentClassDeletePageData(classID int64, className string) data.StudentPageData {
	return data.StudentPageData{
		Routes:        data.DefaultDashboardRoutes,
		StudentRoutes: data.DefaultStudentRoutes,
		PageTitle:     "delete all student",
		ClassDelete:   data.StudentClassDeleteData{ID: classID, Name: className},
	}
}
