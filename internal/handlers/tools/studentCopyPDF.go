package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/skip2/go-qrcode"
)

type StudentCoupon struct{ Name, Exam, Class, URL, Code string }

// BuildStudentCouponsPDF uses four full-width strips per A4 page. The URL is
// also printed so a damaged QR does not make the coupon unusable.
func BuildStudentCouponsPDF(ctx context.Context, username string, coupons []StudentCoupon) ([]byte, error) {
	if len(coupons) == 0 {
		return nil, errors.New("no coupons")
	}
	op := "coupons-" + uuid.NewString()
	workspace, ok := CreateOperationTempDir(username, op)
	if !ok {
		return nil, errors.New("create coupon workspace")
	}
	defer RemoveOperationTempDir(username, op)
	var body strings.Builder
	body.WriteString(`#set page(paper: "a4", margin: 10mm)
#set text(font: "Liberation Sans", size: 9pt)
#set par(spacing: 0pt)
`)
	for i, c := range coupons {
		if i > 0 && i%4 == 0 {
			body.WriteString("#pagebreak()\n")
		}
		name := fmt.Sprintf("qr-%03d.png", i)
		if err := qrcode.WriteFile(c.URL, qrcode.Medium, 400, filepath.Join(workspace, name)); err != nil {
			return nil, err
		}
		body.WriteString("#block(width: 100%, height: 60mm, inset: (left: 3mm, right: 3mm, top: 2mm), stroke: (bottom: 0.3pt + gray))[\n")
		body.WriteString("#grid(columns: (1fr, 39mm), gutter: 4mm, [\n")
		fmt.Fprintf(&body, "#text(size: 13pt, weight: \"bold\", %s) \\ \n", typstStringLiteral(c.Name))
		fmt.Fprintf(&body, "#text(size: 10pt, %s) \\ \n", typstStringLiteral(c.Exam))
		fmt.Fprintf(&body, "#text(size: 9pt, %s) \\ \n", typstStringLiteral(c.Class))
		body.WriteString("*Retrouver ma copie corrigée* \\ \n")
		fmt.Fprintf(&body, "#text(weight: \"bold\", %s) \\ \n", typstStringLiteral("Code : "+c.Code))
		body.WriteString("URL de secours : \\ \n")
		fmt.Fprintf(&body, "#text(size: 7pt, hyphenate: false, %s) \\ \n", typstStringLiteral(c.URL))
		body.WriteString("Accès pendant 14 jours après publication.\n], [")
		fmt.Fprintf(&body, "#image(%s, width: 35mm)", typstStringLiteral(name))
		body.WriteString("])\n]\n")
	}
	input := filepath.Join(workspace, "coupons.typ")
	output := filepath.Join(workspace, "coupons.pdf")
	if err := os.WriteFile(input, []byte(body.String()), 0o600); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "typst", "compile", "--root", workspace, input, output)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("compile coupons: %w", err)
	}
	return os.ReadFile(output)
}

// BuildStudentCorrectedPDF renders only one selected copy using the same
// snapshot, effective answers, aligned scans and annotation code as the
// teacher's corrected PDF. No class-level PDF is opened or extracted.
func BuildStudentCorrectedPDF(ctx context.Context, queries *db.Queries, userID int64, username string, jobID, copyResultID, studentExamID int64) ([]byte, error) {
	rows, err := queries.ListMarkingArtifactCopyResults(ctx, db.ListMarkingArtifactCopyResultsParams{MarkingJobID: jobID, UserID: userID})
	if err != nil {
		return nil, err
	}
	var selected *db.ListMarkingArtifactCopyResultsRow
	for i := range rows {
		if rows[i].ID == copyResultID && rows[i].StudentExamID == studentExamID && rows[i].Outcome == "corrected" {
			selected = &rows[i]
			break
		}
	}
	if selected == nil {
		return nil, sql.ErrNoRows
	}
	var snapshot config.QCM
	if err := json.Unmarshal([]byte(selected.SnapshotContent), &snapshot); err != nil {
		return nil, err
	}
	op := "student-copy-" + uuid.NewString()
	workspace, ok := CreateOperationTempDir(username, op)
	if !ok {
		return nil, errors.New("create copy workspace")
	}
	defer RemoveOperationTempDir(username, op)
	_, path, err := regenerateCorrectedCopy(ctx, queries, MarkingArtifactsGenerationInput{UserID: userID, Username: username, MarkingJobID: jobID, StagingDir: workspace}, *selected, snapshot)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}
