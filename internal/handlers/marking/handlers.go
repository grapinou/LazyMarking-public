package marking

import (
	"fmt"
	"log"
	"net/http"

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
	userID, username, ok := tools.CheckRequest(w, r, http.MethodPost)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	file, err := tools.CheckPdfFile(r, 100<<20)
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}

	// Lance la goroutine principale
	go tools.ProcessMarking(userID, username, file, queries)

	// Répond immédiatement
	w.Write([]byte(fmt.Sprintf("Job %s started", "zest parti !")))
}

/*
func ProcessingMarkingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodPost)
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

	// commencer ici la go routine ?

	ctx := context.Background()

	var qrDatas []config.QrCodeInfo
	var qrNotDetected []string
	for _, page := range pages {
		pdf := filepath.Base(page)
		name := strings.TrimSuffix(pdf, filepath.Ext(page)) + ".png"
		png := tools.ConvertPdfToPng(tempDir, pdf, "")
		pngPath := filepath.Join(tempDir, png)
		data, err := tools.QrReader(pngPath)
		if err != nil {
			log.Printf("file : %s, qr not detected : error : %v \n", pngPath, err)
			qrNotDetected = append(qrNotDetected, png)
		} else {
			var info config.QrCodeInfo
			if err := json.Unmarshal([]byte(data), &info); err != nil {
				log.Printf("From ProcessingMarkingHandler -> Unmarshal return error : %v", err)
				http.Error(w, "Something went wrong !", http.StatusInternalServerError)
				return
			}
			info.PageName = name

			qrDatas = append(qrDatas, info)
		}
	}

	if err := tools.RemoveFiles(pages); err != nil {
		log.Printf("From ProcessingMarkingHandler -> RemoveFiles return error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	exams := tools.GroupQrCodes(qrDatas)

	var markExams []config.MarkExam
	var notMarkedExams []config.MarkExam

	for _, exam := range exams {
		markExam, err := tools.MarkingStudentExam(userID, username, tempDir, exam, ctx, queries)
		if err != nil {
			log.Printf("Error with MarkingStudentExam, %v", err)
			notMarkedExams = append(notMarkedExams, markExam)
		}
		if markExam.Status {
			markExams = append(markExams, markExam)
		}
	}

	fmt.Println("Done !")
}
*/
