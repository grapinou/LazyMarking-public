package questions

import (
	"net/url"
	"sort"
	"strings"
	"unicode"

	"github.com/grapinou/LazyMarking/internal/questionfamilies"
	"golang.org/x/text/unicode/norm"
)

type libraryFilterView struct {
	Search, Subject, Level, Theme string
	Subjects, Levels, Themes      []string
}

func libraryFilters(query url.Values, families []questionfamilies.QuestionFamily) libraryFilterView {
	view := libraryFilterView{Search: strings.TrimSpace(query.Get("q")), Subject: query.Get("subject"), Level: query.Get("level"), Theme: query.Get("theme")}
	subjects, levels, themes := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, family := range families {
		subjects[family.Main.SubjectName] = true
		levels[family.Main.YearLevelName] = true
		themes[family.Main.ThemeName] = true
	}
	options := func(values map[string]bool) []string {
		var result []string
		for value := range values {
			if value != "" {
				result = append(result, value)
			}
		}
		sort.Strings(result)
		return result
	}
	view.Subjects, view.Levels, view.Themes = options(subjects), options(levels), options(themes)
	return view
}

func searchText(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, norm.NFD.String(strings.ToLower(value)))
}

func filterLibrary(families []questionfamilies.QuestionFamily, filter libraryFilterView) []questionfamilies.QuestionFamily {
	result := make([]questionfamilies.QuestionFamily, 0, len(families))
	search := searchText(filter.Search)
	for _, family := range families {
		q := family.Main
		if filter.Subject != "" && q.SubjectName != filter.Subject ||
			filter.Level != "" && q.YearLevelName != filter.Level ||
			filter.Theme != "" && q.ThemeName != filter.Theme {
			continue
		}
		text := q.Content + " " + q.SubjectName + " " + q.YearLevelName + " " + q.ThemeName + " " + q.SkillName
		for _, variant := range family.Variants {
			text += " " + variant.Content
		}
		if strings.Contains(searchText(text), search) {
			result = append(result, family)
		}
	}
	return result
}
