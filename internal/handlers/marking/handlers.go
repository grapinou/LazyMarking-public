package marking

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

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

func ProcessingMarkingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries, appCtx context.Context) {
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

	stagedFile, err := os.CreateTemp("", "lazymarking-upload-*.pdf")
	if err != nil {
		http.Error(w, "Unable to stage upload", http.StatusInternalServerError)
		return
	}
	if _, err = io.Copy(stagedFile, file); err != nil {
		file.Close()
		stagedFile.Close()
		os.Remove(stagedFile.Name())
		http.Error(w, "Unable to stage upload", http.StatusInternalServerError)
		return
	}
	if err = file.Close(); err != nil {
		stagedFile.Close()
		os.Remove(stagedFile.Name())
		http.Error(w, "Unable to close upload", http.StatusInternalServerError)
		return
	}
	if _, err = stagedFile.Seek(0, io.SeekStart); err != nil {
		stagedFile.Close()
		os.Remove(stagedFile.Name())
		http.Error(w, "Unable to stage upload", http.StatusInternalServerError)
		return
	}

	jobDBID, err := queries.CreateMarkingJob(r.Context(), userID)
	if err != nil {
		stagedFile.Close()
		os.Remove(stagedFile.Name())
		log.Printf("From ProcessingMarkingHandler -> queries.CreateMarkingJob DB error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}

	// Lance la goroutine principale
	go func() {
		defer stagedFile.Close()
		defer os.Remove(stagedFile.Name())
		tools.ProcessMarking(appCtx, userID, username, jobDBID, stagedFile, queries)
	}()

	params := "?job_id=" + url.QueryEscape(strconv.FormatInt(jobDBID, 10))
	propressMarkingURL := data.DefaultMarkingRoutes.ProgressMarking + params
	http.Redirect(w, r, propressMarkingURL, http.StatusSeeOther)
}

func ProgressMarkingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, _, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jobIDStr := r.URL.Query().Get("job_id")
	if jobIDStr == "" {
		log.Println("From ProgressMarkingHandler -> no job id")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		log.Printf("From ProgressMarkingHandler -> strconv.ParseInt invalid jobIDStr, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	markingStatus, err := queries.GetMarkingStatus(r.Context(), db.GetMarkingStatusParams{
		ID:     jobID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From ProgressMarkingHandler -> GetMarkingStatus : DB error : %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	if markingStatus.Status == "failed" {
		log.Println("From ProgressMarkingHandler -> marking status failed")
		errorMessage := url.QueryEscape("Erreur lors de la correction de l'examen. Vérifier que le fichier soit le bon. Si le problème persiste, contacter l'admin et corriger à la mano en attendant.")
		if err := queries.DeleteMarkingJob(r.Context(), db.DeleteMarkingJobParams{
			ID:     jobID,
			UserID: userID,
		}); err != nil {
			log.Printf("From GetExamProgressHandler -> DeleteExamGenerated : DB error : %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, data.ErrorMessageURL+"?errormessage="+errorMessage, http.StatusSeeOther)
		return
	}

	if markingStatus.Status == "success" && markingStatus.StatusPdf == "success" {
		params := "?job_id=" + url.QueryEscape(strconv.FormatInt(jobID, 10))
		successURL := data.DefaultMarkingRoutes.SuccessURL + params
		http.Redirect(w, r, successURL, http.StatusSeeOther)
		return
	}

	progress, err := queries.GetMarkingProgress(r.Context(), db.GetMarkingProgressParams{
		ID:     jobID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From ProgressMarkingHandler -> GetMarkingProgress : DB error : %v", err)
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	dataPage := data.MarkingPageData{
		Routes:        data.DefaultDashboardRoutes,
		MarkingRoutes: data.DefaultMarkingRoutes,
		PageTitle:     "Processing Marking",
		ExtraData: map[string]any{
			"JobID":         jobIDStr,
			"ProcessedPage": progress.DonePages.Int64,
			"TotalPages":    progress.TotalPages.Int64,
			"ProcessedExam": progress.DoneExams.Int64,
			"TotalExams":    progress.TotalExams.Int64,
			"Status":        markingStatus.Status,
			"StatusPdf":     markingStatus.StatusPdf,
		},
	}

	RenderProgressMarkingPage(w, dataPage)
}

func SuccessMarkingProcessingHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	userID, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From SuccessMarkingProcessingHandler-> tools.CheckRequest return not ok")
		return
	}
	jobIDStr := r.URL.Query().Get("job_id")
	if jobIDStr == "" {
		log.Println("From SuccessMarkingProcessingHandler -> no job id")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		log.Printf("From SuccessMarkingProcessingHandler -> strconv.ParseInt invalid jobIDStr, error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	name, err := queries.GetExamAndMarkName(r.Context(), db.GetExamAndMarkNameParams{
		ID:     jobID,
		UserID: userID,
	})
	if err != nil {
		log.Printf("From SuccessMarkingProcessingHandler -> GetExamAndMarkName DB error : %v", err)
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	examName := filepath.Base(name.ExamName.String)
	markTableName := filepath.Base(name.MarkTableName.String)

	operation := "marking-" + strconv.FormatInt(jobID, 10)
	pdfExamURL := data.DefaultMarkingRoutes.ServePDF + "?operation=" + url.QueryEscape(operation) + "&file=" + url.QueryEscape(examName)
	pdfMarkTalbeURL := data.DefaultMarkingRoutes.ServePDF + "?operation=" + url.QueryEscape(operation) + "&file=" + url.QueryEscape(markTableName)

	// on regarde si des pages n'ont pas été traitées.
	tempDir, ok := tools.CreateOperationTempDir(username, operation)
	if !ok {
		http.Error(w, "Unable to access marking workspace", http.StatusInternalServerError)
		return
	}
	leftPages, err := tools.LeftPages(tempDir, name)
	if err != nil {
		log.Printf("From SuccessMarkingProcessingHandler -> LeftPages: %v", err)
		http.Error(w, "Something went wrong !", http.StatusInternalServerError)
		return
	}
	var pdfLeftPagesUrl string
	if leftPages != "" {
		pdfLeftPagesUrl = data.DefaultMarkingRoutes.ServePDF + "?operation=" + url.QueryEscape(operation) + "&file=" + url.QueryEscape(leftPages)
	}

	dataPage := data.MarkingPageData{
		Routes:        data.DefaultDashboardRoutes,
		MarkingRoutes: data.DefaultMarkingRoutes,
		PageTitle:     "Success Processing Marking",
		ExtraData: map[string]any{
			"Status":          "Success",
			"PdfExamURL":      pdfExamURL,
			"PdfMarkTable":    pdfMarkTalbeURL,
			"PdfLeftPagesUrl": pdfLeftPagesUrl,
		},
	}

	RenderSuccessProgressMarkingPage(w, dataPage)
}

func ServeFullMarkingPdfHandler(w http.ResponseWriter, r *http.Request, queries *db.Queries) {
	_, username, ok := tools.CheckRequest(w, r, http.MethodGet)
	if !ok {
		log.Println("From ServeFullMarkingPdfHandler -> tools.CheckRequest return not ok")
		return
	}

	if username == "" {
		log.Println("From ServeFullMarkingPdfHandler , no username")
		http.Error(w, "Something went wrong !", http.StatusBadRequest)
		return
	}

	filename := r.URL.Query().Get("file")
	operation := r.URL.Query().Get("operation")
	if filename == "" || operation == "" {
		http.Error(w, "Missing file parameter", http.StatusBadRequest)
		return
	}

	tools.ServePdfNamed(username, operation, filename, w, r)
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
