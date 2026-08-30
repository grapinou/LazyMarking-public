package classcodes

import (
	"net/url"
	"strconv"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func buildClassCodeListPageData(classCodes []db.ClassCode) data.ClassCodePageData {
	items := make([]data.ClassCodeListItem, 0, len(classCodes))
	for _, classCode := range classCodes {
		params := "?class_code_id=" + url.QueryEscape(strconv.FormatInt(classCode.ID, 10))
		items = append(items, data.ClassCodeListItem{
			ID:        classCode.ID,
			Name:      classCode.Name,
			EditURL:   data.DefaultClassCodeRoutes.EditURL + params,
			DeleteURL: data.DefaultClassCodeRoutes.DeleteURL + params,
		})
	}

	return data.ClassCodePageData{
		Routes:          data.DefaultDashboardRoutes,
		ClassCodeRoutes: data.DefaultClassCodeRoutes,
		PageTitle:       "class codes",
		List: data.ClassCodeListData{
			Items:     items,
			NoClasses: len(items) == 0,
		},
	}
}

func buildClassCodePageData(pageTitle string) data.ClassCodePageData {
	return data.ClassCodePageData{
		Routes:          data.DefaultDashboardRoutes,
		ClassCodeRoutes: data.DefaultClassCodeRoutes,
		PageTitle:       pageTitle,
	}
}

func buildClassCodeContextPageData(classCodeID int64, classCodeName, pageTitle string) data.ClassCodePageData {
	page := buildClassCodePageData(pageTitle)
	page.ClassCode = data.ClassCodeContext{ID: classCodeID, Name: classCodeName}
	return page
}
