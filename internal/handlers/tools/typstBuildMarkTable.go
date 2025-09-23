package tools

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/grapinou/LazyMarking/internal/config"
)

func TypstBuildMarkTable(tempDir string, markExams []config.MarkExam, mean, stdDev, median float64,
	globalSkills map[int64]config.CounterTag,
	globalThemeSkills map[string]config.CounterTag,
) (string, bool) {
	refTypst := config.RefMarkTableTypst // fichier existant

	examName := markExams[0].ExamName
	className := markExams[0].ClassName

	// 1. Ouvrir l'ancien fichier pour lecture
	input, err := os.Open(refTypst)
	if err != nil {
		log.Printf("Can't open file : %s, error : %v", refTypst, err)
		return "", false
	}
	defer input.Close()

	// 2. Créer le nouveau fichier (écrasement s’il existe)
	typstFilePath := filepath.Join(tempDir, fmt.Sprintf("result_%s_%s.typ", examName, className))
	output, err := os.Create(typstFilePath)
	if err != nil {
		log.Printf("Can't create file : %s, error : %v", typstFilePath, err)
		return "", false
	}
	defer output.Close()

	// 3. Écrire une ligne au début

	tot := markExams[0].Total
	meanTypst := fmt.Sprintf("#let mean=\"%.2f/%d\" \n", mean, tot) // #let mean="4.04/7"
	stdDevTypst := fmt.Sprintf("#let std=\"%.2f/%d\" \n", stdDev, tot)
	medianTypst := fmt.Sprintf("#let med=\"%.2f/%d\" \n", median, tot)

	stat := meanTypst + stdDevTypst + medianTypst

	_, err = output.WriteString(stat)
	if err != nil {
		log.Printf("can't write : %s, error : %v", stat, err)
		return "", false
	}

	// 4. Copier le contenu de l’ancien fichier
	_, err = io.Copy(output, input)
	if err != nil {
		log.Printf("Can't copy ref file into : %s, error : %v", typstFilePath, err)
		return "", false
	}

	// Tri par LastName
	sort.Slice(markExams, func(i, j int) bool {
		return markExams[i].LastName < markExams[j].LastName
	})

	// 5. Ajouter des lignes à la fin
	content := "\n"
	for _, exam := range markExams {
		add := fmt.Sprintf("\"%s %s\", \"%.2f/%d\",", exam.FirstName, exam.LastName, exam.Score, exam.Total)
		content += add
	}

	content += ")\n"

	_, err = output.WriteString(content)
	if err != nil {
		log.Printf("Can't write content, error : %v", err)
		return "", false
	}

	// tableau skills - theme skill
	contentSkill := "#text(20pt)[*Réussite des compétences globales :*]\n"
	contentSkill += "#table(columns: (auto, auto),align: center,table.header([*Compétence*], [*Réussite en %*],),"

	for _, value := range globalSkills {
		name := value.Name
		success := (value.Score / float64(value.Total)) * 100

		add := fmt.Sprintf("\"%s\", \"%.2f\", ", name, success)
		contentSkill += add
	}
	contentSkill += ")\n"

	contentThemeSkill := "#text(20pt)[*Réussite des compétences par thèmes :*]\n"
	contentThemeSkill += "#table(columns: (auto, auto),align: center,table.header([*Thème - Compétence*], [*Réussite en %*],),"

	for _, value := range globalThemeSkills {
		name := value.Name
		success := (value.Score / float64(value.Total)) * 100

		add := fmt.Sprintf("\"%s\", \"%.2f\", ", name, success)
		contentThemeSkill += add
	}
	contentThemeSkill += ")\n"

	totAdd := contentSkill + contentThemeSkill
	_, err = output.WriteString(totAdd)
	if err != nil {
		log.Printf("Can't write content, error : %v", err)
		return "", false
	}

	return typstFilePath, true
}
