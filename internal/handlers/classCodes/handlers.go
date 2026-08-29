package classcodes

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TableClassCodesHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From TableClassCodesHandler -> tools.CheckRequest return not ok")
		return
	}

	classCodesDB, err := queries.GetAllClassCodes(r.Context(), userID)
	if err != nil {
		log.Printf("From TableClassCodesHandler -> GetAllClassCodes DB error: %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	noClassCode := true
	if len(classCodesDB) > 0 {
		noClassCode = false
	}

	var actionsURLParameters []data.ClassCodeActionURLs
	if !noClassCode {
		for _, classCode := range classCodesDB {
			params := "?class_code_id=" + url.QueryEscape(strconv.FormatInt(classCode.ID, 10))
			editURL := data.DefaultClassCodeRoutes.EditURL + params
			deleteURL := data.DefaultClassCodeRoutes.DeleteURL + params

			actionsURLParameters = append(actionsURLParameters, data.ClassCodeActionURLs{
				EditURL:   editURL,
				DeleteURL: deleteURL,
			})
		}
	}

	dataPage := data.ClassCodePageData{
		Routes:          data.DefaultDashboardRoutes,
		ClassCodeRoutes: data.DefaultClassCodeRoutes,
		PageTitle:       "class codes",
		ExtraData: map[string]any{
			"NoClassCode": noClassCode,
			"Action":      actionsURLParameters,
			"ClassCodes":  classCodesDB,
		},
	}

	RenderTableClassCodePage(w, dataPage)
}

func AddFormClassCodeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddFormClassCodeHandler -> tools.CheckRequest return not ok")
		return
	}

	dataPage := data.ClassCodePageData{
		Routes:          data.DefaultDashboardRoutes,
		ClassCodeRoutes: data.DefaultClassCodeRoutes,
		PageTitle:       "add class code",
	}
	RenderAddFormClassCodePage(w, dataPage)
}

func AddClassCodeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From AddClassCodeHandler -> tools.CheckRequest return not ok")
		return
	}

	name := strings.TrimSpace(r.FormValue("class_code"))

	err := queries.CreateClassCode(r.Context(), db.CreateClassCodeParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From AddClassCodeHandler -> CreateClassCode : DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois la même classe ou la classe ne peut être sans nom.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, data.DefaultStudentRoutes.ClassCodesURL, http.StatusSeeOther)
}

func EditFormClassCodeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From EditFormClassCodeHandler -> tools.CheckRequest return not ok")
		return
	}

	classCodeIDStr := r.URL.Query().Get("class_code_id")
	if classCodeIDStr == "" {
		log.Println("From EditFormClassCodeHandler : no class code id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditFormClassCodeHandler -> strconv.ParseInt, invalid class code ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCode, err := queries.GetClassCodeNameByID(r.Context(), db.GetClassCodeNameByIDParams{
		ID:     classCodeID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "EditFormClassCodeHandler GetClassCodeNameByID")
		return
	}

	dataPage := data.ClassCodePageData{
		Routes:          data.DefaultDashboardRoutes,
		ClassCodeRoutes: data.DefaultClassCodeRoutes,
		PageTitle:       "edit class code",
		ExtraData: map[string]any{
			"ClassCode":   classCode,
			"ClassCodeID": classCodeIDStr,
		},
	}
	RenderEditFormClassCodePage(w, dataPage)
}

func EditClassCodeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From EditClassCodeHandler -> tools.CheckRequest return not ok")
		return
	}

	newClassCode := strings.TrimSpace(r.FormValue("new_class_code"))

	classCodeIDStr := r.FormValue("class_code_id")
	if classCodeIDStr == "" {
		log.Println("From EditClassCodeHandler : no class code ID")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}
	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From EditClassCodeHandler -> strconv.ParseInt, invalid skillID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.UpdateClassCode(r.Context(), db.UpdateClassCodeParams{
		Name:   newClassCode,
		ID:     classCodeID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From EditClassCodeHandler : UpdateClassCode DB error: %v", err)
		errorMessage := url.QueryEscape("Il ne peut pas exister deux fois le même champ ou le champ ne peut pas être vide.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "UpdateClassCode") {
		return
	}

	http.Redirect(w, r, data.DefaultStudentRoutes.ClassCodesURL, http.StatusSeeOther)
}

func DeleteFormClassCodeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From DeleteFormClassCodeHandler -> tools.CheckRequest return not ok")
		return
	}

	classCodeIDStr := r.URL.Query().Get("class_code_id")
	if classCodeIDStr == "" {
		log.Println("From DeleteFormClassCodeHandler : no class code id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteFormClassCodeHandler -> strconv.ParseInt, invalid class code ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCode, err := queries.GetClassCodeNameByID(r.Context(), db.GetClassCodeNameByIDParams{
		ID:     classCodeID,
		UserID: userID,
	})
	if err != nil {
		tools.HandleOwnedLookupError(w, err, "DeleteFormClassCodeHandler GetClassCodeNameByID")
		return
	}

	dataPage := data.ClassCodePageData{
		Routes:          data.DefaultDashboardRoutes,
		ClassCodeRoutes: data.DefaultClassCodeRoutes,
		PageTitle:       "delete class code",
		ExtraData: map[string]any{
			"ClassCode":   classCode,
			"ClassCodeID": classCodeIDStr,
		},
	}

	RenderDeleteFormClassCodePage(w, dataPage)
}

func DeleteClassCodeHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From DeleteClassCodeHandler -> tools.CheckRequest return not ok")
		return
	}

	classCodeIDStr := r.FormValue("class_code_id")
	if classCodeIDStr == "" {
		log.Println("From DeleteClassCodeHandler : no class code id parameter")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	classCodeID, err := strconv.ParseInt(classCodeIDStr, 10, 64)
	if err != nil {
		log.Printf("From DeleteClassCodeHandler -> strconv.ParseInt, invalid class code ID, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	rows, err := queries.DeleteClassCode(r.Context(), db.DeleteClassCodeParams{
		ID:     classCodeID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From DeleteClassCodeHandler : DeleteClassCode DB error: %v", err)
		if tools.IsSQLiteForeignKeyConstraint(err) {
			errorMessage := url.QueryEscape("Cette classe est utilisée par une évaluation ou contient encore des élèves et ne peut pas être supprimée.")
			http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
			return
		}
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	if !tools.HandleOwnedMutationRows(w, rows, "DeleteClassCode") {
		return
	}

	http.Redirect(w, r, data.DefaultStudentRoutes.ClassCodesURL, http.StatusSeeOther)
}
