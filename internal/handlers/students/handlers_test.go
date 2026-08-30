package students

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/login"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestTableStudentsReturnsInternalServerErrorWhenStudentQueryFails(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	response := serveAuthenticatedStudentRequest(t, http.MethodGet, "/dashboard/students", nil, func(w http.ResponseWriter, r *http.Request) {
		TableStudentsHandler(w, r, queries)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	if strings.Contains(response.Body.String(), "<html") || strings.Contains(response.Body.String(), "Aucun élève") {
		t.Fatalf("normal student page rendered after DB failure: %q", response.Body.String())
	}
}

func TestAddStudentClassifiesUniqueAndCheckConstraints(t *testing.T) {
	tests := []struct {
		name    string
		form    url.Values
		message string
	}{
		{
			name:    "duplicate",
			form:    url.Values{"first_name": {"Owned"}, "last_name": {"Student"}, "class_code_id": {"10"}},
			message: "Cet élève existe déjà.",
		},
		{
			name:    "blank first name",
			form:    url.Values{"first_name": {"   "}, "last_name": {"Student"}, "class_code_id": {"10"}},
			message: "Le prénom et le nom de l’élève doivent être renseignés.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, queries := newStudentHandlerTestDB(t)
			response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", test.form, func(w http.ResponseWriter, r *http.Request) {
				AddStudentHandler(w, r, queries, conn)
			})
			assertBusinessRedirectMessage(t, response, test.message)
			assertTableRowCount(t, conn, "students", "user_id = 1", 1)
			assertTableRowCount(t, conn, "student_class_codes", "user_id = 1", 0)
		})
	}
}

func TestAddStudentReturnsInternalServerErrorForUnexpectedDBFailure(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_student_insert BEFORE INSERT ON students
		BEGIN SELECT RAISE(ABORT, 'forced unexpected insert failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", url.Values{"first_name": {"New"}, "last_name": {"Student"}, "class_code_id": {"10"}}, func(w http.ResponseWriter, r *http.Request) {
		AddStudentHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	assertTableRowCount(t, conn, "students", "first_name = 'New'", 0)
	assertTableRowCount(t, conn, "student_class_codes", "user_id = 1", 0)
}

func TestAddStudentRollsBackUnexpectedFirstClassFailure(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_first_class_insert BEFORE INSERT ON student_class_codes
		BEGIN SELECT RAISE(ABORT, 'forced unexpected class relation failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", url.Values{"first_name": {"Created"}, "last_name": {"Then rollback"}, "class_code_id": {"10"}}, func(w http.ResponseWriter, r *http.Request) {
		AddStudentHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	assertTableRowCount(t, conn, "students", "first_name = 'Created'", 0)
	assertTableRowCount(t, conn, "student_class_codes", "user_id = 1", 0)
}

func TestManualAndCSVAddPreserveLongUnicodeIdentities(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	firstName := "Éléonore-Alexandrine-Çağdaş-李小龍"
	manualLastName := "D’Estaing-Manuel-Ångström-非常に長い名前"
	csvLastName := "D’Estaing-CSV-Coëffé-非常に長い名前"

	manualResponse := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", url.Values{
		"first_name": {"  " + firstName + "  "}, "last_name": {"  " + manualLastName + "  "}, "class_code_id": {"10"},
	}, func(w http.ResponseWriter, r *http.Request) {
		AddStudentHandler(w, r, queries, conn)
	})
	if manualResponse.Code != http.StatusSeeOther {
		t.Fatalf("manual status=%d, want 303", manualResponse.Code)
	}

	csvContent := "  " + firstName + "  ;  " + csvLastName + "  \n"
	csvResponse := serveAuthenticatedStudentCSVRequest(t, csvContent, "10", func(w http.ResponseWriter, r *http.Request) {
		AddCSVStudentHandler(w, r, queries, conn)
	})
	if csvResponse.Code != http.StatusSeeOther {
		t.Fatalf("CSV status=%d body=%q, want 303", csvResponse.Code, csvResponse.Body.String())
	}

	for _, lastName := range []string{manualLastName, csvLastName} {
		var storedFirstName, storedLastName string
		if err := conn.QueryRow("SELECT first_name, last_name FROM students WHERE user_id = 1 AND last_name = ?", lastName).Scan(&storedFirstName, &storedLastName); err != nil {
			t.Fatal(err)
		}
		if storedFirstName != firstName || storedLastName != lastName {
			t.Fatalf("stored identity=(%q, %q), want (%q, %q)", storedFirstName, storedLastName, firstName, lastName)
		}
	}
}

func TestAddCSVStudentRejectsOversizedMultipartBeforeImport(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	csvContent := strings.Repeat("x", int(tools.MaxCSVRequestBytes))

	response := serveAuthenticatedStudentCSVRequest(t, csvContent, "10", func(w http.ResponseWriter, r *http.Request) {
		AddCSVStudentHandler(w, r, queries, conn)
	})

	assertBusinessRedirectMessage(t, response, "La requête d’import CSV dépasse la taille maximale autorisée de 2 Mio.")
	assertTableRowCount(t, conn, "students", "user_id = 1", 1)
	assertTableRowCount(t, conn, "student_class_codes", "user_id = 1", 0)
}

func TestAddCSVStudentPreservesLiteralQuotesExactly(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	firstName := `"Jean ""Junior"""`
	lastName := `D'Arc "Senior"`
	csvContent := encodeStudentCSV(t, []string{firstName, lastName})

	response := serveAuthenticatedStudentCSVRequest(t, csvContent, "10", func(w http.ResponseWriter, r *http.Request) {
		AddCSVStudentHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != data.DefaultDashboardRoutes.StudentURL {
		t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
	}

	var storedFirstName, storedLastName string
	if err := conn.QueryRow("SELECT first_name, last_name FROM students WHERE user_id = 1 AND first_name = ?", firstName).Scan(&storedFirstName, &storedLastName); err != nil {
		t.Fatal(err)
	}
	if storedFirstName != firstName || storedLastName != lastName {
		t.Fatalf("stored identity=(%q, %q), want (%q, %q)", storedFirstName, storedLastName, firstName, lastName)
	}
}

func TestAddCSVStudentDuplicateRollsBackWholeBatch(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	csvContent := encodeStudentCSV(t,
		[]string{"First", "Imported"},
		[]string{"Owned", "Student"},
	)

	response := serveAuthenticatedStudentCSVRequest(t, csvContent, "10", func(w http.ResponseWriter, r *http.Request) {
		AddCSVStudentHandler(w, r, queries, conn)
	})
	assertBusinessRedirectMessage(t, response, "Cet élève existe déjà.")
	assertTableRowCount(t, conn, "students", "first_name = 'First' AND last_name = 'Imported'", 0)
	assertTableRowCount(t, conn, "student_class_codes", "user_id = 1", 0)
}

func TestAddCSVStudentUnexpectedSecondInsertFailureReturns500AndRollsBackBatch(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_second_csv_student BEFORE INSERT ON students
		WHEN NEW.first_name = 'Break'
		BEGIN SELECT RAISE(ABORT, 'forced unexpected CSV insert failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	csvContent := encodeStudentCSV(t,
		[]string{"First", "Imported"},
		[]string{"Break", "Import"},
	)

	response := serveAuthenticatedStudentCSVRequest(t, csvContent, "10", func(w http.ResponseWriter, r *http.Request) {
		AddCSVStudentHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d location=%q, want 500", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Header().Get("Location"), "existe") {
		t.Fatalf("unexpected duplicate redirect: %q", response.Header().Get("Location"))
	}
	assertTableRowCount(t, conn, "students", "first_name IN ('First', 'Break')", 0)
	assertTableRowCount(t, conn, "student_class_codes", "user_id = 1", 0)
}

func TestAddCSVStudentRelationFailureReturns500WithoutOrphan(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_csv_relation BEFORE INSERT ON student_class_codes
		BEGIN SELECT RAISE(ABORT, 'forced unexpected CSV relation failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	csvContent := encodeStudentCSV(t, []string{"No", "Orphan"})

	response := serveAuthenticatedStudentCSVRequest(t, csvContent, "10", func(w http.ResponseWriter, r *http.Request) {
		AddCSVStudentHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	assertTableRowCount(t, conn, "students", "first_name = 'No' AND last_name = 'Orphan'", 0)
	assertTableRowCount(t, conn, "student_class_codes", "user_id = 1", 0)
}

func TestEditStudentClassifiesUniqueAndCheckConstraints(t *testing.T) {
	tests := []struct {
		name    string
		first   string
		last    string
		message string
	}{
		{"duplicate", "Other", "Owned", "Cet élève existe déjà."},
		{"blank last name", "Owned", "   ", "Le prénom et le nom de l’élève doivent être renseignés."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, queries := newStudentHandlerTestDB(t)
			if _, err := conn.Exec("INSERT INTO students(id, first_name, last_name, user_id) VALUES(3, 'Other', 'Owned', 1)"); err != nil {
				t.Fatal(err)
			}
			form := url.Values{"student_id": {"1"}, "new_first_name": {test.first}, "new_last_name": {test.last}}
			response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
				EditStudentHandler(w, r, queries)
			})
			assertBusinessRedirectMessage(t, response, test.message)
			assertTableRowCount(t, conn, "students", "id = 1 AND first_name = 'Owned' AND last_name = 'Student'", 1)
		})
	}
}

func TestEditStudentReturnsInternalServerErrorForUnexpectedDBFailure(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_student_update BEFORE UPDATE ON students
		BEGIN SELECT RAISE(ABORT, 'forced unexpected update failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	form := url.Values{"student_id": {"1"}, "new_first_name": {"Changed"}, "new_last_name": {"Student"}}
	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
		EditStudentHandler(w, r, queries)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	assertTableRowCount(t, conn, "students", "id = 1 AND first_name = 'Owned' AND last_name = 'Student'", 1)
}

func TestAddStudentRollsBackWhenClassOwnershipFails(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	form := url.Values{
		"first_name":    {"Should"},
		"last_name":     {"Rollback"},
		"class_code_id": {"20"},
	}
	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", form, func(w http.ResponseWriter, r *http.Request) {
		AddStudentHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM students WHERE first_name = 'Should' AND last_name = 'Rollback'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partially created students = %d, want 0", count)
	}
}

func TestStudentHandlersReturnNotFoundForMissingOrForeignStudent(t *testing.T) {
	_, queries := newStudentHandlerTestDB(t)
	tests := []struct {
		name   string
		method string
		target string
		form   url.Values
		serve  func(http.ResponseWriter, *http.Request)
	}{
		{name: "update missing", method: http.MethodPost, target: "/", form: url.Values{"student_id": {"999"}, "new_first_name": {"New"}, "new_last_name": {"Name"}}, serve: func(w http.ResponseWriter, r *http.Request) { EditStudentHandler(w, r, queries) }},
		{name: "delete foreign", method: http.MethodPost, target: "/", form: url.Values{"student_id": {"2"}}, serve: func(w http.ResponseWriter, r *http.Request) { DeleteStudentHandler(w, r, queries) }},
		{name: "edit form foreign", method: http.MethodGet, target: "/?student_id=2", serve: func(w http.ResponseWriter, r *http.Request) { EditFormStudentHandler(w, r, queries) }},
		{name: "delete form missing", method: http.MethodGet, target: "/?student_id=999", serve: func(w http.ResponseWriter, r *http.Request) { DeleteFormStudentHandler(w, r, queries) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			response := serveAuthenticatedStudentRequest(t, tc.method, tc.target, tc.form, tc.serve)
			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.Code)
			}
		})
	}
}

func TestDeleteAllStudentsRollsBackWhenMembershipRemovalFails(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		INSERT INTO students(id, first_name, last_name, user_id) VALUES (3, 'Multi', 'Class', 1);
		INSERT INTO class_codes VALUES (11, 'Other owned', 1);
		INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES (1, 10, 1), (3, 10, 1), (3, 11, 1);
		CREATE TRIGGER fail_membership_delete BEFORE DELETE ON student_class_codes
		WHEN OLD.student_id = 3
		BEGIN SELECT RAISE(ABORT, 'forced membership failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", url.Values{"class_code_id": {"10"}}, func(w http.ResponseWriter, r *http.Request) {
		DeleteAllStudentsHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM students WHERE id = 1 AND user_id = 1").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("owned student count after rollback = %d, want 1", count)
	}
}

func TestDeleteStudentReturnsBusinessErrorWhenExamHistoryProtectsStudent(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		INSERT INTO exams_generated(id, user_id) VALUES (100, 1);
		INSERT INTO student_exam(id, exam_generated_id, student_id, user_id) VALUES (200, 100, 1, 1);
	`); err != nil {
		t.Fatal(err)
	}

	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", url.Values{"student_id": {"1"}}, func(w http.ResponseWriter, r *http.Request) {
		DeleteStudentHandler(w, r, queries)
	})
	assertBusinessRedirectMessage(t, response, "Cet élève ne peut pas être supprimé car il est déjà associé à une évaluation générée.")
	assertTableRowCount(t, conn, "students", "id = 1", 1)
	assertTableRowCount(t, conn, "student_exam", "id = 200", 1)
}

func TestDeleteStudentStillDeletesStudentWithoutExamHistory(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", url.Values{"student_id": {"1"}}, func(w http.ResponseWriter, r *http.Request) {
		DeleteStudentHandler(w, r, queries)
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard/students" {
		t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	assertTableRowCount(t, conn, "students", "id = 1", 0)
}

func TestDeleteStudentKeepsUnexpectedConstraintAsInternalServerError(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		CREATE TRIGGER fail_student_delete BEFORE DELETE ON students
		WHEN OLD.id = 1
		BEGIN SELECT RAISE(ABORT, 'forced unexpected failure'); END;
	`); err != nil {
		t.Fatal(err)
	}
	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", url.Values{"student_id": {"1"}}, func(w http.ResponseWriter, r *http.Request) {
		DeleteStudentHandler(w, r, queries)
	})
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	assertTableRowCount(t, conn, "students", "id = 1", 1)
}

func TestDeleteAllStudentsRollsBackWhenExamHistoryProtectsStudent(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		INSERT INTO students(id, first_name, last_name, user_id) VALUES
			(3, 'Protected', 'Mono', 1),
			(4, 'Ordinary', 'Mono', 1),
			(5, 'Several', 'Classes', 1);
		INSERT INTO class_codes VALUES (11, 'Other owned', 1);
		INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES
			(3, 10, 1), (4, 10, 1), (5, 10, 1), (5, 11, 1);
		INSERT INTO exams_generated(id, user_id) VALUES (100, 1);
		INSERT INTO student_exam(id, exam_generated_id, student_id, user_id) VALUES (200, 100, 3, 1);
	`); err != nil {
		t.Fatal(err)
	}

	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", url.Values{"class_code_id": {"10"}}, func(w http.ResponseWriter, r *http.Request) {
		DeleteAllStudentsHandler(w, r, queries, conn)
	})
	assertBusinessRedirectMessage(t, response, "Impossible de supprimer les élèves de cette classe car au moins un élève est déjà associé à une évaluation générée.")
	for _, condition := range []string{"id = 3", "id = 4", "id = 5"} {
		assertTableRowCount(t, conn, "students", condition, 1)
	}
	for _, condition := range []string{
		"student_id = 3 AND class_code_id = 10",
		"student_id = 4 AND class_code_id = 10",
		"student_id = 5 AND class_code_id = 10",
		"student_id = 5 AND class_code_id = 11",
	} {
		assertTableRowCount(t, conn, "student_class_codes", condition, 1)
	}
	assertTableRowCount(t, conn, "student_exam", "id = 200", 1)
}

func TestDeleteAllStudentsKeepsExistingCardinalityBehaviorWithoutExamHistory(t *testing.T) {
	conn, queries := newStudentHandlerTestDB(t)
	if _, err := conn.Exec(`
		INSERT INTO students(id, first_name, last_name, user_id) VALUES
			(3, 'Ordinary', 'Mono', 1),
			(4, 'Several', 'Classes', 1);
		INSERT INTO class_codes VALUES (11, 'Other owned', 1);
		INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES
			(3, 10, 1), (4, 10, 1), (4, 11, 1);
	`); err != nil {
		t.Fatal(err)
	}

	response := serveAuthenticatedStudentRequest(t, http.MethodPost, "/", url.Values{"class_code_id": {"10"}}, func(w http.ResponseWriter, r *http.Request) {
		DeleteAllStudentsHandler(w, r, queries, conn)
	})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/dashboard/students" {
		t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	assertTableRowCount(t, conn, "students", "id = 3", 0)
	assertTableRowCount(t, conn, "students", "id = 4", 1)
	assertTableRowCount(t, conn, "student_class_codes", "student_id = 4 AND class_code_id = 10", 0)
	assertTableRowCount(t, conn, "student_class_codes", "student_id = 4 AND class_code_id = 11", 1)
}

func newStudentHandlerTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE students (id INTEGER PRIMARY KEY AUTOINCREMENT, first_name TEXT NOT NULL CHECK(length(trim(first_name)) > 0), last_name TEXT NOT NULL CHECK(length(trim(last_name)) > 0), user_id INTEGER NOT NULL, UNIQUE(user_id, first_name, last_name));
		CREATE TABLE class_codes (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE student_class_codes (id INTEGER PRIMARY KEY, student_id INTEGER NOT NULL REFERENCES students(id) ON DELETE CASCADE, class_code_id INTEGER NOT NULL, user_id INTEGER NOT NULL, UNIQUE(student_id, class_code_id, user_id));
		CREATE TABLE exams_generated (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam (
			id INTEGER PRIMARY KEY,
			exam_generated_id INTEGER NOT NULL REFERENCES exams_generated(id) ON DELETE CASCADE,
			student_id INTEGER NOT NULL REFERENCES students(id),
			user_id INTEGER NOT NULL,
			UNIQUE(exam_generated_id, student_id, user_id)
		);
		INSERT INTO students(id, first_name, last_name, user_id) VALUES (1, 'Owned', 'Student', 1), (2, 'Foreign', 'Student', 2);
		INSERT INTO class_codes VALUES (10, 'Owned', 1), (20, 'Foreign', 2);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}

func assertBusinessRedirectMessage(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status=%d, want 303", response.Code)
	}
	location, err := url.Parse(response.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if location.Path != data.ErrorMessageURL || location.Query().Get("errormessage") != want {
		t.Fatalf("location=%q message=%q", response.Header().Get("Location"), location.Query().Get("errormessage"))
	}
}

func assertTableRowCount(t *testing.T, conn *sql.DB, table, condition string, want int) {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM " + table + " WHERE " + condition).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("%s WHERE %s count=%d, want %d", table, condition, count, want)
	}
}

func serveAuthenticatedStudentRequest(t *testing.T, method, target string, form url.Values, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("SESSION_KEY", "student-handler-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	body := strings.NewReader(form.Encode())
	request := httptest.NewRequest(method, target, body)
	if form != nil {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	session, err := login.GetStore().Get(request, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	session.Values["username"] = "test-user"
	cookies := httptest.NewRecorder()
	if err := session.Save(request, cookies); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range cookies.Result().Cookies() {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	login.CheckAuth(handler).ServeHTTP(response, request)
	return response
}

func serveAuthenticatedStudentCSVRequest(t *testing.T, csvContent, classCodeID string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("class_code_id", classCodeID); err != nil {
		t.Fatal(err)
	}
	file, err := writer.CreateFormFile("csvfile", "students.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte(csvContent)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SESSION_KEY", "student-handler-test-key-32-bytes-long")
	t.Setenv("SESSION_SECURE", "false")
	if err := login.InitSessionStore(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	session, err := login.GetStore().Get(request, "session")
	if err != nil {
		t.Fatal(err)
	}
	session.Values["user_id"] = int64(1)
	session.Values["username"] = "test-user"
	cookies := httptest.NewRecorder()
	if err := session.Save(request, cookies); err != nil {
		t.Fatal(err)
	}
	for _, cookie := range cookies.Result().Cookies() {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	login.CheckAuth(handler).ServeHTTP(response, request)
	return response
}

func encodeStudentCSV(t *testing.T, records ...[]string) string {
	t.Helper()
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	writer.Comma = ';'
	writer.WriteAll(records)
	if err := writer.Error(); err != nil {
		t.Fatal(err)
	}
	return output.String()
}
