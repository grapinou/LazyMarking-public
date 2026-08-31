package tools

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func TestRealStudentExamMarkingSmoke(t *testing.T) {
	testCases := []struct {
		name         string
		pdfEnv       string
		studentIDEnv string
		pageCount    int
	}{
		{name: "one_page", pdfEnv: "LAZYMARKING_TEST_PDF_1_PAGE", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_1_PAGE", pageCount: 1},
		{name: "two_pages", pdfEnv: "LAZYMARKING_TEST_PDF_2_PAGES", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_2_PAGES", pageCount: 2},
		{name: "three_pages", pdfEnv: "LAZYMARKING_TEST_PDF", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_3_PAGES", pageCount: 3},
	}
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	userIDValue := os.Getenv("LAZYMARKING_TEST_USER_ID")
	for _, tc := range testCases {
		if dbPath == "" || userIDValue == "" || os.Getenv(tc.pdfEnv) == "" || os.Getenv(tc.studentIDEnv) == "" {
			t.Skip("private one-, two-, and three-page PDFs, IDs, and historical DB must all be configured; skipping real-marking smoke test")
		}
	}
	userID := parsePositiveFixtureID(t, "LAZYMARKING_TEST_USER_ID")

	queries, closeDB := openCopiedHistoricalDB(t, dbPath)
	t.Cleanup(closeDB)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			studentExamID := parsePositiveFixtureID(t, tc.studentIDEnv)
			corpus := readAndGroupScannedExams(t, os.Getenv(tc.pdfEnv), newMarkingIntegrationTempDir(t))
			targetExam := findExamByID(t, corpus.Exams, studentExamID)
			if got := len(targetExam.Pages); got != tc.pageCount {
				t.Fatalf("student_exam_id %d page count = %d, want %d", studentExamID, got, tc.pageCount)
			}

			markExam, err := markStudentExamWithoutPanic(userID, corpus.TempDir, targetExam, queries)
			if err != nil {
				t.Fatalf("marking stage for student_exam_id %d: %v", studentExamID, err)
			}
			validateSuccessfulMark(t, markExam)
			if got := markExam.Pages; got != tc.pageCount {
				t.Errorf("MarkExam.Pages = %d, want %d", got, tc.pageCount)
			}
		})
	}
}

func TestRealCompleteMarkingBatches(t *testing.T) {
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	batchValue := os.Getenv("LAZYMARKING_TEST_BATCH_PDFS")
	if dbPath == "" || batchValue == "" || os.Getenv("LAZYMARKING_TEST_USER_ID") == "" {
		t.Skip("LAZYMARKING_TEST_DB, LAZYMARKING_TEST_USER_ID, and LAZYMARKING_TEST_BATCH_PDFS are not configured; skipping complete real-marking batches")
	}
	userID := parsePositiveFixtureID(t, "LAZYMARKING_TEST_USER_ID")

	batchPaths := splitConfiguredBatchPaths(batchValue)
	if len(batchPaths) == 0 {
		t.Fatal("LAZYMARKING_TEST_BATCH_PDFS contains no paths")
	}
	queries, closeDB := openCopiedHistoricalDB(t, dbPath)
	t.Cleanup(closeDB)

	for i, batchPath := range batchPaths {
		batchName := fmt.Sprintf("batch_%03d", i+1)
		t.Run(batchName, func(t *testing.T) {
			corpus, err := readAndGroupCompleteBatch(batchPath, newMarkingIntegrationTempDir(t))
			if err != nil {
				t.Fatalf("%s: %v", batchName, err)
			}
			if len(corpus.Exams) == 0 {
				t.Fatalf("%s QR grouping stage: no exams found", batchName)
			}

			seen := make(map[int64]struct{}, len(corpus.Exams))
			for _, exam := range corpus.Exams {
				if exam.StudentExamID <= 0 {
					t.Errorf("%s QR grouping stage: invalid student_exam_id %d", batchName, exam.StudentExamID)
					continue
				}
				if _, exists := seen[exam.StudentExamID]; exists {
					t.Errorf("%s QR grouping stage: duplicate student_exam_id %d", batchName, exam.StudentExamID)
					continue
				}
				seen[exam.StudentExamID] = struct{}{}
				if len(exam.Pages) == 0 {
					t.Errorf("%s QR grouping stage: student_exam_id %d has no pages", batchName, exam.StudentExamID)
					continue
				}

				markExam, err := markStudentExamWithoutPanic(userID, corpus.TempDir, exam, queries)
				if err != nil {
					t.Errorf("%s marking stage: student_exam_id %d: %v", batchName, exam.StudentExamID, err)
					continue
				}
				validateSuccessfulMark(t, markExam)
			}
		})
	}
}

// TestRealHistoricalReferencePathEquivalence validates the path change, not a
// legacy PDF fallback. It reconstructs the legacy Typst raster, stores those
// exact bytes as 0039 references in a temporary DB/workspace, then compares
// both marking paths on independently extracted copies of the same private
// scans. It is skipped unless the existing private fixture variables are set.
func TestRealHistoricalReferencePathEquivalence(t *testing.T) {
	testCases := []struct {
		name, pdfEnv, studentIDEnv string
	}{
		{name: "one_page", pdfEnv: "LAZYMARKING_TEST_PDF_1_PAGE", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_1_PAGE"},
		{name: "two_pages", pdfEnv: "LAZYMARKING_TEST_PDF_2_PAGES", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_2_PAGES"},
		{name: "three_pages", pdfEnv: "LAZYMARKING_TEST_PDF", studentIDEnv: "LAZYMARKING_TEST_STUDENT_EXAM_ID_3_PAGES"},
	}
	dbPath := os.Getenv("LAZYMARKING_TEST_DB")
	if dbPath == "" || os.Getenv("LAZYMARKING_TEST_USER_ID") == "" {
		t.Skip("private historical DB and user ID are not configured; skipping historical-reference path equivalence")
	}
	for _, tc := range testCases {
		if os.Getenv(tc.pdfEnv) == "" || os.Getenv(tc.studentIDEnv) == "" {
			t.Skip("private one-, two-, and three-page PDFs and technical IDs must all be configured; skipping historical-reference path equivalence")
		}
	}
	userID := parsePositiveFixtureID(t, "LAZYMARKING_TEST_USER_ID")
	conn, queries, closeDB := openCopiedHistoricalDBWithConnection(t, dbPath)
	t.Cleanup(closeDB)
	isolatedUsername := fmt.Sprintf("reference-opt-in-%d", time.Now().UnixNano())
	if _, err := conn.Exec(`UPDATE users SET username = ? WHERE id = ?`, isolatedUsername, userID); err != nil {
		t.Fatalf("isolate temporary user workspace: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			studentExamID := parsePositiveFixtureID(t, tc.studentIDEnv)
			baselineCorpus := readAndGroupScannedExams(t, os.Getenv(tc.pdfEnv), newMarkingIntegrationTempDir(t))
			baselineExam := findExamByIDWithTechnicalSummary(t, baselineCorpus.Exams, studentExamID)
			baseline, err := MarkingStudentExam(userID, isolatedUsername, baselineCorpus.TempDir, baselineExam, context.Background(), queries)
			if err != nil {
				t.Fatalf("legacy marking path for student_exam_id %d: %v", studentExamID, err)
			}
			validateSuccessfulMark(t, baseline)

			installReconstructedReferencesForOptInTest(t, queries, userID, isolatedUsername, studentExamID)
			candidateCorpus := readAndGroupScannedExams(t, os.Getenv(tc.pdfEnv), newMarkingIntegrationTempDir(t))
			candidateExam := findExamByID(t, candidateCorpus.Exams, studentExamID)
			candidate, err := MarkingStudentExam(userID, isolatedUsername, candidateCorpus.TempDir, candidateExam, context.Background(), queries)
			if err != nil {
				t.Fatalf("resolver marking path for student_exam_id %d: %v", studentExamID, err)
			}
			validateSuccessfulMark(t, candidate)

			if baseline.Score != candidate.Score || baseline.Total != candidate.Total {
				t.Fatalf("student_exam_id %d score differs: legacy=%v/%d resolver=%v/%d", studentExamID, baseline.Score, baseline.Total, candidate.Score, candidate.Total)
			}
			if baseline.DetailedResult == nil || candidate.DetailedResult == nil {
				t.Fatalf("student_exam_id %d lacks detailed result", studentExamID)
			}
			if !reflect.DeepEqual(baseline.DetailedResult.Questions, candidate.DetailedResult.Questions) {
				t.Fatalf("student_exam_id %d MeanGray, detected_state, or QuestionMark differs", studentExamID)
			}
			t.Logf("student_exam_id %d: %d page(s), %d question result(s), identical", studentExamID, candidate.Pages, len(candidate.DetailedResult.Questions))
		})
	}
}

func findExamByIDWithTechnicalSummary(t *testing.T, exams []config.Exam, studentExamID int64) config.Exam {
	t.Helper()
	for _, exam := range exams {
		if exam.StudentExamID == studentExamID {
			return exam
		}
	}
	type technicalExam struct {
		ID    int64
		Pages []int
	}
	summary := make([]technicalExam, 0, len(exams))
	for _, exam := range exams {
		pages := make([]int, 0, len(exam.Pages))
		for _, page := range exam.Pages {
			pages = append(pages, page.Number)
		}
		sort.Ints(pages)
		summary = append(summary, technicalExam{ID: exam.StudentExamID, Pages: pages})
	}
	sort.Slice(summary, func(i, j int) bool { return summary[i].ID < summary[j].ID })
	t.Fatalf("student_exam_id %d was not found; recognized technical groups: %+v", studentExamID, summary)
	return config.Exam{}
}

func installReconstructedReferencesForOptInTest(t *testing.T, queries *db.Queries, userID int64, username string, studentExamID int64) {
	t.Helper()
	ctx := context.Background()
	content, err := queries.GetStudentContentExam(ctx, db.GetStudentContentExamParams{StudentExamID: studentExamID, UserID: userID})
	if err != nil {
		t.Fatal(err)
	}
	var qcm config.QCM
	if err := json.Unmarshal([]byte(content.Content), &qcm); err != nil {
		t.Fatal(err)
	}
	identity, err := queries.GetStudentExamPageReference(ctx, db.GetStudentExamPageReferenceParams{StudentExamID: studentExamID, Page: 1, UserID: userID})
	if err != nil {
		t.Fatal(err)
	}
	workspace, ok := CreateOperationTempDir(username, "exam-"+strconv.FormatInt(identity.GenerationID, 10))
	if !ok {
		t.Fatal("create isolated generation workspace")
	}
	t.Cleanup(func() { _ = RemoveOperationTempDir(username, "exam-"+strconv.FormatInt(identity.GenerationID, 10)) })
	typstPath, ok := TypstWriter(workspace, username, qcm, config.ExamQCM)
	if !ok {
		t.Fatal("reconstruct legacy reference for resolver comparison")
	}
	pages, ok := ExportTypstToPNGs(typstPath)
	if !ok || len(pages) != int(content.PageTot) {
		t.Fatalf("reconstructed reference count=%d, want %d", len(pages), content.PageTot)
	}
	for index, pagePath := range pages {
		if _, err := StoreStudentExamPageReference(ctx, queries, userID, username, identity.GenerationID, studentExamID, int64(index+1), pagePath); err != nil {
			t.Fatalf("store reconstructed reference page %d: %v", index+1, err)
		}
	}
	// The source Typst and pre-QR files are deliberately removable: only the
	// durable reference hierarchy must remain for the candidate marking pass.
	if err := os.Remove(typstPath); err != nil {
		t.Fatal(err)
	}
	for _, pagePath := range pages {
		if err := os.Remove(pagePath); err != nil {
			t.Fatal(err)
		}
	}
}

func newMarkingIntegrationTempDir(t *testing.T) string {
	t.Helper()
	tempDir, err := os.MkdirTemp(".", ".real-marking-integration-*")
	if err != nil {
		t.Fatalf("create integration-test directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("remove integration-test directory: %v", err)
		}
	})

	tempDir, err = filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("resolve integration-test directory: %v", err)
	}
	return tempDir
}

func findExamByID(t *testing.T, exams []config.Exam, studentExamID int64) config.Exam {
	t.Helper()
	for _, exam := range exams {
		if exam.StudentExamID == studentExamID {
			return exam
		}
	}
	t.Fatalf("student_exam_id %d was not found in scanned exams", studentExamID)
	return config.Exam{}
}

func parsePositiveFixtureID(t *testing.T, envName string) int64 {
	t.Helper()
	id, err := strconv.ParseInt(os.Getenv(envName), 10, 64)
	if err != nil || id <= 0 {
		t.Fatalf("%s must contain a positive student_exam_id", envName)
	}
	return id
}

func splitConfiguredBatchPaths(value string) []string {
	parts := filepath.SplitList(value)
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		if path := strings.TrimSpace(part); path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func openCopiedHistoricalDB(t *testing.T, sourcePath string) (*db.Queries, func()) {
	_, queries, closeDB := openCopiedHistoricalDBWithConnection(t, sourcePath)
	return queries, closeDB
}

func openCopiedHistoricalDBWithConnection(t *testing.T, sourcePath string) (*sql.DB, *db.Queries, func()) {
	t.Helper()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	t.Chdir(filepath.Clean(filepath.Join(workingDir, "../../..")))

	absoluteSourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		t.Fatal("historical DB copy stage: resolve source fixture path")
	}
	sourceDigest, err := fileSHA256ForPrivateFixture(absoluteSourcePath)
	if err != nil {
		t.Fatal("historical DB copy stage: hash source fixture")
	}
	source, err := os.Open(absoluteSourcePath)
	if err != nil {
		t.Fatal("historical DB copy stage: source fixture is not readable")
	}
	defer source.Close()

	copyDir := newMarkingIntegrationTempDir(t)
	copyPath := filepath.Join(copyDir, "historical-copy.db")
	destination, err := os.OpenFile(copyPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("historical DB copy stage: create temporary copy: %v", err)
	}
	if _, err := io.Copy(destination, source); err != nil {
		destination.Close()
		t.Fatalf("historical DB copy stage: copy fixture: %v", err)
	}
	if err := destination.Close(); err != nil {
		t.Fatalf("historical DB copy stage: close temporary copy: %v", err)
	}
	copyPath, err = filepath.Abs(copyPath)
	if err != nil {
		t.Fatalf("historical DB copy stage: resolve temporary copy path: %v", err)
	}
	conn, err := db.InitDB(copyPath)
	if err != nil {
		t.Fatalf("historical DB copy stage: initialize temporary copy: %v", err)
	}
	applyHistoricalCopyMigrations(t, conn, filepath.Join("db", "migrations"), 39)
	assertHistoricalCopyReferenceSchema(t, conn)
	return conn, db.New(conn), func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close temporary historical DB copy: %v", err)
		}
		currentSourceDigest, err := fileSHA256ForPrivateFixture(absoluteSourcePath)
		if err != nil {
			t.Errorf("verify original historical DB remains readable")
		} else if currentSourceDigest != sourceDigest {
			t.Errorf("original historical DB changed during opt-in test")
		}
	}
}

func applyHistoricalCopyMigrations(t *testing.T, conn *sql.DB, migrationsDir string, targetVersion int64) {
	t.Helper()
	var currentVersion int64
	if err := conn.QueryRow(`SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied = 1`).Scan(&currentVersion); err != nil {
		t.Fatalf("historical DB migration stage: read source migration version from temporary copy: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		t.Fatalf("historical DB migration stage: list migrations: %v", err)
	}
	sort.Strings(files)
	for _, migrationPath := range files {
		base := filepath.Base(migrationPath)
		if len(base) < 4 {
			continue
		}
		version, parseErr := strconv.ParseInt(base[:4], 10, 64)
		if parseErr != nil || version <= currentVersion || version > targetVersion {
			continue
		}
		contents, readErr := os.ReadFile(migrationPath)
		if readErr != nil {
			t.Fatalf("historical DB migration stage: read migration %04d: %v", version, readErr)
		}
		parts := strings.SplitN(string(contents), "-- +goose Down", 2)
		if len(parts) != 2 {
			t.Fatalf("historical DB migration stage: migration %04d has no Down boundary", version)
		}
		up := strings.Replace(parts[0], "-- +goose NO TRANSACTION", "", 1)
		up = strings.Replace(up, "-- +goose Up", "", 1)
		if _, execErr := conn.Exec(up); execErr != nil {
			t.Fatalf("historical DB migration stage: apply migration %04d to temporary copy: %v", version, execErr)
		}
		if _, recordErr := conn.Exec(`INSERT INTO goose_db_version(version_id, is_applied) VALUES(?, 1)`, version); recordErr != nil {
			t.Fatalf("historical DB migration stage: record migration %04d on temporary copy: %v", version, recordErr)
		}
		currentVersion = version
	}
	if currentVersion != targetVersion {
		t.Fatalf("historical DB migration stage: temporary copy version=%d, want %d", currentVersion, targetVersion)
	}
}

func assertHistoricalCopyReferenceSchema(t *testing.T, conn *sql.DB) {
	t.Helper()
	var migrationVersion int64
	if err := conn.QueryRow(`SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied = 1`).Scan(&migrationVersion); err != nil {
		t.Fatalf("historical DB migration stage: read temporary copy migration version: %v", err)
	}
	if migrationVersion < 39 {
		t.Fatalf("historical DB migration stage: temporary copy version=%d, want at least 39", migrationVersion)
	}
	rows, err := conn.Query(`PRAGMA table_info(student_exam_page_content)`)
	if err != nil {
		t.Fatalf("historical DB migration stage: inspect temporary copy schema: %v", err)
	}
	defer rows.Close()
	want := map[string]bool{
		"reference_storage_key": false,
		"reference_width":       false,
		"reference_height":      false,
		"reference_dpi":         false,
		"reference_sha256":      false,
	}
	for rows.Next() {
		var cid, notNull, primaryKey int64
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatalf("historical DB migration stage: scan temporary copy schema: %v", err)
		}
		if _, exists := want[name]; exists {
			want[name] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("historical DB migration stage: inspect temporary copy schema: %v", err)
	}
	for column, found := range want {
		if !found {
			t.Fatalf("historical DB migration stage: temporary copy lacks column %s", column)
		}
	}
	for column := range want {
		var nonNullCount int64
		query := fmt.Sprintf(`SELECT COUNT(*) FROM student_exam_page_content WHERE %s IS NOT NULL`, column)
		if err := conn.QueryRow(query).Scan(&nonNullCount); err != nil {
			t.Fatalf("historical DB migration stage: inspect legacy values for %s: %v", column, err)
		}
		if nonNullCount != 0 {
			t.Fatalf("historical DB migration stage: legacy column %s has %d unexpected values", column, nonNullCount)
		}
	}
}

func fileSHA256ForPrivateFixture(path string) ([sha256.Size]byte, error) {
	var empty [sha256.Size]byte
	file, err := os.Open(path)
	if err != nil {
		return empty, err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return empty, err
	}
	var digest [sha256.Size]byte
	copy(digest[:], hasher.Sum(nil))
	return digest, nil
}

func readAndGroupCompleteBatch(pdfPath, tempDir string) (scannedExamCorpus, error) {
	pdf, err := os.Open(pdfPath)
	if err != nil {
		return scannedExamCorpus{}, fmt.Errorf("PDF read stage: configured batch is not readable")
	}
	defer pdf.Close()

	if err := SplitPdf(pdf, tempDir, "page-%d.pdf"); err != nil {
		return scannedExamCorpus{}, fmt.Errorf("PDF read stage: split configured batch: %w", err)
	}
	pages, err := GetAllFiles(tempDir, "*.pdf")
	if err != nil {
		return scannedExamCorpus{}, fmt.Errorf("PDF read stage: list split pages: %w", err)
	}
	if len(pages) == 0 {
		return scannedExamCorpus{}, fmt.Errorf("PDF read stage: split produced no pages")
	}

	qrCodes := make([]config.QrCodeInfo, 0, len(pages))
	for pageIndex, pagePath := range pages {
		pageName := filepath.Base(pagePath)
		pngName, err := ConvertPdfToPng(tempDir, pageName, "")
		if err != nil {
			return scannedExamCorpus{}, fmt.Errorf("QR grouping stage: convert page %d: %w", pageIndex+1, err)
		}
		qrData, err := QrReader(filepath.Join(tempDir, pngName))
		if err != nil {
			return scannedExamCorpus{}, fmt.Errorf("QR grouping stage: read page %d QR code: %w", pageIndex+1, err)
		}
		var info config.QrCodeInfo
		if err := json.Unmarshal([]byte(qrData), &info); err != nil {
			return scannedExamCorpus{}, fmt.Errorf("QR grouping stage: decode page %d QR data: %w", pageIndex+1, err)
		}
		info.PageName = pngName
		qrCodes = append(qrCodes, info)
	}

	return scannedExamCorpus{
		Exams:     GroupQrCodes(qrCodes),
		TempDir:   tempDir,
		PageCount: len(pages),
		QRCount:   len(qrCodes),
	}, nil
}

func markStudentExamWithoutPanic(userID int64, tempDir string, exam config.Exam, queries *db.Queries) (markExam config.MarkExam, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return MarkingStudentExam(userID, "integration-test", tempDir, exam, context.Background(), queries)
}

func validateSuccessfulMark(t *testing.T, markExam config.MarkExam) {
	t.Helper()
	if !markExam.Status {
		t.Errorf("marking stage: student_exam_id %d: Status = false, want true", markExam.StudentExamID)
	}
	if markExam.Pages <= 0 {
		t.Errorf("marking stage: student_exam_id %d: Pages = %d, want > 0", markExam.StudentExamID, markExam.Pages)
	}
	if markExam.Total <= 0 {
		t.Errorf("marking stage: student_exam_id %d: Total = %d, want > 0", markExam.StudentExamID, markExam.Total)
	}
	if markExam.Score < 0 {
		t.Errorf("marking stage: student_exam_id %d: Score = %v, want >= 0", markExam.StudentExamID, markExam.Score)
	}
	if markExam.Score > float64(markExam.Total) {
		t.Errorf("marking stage: student_exam_id %d: Score = %v, want <= Total (%d)", markExam.StudentExamID, markExam.Score, markExam.Total)
	}
}
