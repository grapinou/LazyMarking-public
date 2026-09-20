package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

// BuildMarkingGenerationPDF renders the same ordered, masked result view as the
// screen. There is no stored report to invalidate: every request reads current
// committed results, including human decisions, before rendering this snapshot.
func BuildMarkingGenerationPDF(ctx context.Context, username string, page data.MarkingGenerationPageData) ([]byte, error) {
	operation := "bilan-" + uuid.NewString()
	workspace, ok := CreateOperationTempDir(username, operation)
	if !ok {
		return nil, fmt.Errorf("create cumulative PDF workspace")
	}
	defer RemoveOperationTempDir(username, operation)
	input := filepath.Join(workspace, "bilan-evaluation.typ")
	output := filepath.Join(workspace, "bilan-evaluation.pdf")
	if err := os.WriteFile(input, []byte(markingGenerationTypst(page)), 0o600); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, externalCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "typst", "compile", "--root", workspace, input, output)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("compile cumulative PDF: %w", err)
	}
	return os.ReadFile(output)
}

func markingGenerationTypst(page data.MarkingGenerationPageData) string {
	var out strings.Builder
	out.WriteString("#set text(font: \"Liberation Sans\", size: 10pt)\n#set page(paper: \"a4\", margin: 15mm, numbering: \"1 / 1\")\n")
	out.WriteString("#heading(" + typstStringLiteral("Bilan de l’évaluation") + ")\n")
	out.WriteString("#text(" + typstStringLiteral(MarkingGenerationTitle(page.ClassName, page.ExamName)) + ")\n\n")
	fmt.Fprintf(&out, "#text(%s)\n\n", typstStringLiteral(fmt.Sprintf("Génération %d — %d copie(s) corrigée(s), %d à vérifier, %d à contrôler, %d non détectée(s).",
		page.GenerationID, page.Summary.Corrected, page.Summary.PendingReview, page.Summary.Issues, page.Summary.NotSeen)))
	writeMarkingPedagogicalTypst(&out, page.Pedagogy)
	if page.Summary.PendingReview > 0 {
		out.WriteString("#text(" + typstStringLiteral("Bilan provisoire : les notes en attente de vérification ne sont pas définitives.") + ")\n\n")
	}
	if len(page.Summary.Results) == 0 {
		out.WriteString("#text(" + typstStringLiteral("Aucun résultat disponible. Importez les copies pour établir le bilan.") + ")\n")
		return out.String()
	}
	out.WriteString("#heading(level: 2, " + typstStringLiteral("Résultats des élèves") + ")\n")
	out.WriteString("#table(columns: (2fr, 2fr, 1.5fr, 1.2fr),\n")
	// table.header automatically repeats on subsequent pages.
	out.WriteString("table.header([Élève], [État courant], [Note], [Import source]),\n")
	for _, result := range page.Summary.Results {
		score := "—"
		if result.Pending {
			score = "Après vérification"
		} else if result.HasScore {
			score = result.ScoreLabel
		}
		fmt.Fprintf(&out, "%s, %s, %s, %s,\n", typstStringLiteral(result.StudentName), typstStringLiteral(result.StatusLabel),
			typstStringLiteral(score), typstStringLiteral(fmt.Sprintf("Import %d", result.SourceJobID)))
	}
	out.WriteString(")\n")
	return out.String()
}

func writeMarkingPedagogicalTypst(out *strings.Builder, stats data.MarkingPedagogicalSummaryView) {
	text := func(value string) { out.WriteString("#text(" + typstStringLiteral(value) + ")\n\n") }
	heading := func(value string) { out.WriteString("#heading(level: 2, " + typstStringLiteral(value) + ")\n") }
	heading("Résumé pédagogique")
	text(fmt.Sprintf("Statistiques sur %d copie(s) corrigée(s) sans revue en attente. %d copie(s) exclue(s).", stats.IncludedCopies, stats.ExcludedCopies))
	text("Les copies non détectées, incomplètes, en erreur ou à vérifier ne comptent pas comme des zéros. Les notes sans barème positif sont également exclues.")
	if len(stats.ScoreGroups) == 0 {
		text("Moyenne, médiane et écart-type : indisponibles, aucune note finalisée exploitable.")
	}
	if len(stats.ScoreGroups) > 1 {
		text("Les barèmes diffèrent : statistiques présentées séparément pour chaque barème.")
	}
	for _, group := range stats.ScoreGroups {
		text(fmt.Sprintf("Barème : / %d — %d copie(s)", group.Total, group.Count))
		text(fmt.Sprintf("Moyenne : %s / %d — Médiane : %s / %d — Écart-type : %s", group.Mean, group.Total, group.Median, group.Total, group.StdDev))
	}
	heading("Réussite par question")
	text(fmt.Sprintf("Détails pédagogiques disponibles pour %d / %d copie(s) incluse(s).", stats.DetailedCopies, stats.IncludedCopies))
	if stats.DetailedCopies < stats.IncludedCopies {
		text("Les copies dont le détail historique est absent ou incohérent restent dans la moyenne, mais sont exclues des statistiques par question et compétence.")
	}
	if len(stats.Questions) == 0 {
		text("Aucun résultat par question exploitable pour le moment.")
	} else {
		text("Les numéros ci-dessous sont des repères du bilan, indépendants de l'ordre sur les copies. Chaque version d'une famille de questions est distinguée. Réussite = points obtenus / points possibles, crédit partiel inclus.")
		out.WriteString("#table(columns: (3.5fr, 1fr, 1fr, 1fr), table.header([Question / version], [Réussite], [Entièrement réussies], [Évaluées]),\n")
		for _, question := range stats.Questions {
			fmt.Fprintf(out, "%s, %s, %s, %s,\n", typstStringLiteral(question.Label), typstStringLiteral(question.Success+" %"), typstStringLiteral(fmt.Sprint(question.Correct)), typstStringLiteral(fmt.Sprint(question.Count)))
		}
		out.WriteString(")\n")
	}
	for _, section := range []struct {
		title string
		rows  []data.MarkingSuccessRateView
	}{{"Réussite des compétences globales", stats.Skills}, {"Réussite des compétences par thème", stats.ThemeSkills}} {
		if len(section.rows) == 0 {
			continue
		}
		heading(section.title)
		text("Pourcentage des points obtenus sur les points possibles, crédit partiel inclus.")
		out.WriteString("#table(columns: (3fr, 1fr), table.header([Compétence], [Réussite]),\n")
		for _, row := range section.rows {
			fmt.Fprintf(out, "%s, %s,\n", typstStringLiteral(row.Label), typstStringLiteral(row.Success+" %"))
		}
		out.WriteString(")\n")
	}
}
