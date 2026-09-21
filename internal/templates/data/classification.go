package data

import (
	"fmt"
	"strconv"
)

const ClassificationAdvice = "Conseil : gardez des libellés clairs, complets et cohérents (par exemple « Seconde »), et évitez les abréviations inutiles. Vos appellations restent libres."

// Only an already owner-scoped list item supplies the displayed name.
func ClassificationNotice(requestedID string, id int64, name, group string) string {
	if requestedID != strconv.FormatInt(id, 10) {
		return ""
	}
	return fmt.Sprintf("« %s » existe déjà dans vos %s. La valeur existante a été conservée.", name, group)
}
