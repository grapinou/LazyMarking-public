package tools

import (
	"database/sql"
	"errors"
	"net/http"
)

// une fonction qui renvoie le field id en un sql.NullInt64 pour faire fonctionner la requete de filtrage sur les questions
// renvoie également la forme en int64
func GetFieldFiltered(fieldID string, r *http.Request) (sql.NullInt64, int64, error) {
	IDField := r.URL.Query().Get(fieldID)
	if IDField == "" {
		return sql.NullInt64{Valid: false}, -1, nil
	}
	intID, ok := StrToInt(IDField)
	if !ok {
		return sql.NullInt64{}, -1, errors.New("StrToInt return not ok")
	}
	return sql.NullInt64{Int64: intID, Valid: true}, intID, nil
}
