package tools_test

import (
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	classcodes "github.com/grapinou/LazyMarking/internal/handlers/classCodes"
	"github.com/grapinou/LazyMarking/internal/handlers/periods"
	"github.com/grapinou/LazyMarking/internal/handlers/years"
	"github.com/grapinou/LazyMarking/internal/templates/data"
)

func TestExamParentDeleteHandlersClassifyDatabaseErrors(t *testing.T) {
	tests := []struct {
		name        string
		table       string
		formKey     string
		listURL     string
		messagePart string
		handler     referenceDeleteHandler
	}{
		{name: "class", table: "class_codes", formKey: "class_code_id", listURL: data.DefaultStudentRoutes.ClassCodesURL, messagePart: "Cette classe est utilisée", handler: classcodes.DeleteClassCodeHandler},
		{name: "year", table: "years", formKey: "year_id", listURL: data.DefaultExamRoutes.YearsURL, messagePart: "Cette année est utilisée", handler: years.DeleteYearHandler},
		{name: "period", table: "periods", formKey: "period_id", listURL: data.DefaultExamRoutes.PeriodsURL, messagePart: "Cette période est utilisée", handler: periods.DeletePeriodHandler},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn, queries := newExamParentDeleteHandlerDB(t)

			response := serveAuthenticatedReferenceDelete(t, url.Values{tc.formKey: {"1"}}, tc.handler, queries)
			assertReferenceDeleteResponse(t, response, http.StatusSeeOther, tc.listURL)
			assertReferenceRowCount(t, conn, tc.table, 1, 0)

			response = serveAuthenticatedReferenceDelete(t, url.Values{tc.formKey: {"999"}}, tc.handler, queries)
			assertReferenceDeleteResponse(t, response, http.StatusNotFound, "")

			response = serveAuthenticatedReferenceDelete(t, url.Values{tc.formKey: {"2"}}, tc.handler, queries)
			assertReferenceDeleteResponse(t, response, http.StatusNotFound, "")
			assertReferenceRowCount(t, conn, tc.table, 2, 1)

			response = serveAuthenticatedReferenceDelete(t, url.Values{tc.formKey: {"3"}}, tc.handler, queries)
			assertReferenceDeleteResponse(t, response, http.StatusSeeOther, data.ErrorMessageURL)
			location, err := response.Result().Location()
			if err != nil {
				t.Fatal(err)
			}
			if message := location.Query().Get("errormessage"); !strings.Contains(message, tc.messagePart) {
				t.Fatalf("error message %q does not contain %q", message, tc.messagePart)
			}
			assertReferenceRowCount(t, conn, tc.table, 3, 1)
			assertReferenceRowCount(t, conn, "exams", 10, 1)

			if _, err := conn.Exec("CREATE TRIGGER forced_delete_failure BEFORE DELETE ON " + tc.table + " BEGIN SELECT missing_delete_function(); END;"); err != nil {
				t.Fatal(err)
			}
			response = serveAuthenticatedReferenceDelete(t, url.Values{tc.formKey: {"4"}}, tc.handler, queries)
			assertReferenceDeleteResponse(t, response, http.StatusInternalServerError, "")
			assertReferenceRowCount(t, conn, tc.table, 4, 1)
			assertReferenceRowCount(t, conn, "exams", 10, 1)
		})
	}
}

func newExamParentDeleteHandlerDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec(`
		CREATE TABLE class_codes(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE years(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE periods(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE qcm(id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE exams(
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			qcm_id INTEGER NOT NULL REFERENCES qcm(id) ON DELETE RESTRICT,
			class_code_id INTEGER NOT NULL REFERENCES class_codes(id) ON DELETE RESTRICT,
			year_id INTEGER NOT NULL REFERENCES years(id) ON DELETE RESTRICT,
			period_id INTEGER NOT NULL REFERENCES periods(id) ON DELETE RESTRICT,
			user_id INTEGER NOT NULL
		);
		INSERT INTO class_codes VALUES(1,'free',1),(2,'foreign',2),(3,'used',1),(4,'broken',1);
		INSERT INTO years VALUES(1,'free',1),(2,'foreign',2),(3,'used',1),(4,'broken',1);
		INSERT INTO periods VALUES(1,'free',1),(2,'foreign',2),(3,'used',1),(4,'broken',1);
		INSERT INTO qcm VALUES(1,'owned',1);
		INSERT INTO exams VALUES(10,'evaluation',1,3,3,3,1);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, db.New(conn)
}
