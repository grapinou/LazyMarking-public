package tools

import (
	"errors"
	"log"
	"net/http"

	"github.com/grapinou/LazyMarking/internal/db"
)

var ErrClassCodeWithNoStudents = errors.New("no student found in class code")

// retourne tous les students d'une classe selon son class code id.
// s'il n'y a pas d'élève dans la classe, renvoie false.
func GetAllStudentsFromClassCode(userID, classCodeID int64, r *http.Request, queries *db.Queries) ([]db.Student, error) {

	studentsID, err := queries.GetAllStudentsByClassCodeID(r.Context(), db.GetAllStudentsByClassCodeIDParams{
		ClassCodeID: classCodeID,
		UserID:      userID,
	})
	if err != nil {
		log.Printf("From GetAllStudentsFromClassCode -> GetAllStudentsByClassCodeID : error DB : %v", err)
		return studentsID, err
	}
	if len(studentsID) == 0 {
		return studentsID, ErrClassCodeWithNoStudents
	}

	return studentsID, nil
}
