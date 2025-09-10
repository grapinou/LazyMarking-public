package marking

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func AddPdfFormMarkingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From AddPdfFormMarkingHandler -> tools.CheckRequest return not ok")
		return
	}

	examsGeneratedSuccess, err := queries.GetExamsGeneratedSuccess(r.Context(), userID)
	if err != nil {
		log.Printf("From AddPdfFormMarkingHandler -> queries.GetExamsGeneratedSuccess error DB : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	noExamGenerated := true
	if len(examsGeneratedSuccess) > 0 {
		noExamGenerated = false
	}

	dataPage := data.MarkingPageData{
		Routes:        data.DefaultDashboardRoutes,
		MarkingRoutes: data.DefaultMarkingRoutes,
		PageTitle:     "Processing Marking",
		ExtraData: map[string]any{
			"NoExamGenerated": noExamGenerated,
			"Exams":           examsGeneratedSuccess,
		},
	}

	RenderAddPdfFormMarkingPage(w, dataPage)
}

func ProcessingMarkingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, username, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		log.Println("From ProcessingMarkingHandler-> tools.CheckRequest return not ok")
		return
	}

	file, err := tools.CheckPdfFile(r, 100<<20) // 100 Mo ou (50<<20) pour 50 Mo
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> CheckPdfFile : error : %v", err)
		errorMessage := url.QueryEscape("Fichier probablement trop volumineux ou le fichier n'est pas un pdf.")
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	tempDir, ok := tools.CreateUserTempDir(username)
	if !ok {
		log.Println("From ProcessingMarkingHandler -> CreateUserTempDir return not ok")
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	if err := tools.ClearDir(tempDir); err != nil {
		log.Printf("From ProcessingMarkingHandler -> ClearDir return error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	if err := tools.SplitPdf(file, tempDir, "page-%d.pdf"); err != nil {
		log.Printf("From ProcessingMarkingHandler -> SplitPdf return error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	pages, err := tools.GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		log.Printf("From ProcessingMarkingHandler -> GetAllFiles return error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	var qrDetected []string
	var qrNotDetected []string
	for _, page := range pages {
		pdf := filepath.Base(page)
		name := strings.TrimSuffix(pdf, filepath.Ext(page))
		png := tools.ConvertPdfToPng(tempDir, pdf, name)
		pngPath := filepath.Join(tempDir, png)
		data, err := tools.QrReader(pngPath)
		if err != nil {
			log.Printf("file : %s, qr not detected : error : %v \n", pngPath, err)
			qrNotDetected = append(qrNotDetected, png)
		} else {
			qrDetected = append(qrDetected, data)
		}
	}

	if err := tools.RemoveFiles(pages); err != nil {
		log.Printf("From ProcessingMarkingHandler -> RemoveFiles return error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	var qrDatas []config.QrCodeInfo
	for _, qr := range qrDetected {
		var info config.QrCodeInfo
		if err := json.Unmarshal([]byte(qr), &info); err != nil {
			log.Printf("From ProcessingMarkingHandler -> Unmarshal return error : %v", err)
			http.Error(w, "Something went wrong !", http.StatusInternalServerError)
			return
		}
		qrDatas = append(qrDatas, info)
	}

	fmt.Println(qrDatas)
	fmt.Println(qrNotDetected)
}
