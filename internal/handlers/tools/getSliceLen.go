package tools

import "github.com/grapinou/LazyMarking/internal/db"

func GetSliceLen(v any) (int, bool) {
	ok := true
	switch slice := v.(type) {
	case []db.Subject:
		return len(slice), ok
	case []db.Theme:
		return len(slice), ok
	case []db.YearLevel:
		return len(slice), ok
	case []db.Skill:
		return len(slice), ok
	case []db.Difficulty:
		return len(slice), ok
	case []db.Point:
		return len(slice), ok
	default:
		ok = false
		return 0, ok
	}
}
