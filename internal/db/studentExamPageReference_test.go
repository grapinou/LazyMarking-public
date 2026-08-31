package db

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

func TestStudentExamPageReferenceMigrationAndOwnershipContract(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY, username TEXT NOT NULL);
		CREATE TABLE exams_generated(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam(id INTEGER PRIMARY KEY, exam_generated_id INTEGER NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam_page_content(
			id INTEGER PRIMARY KEY AUTOINCREMENT, student_exam_id INTEGER NOT NULL,
			page INTEGER NOT NULL, content TEXT NOT NULL, user_id INTEGER NOT NULL,
			UNIQUE(student_exam_id, page, content, user_id));
		INSERT INTO users VALUES(1,'alice'),(2,'bob');
		INSERT INTO exams_generated VALUES(10,1),(20,2);
		INSERT INTO student_exam VALUES(100,10,1),(200,20,2);
		INSERT INTO student_exam_page_content(student_exam_id,page,content,user_id)
		VALUES(100,1,'historical page one',1),(100,2,'historical page two',1),
			(100,3,'legacy duplicate A',1),(100,3,'legacy duplicate B',1),(200,1,'bob page',2);
	`); err != nil {
		t.Fatal(err)
	}
	up, down := pageReferenceMigration(t)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("migration Up: %v", err)
	}

	var content string
	var key sql.NullString
	if err := conn.QueryRow(`SELECT content, reference_storage_key FROM student_exam_page_content WHERE student_exam_id=100 AND page=1`).Scan(&content, &key); err != nil {
		t.Fatal(err)
	}
	if content != "historical page one" || key.Valid {
		t.Fatalf("legacy row content=%q key=%v", content, key)
	}
	if _, err := conn.Exec(`UPDATE student_exam_page_content SET reference_storage_key='references/x.png' WHERE student_exam_id=100 AND page=1`); err == nil {
		t.Fatal("partial metadata update succeeded")
	}
	if _, err := conn.Exec(`INSERT INTO student_exam_page_content(student_exam_id,page,content,user_id,reference_storage_key) VALUES(100,4,'partial',1,'references/x.png')`); err == nil {
		t.Fatal("partial metadata insert succeeded")
	}
	if _, err := conn.Exec(`UPDATE student_exam_page_content SET
		reference_storage_key='references/student-exam-100/page-1.png', reference_width=2480,
		reference_height=3508, reference_dpi=299, reference_sha256=?
		WHERE student_exam_id=100 AND page=1`, strings.Repeat("a", 64)); err == nil {
		t.Fatal("non-300 DPI update succeeded")
	}

	queries := New(conn)
	params := validPageReferenceParams(1, 100, 1)
	rows, err := queries.SetStudentExamPageReference(t.Context(), params)
	if err != nil || rows != 1 {
		t.Fatalf("set valid reference rows=%d err=%v", rows, err)
	}
	got, err := queries.GetStudentExamPageReference(t.Context(), GetStudentExamPageReferenceParams{StudentExamID: 100, Page: 1, UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got.GenerationID != 10 || got.ReferenceStorageKey.String != params.ReferenceStorageKey.String || got.ReferenceWidth.Int64 != 2480 || got.ReferenceHeight.Int64 != 3508 || got.ReferenceDpi.Int64 != 300 || got.ReferenceSha256.String != params.ReferenceSha256.String {
		t.Fatalf("reference=%+v", got)
	}
	if rows, err := queries.SetStudentExamPageReference(t.Context(), params); err != nil || rows != 1 {
		t.Fatalf("identical no-op rows=%d err=%v", rows, err)
	}

	for name, statement := range map[string]string{
		"storage key": `UPDATE student_exam_page_content SET reference_storage_key='references/changed.png' WHERE student_exam_id=100 AND page=1`,
		"hash":        `UPDATE student_exam_page_content SET reference_sha256='` + strings.Repeat("b", 64) + `' WHERE student_exam_id=100 AND page=1`,
		"width":       `UPDATE student_exam_page_content SET reference_width=1 WHERE student_exam_id=100 AND page=1`,
		"dpi":         `UPDATE student_exam_page_content SET reference_dpi=300+1 WHERE student_exam_id=100 AND page=1`,
		"clear":       `UPDATE student_exam_page_content SET reference_storage_key=NULL,reference_width=NULL,reference_height=NULL,reference_dpi=NULL,reference_sha256=NULL WHERE student_exam_id=100 AND page=1`,
	} {
		t.Run("immutable "+name, func(t *testing.T) {
			if _, err := conn.Exec(statement); err == nil {
				t.Fatal("metadata mutation succeeded")
			}
		})
	}

	for _, tc := range []struct {
		name   string
		params SetStudentExamPageReferenceParams
	}{
		{name: "absent page", params: validPageReferenceParams(1, 100, 99)},
		{name: "wrong user", params: validPageReferenceParams(2, 100, 2)},
		{name: "foreign student exam", params: validPageReferenceParams(1, 200, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := queries.SetStudentExamPageReference(t.Context(), tc.params)
			if err != nil || rows != 0 {
				t.Fatalf("rows=%d err=%v, want zero", rows, err)
			}
		})
	}
	if _, err := conn.Exec(`INSERT INTO student_exam_page_content(student_exam_id,page,content,user_id) VALUES(100,2,'different content',1)`); err == nil {
		t.Fatal("new duplicate page succeeded")
	}
	duplicateParams := validPageReferenceParams(1, 100, 3)
	if rows, err := queries.SetStudentExamPageReference(t.Context(), duplicateParams); err != nil || rows != 0 {
		t.Fatalf("legacy duplicate set rows=%d err=%v, want zero", rows, err)
	}
	if _, err := queries.GetStudentExamPageReference(t.Context(), GetStudentExamPageReferenceParams{StudentExamID: 100, Page: 3, UserID: 1}); err != sql.ErrNoRows {
		t.Fatalf("legacy duplicate read error=%v, want sql.ErrNoRows", err)
	}
	if _, err := conn.Exec(`UPDATE student_exam_page_content SET
		reference_storage_key='references/../outside.png',reference_width=1,reference_height=1,reference_dpi=300,reference_sha256=?
		WHERE student_exam_id=100 AND page=2`, strings.Repeat("a", 64)); err == nil {
		t.Fatal("traversal storage key succeeded in DB")
	}

	if _, err := conn.Exec(down); err != nil {
		t.Fatalf("migration Down: %v", err)
	}
	if err := conn.QueryRow(`SELECT content FROM student_exam_page_content WHERE student_exam_id=100 AND page=1`).Scan(&content); err != nil || content != "historical page one" {
		t.Fatalf("historical content after Down=%q err=%v", content, err)
	}
	if _, err := conn.Exec(`SELECT reference_storage_key FROM student_exam_page_content`); err == nil {
		t.Fatal("reference column remains after Down")
	}
}

func validPageReferenceParams(userID, studentExamID, page int64) SetStudentExamPageReferenceParams {
	return SetStudentExamPageReferenceParams{
		ReferenceStorageKey: sql.NullString{String: "references/student-exam-100/page-1.png", Valid: true},
		ReferenceWidth:      sql.NullInt64{Int64: 2480, Valid: true}, ReferenceHeight: sql.NullInt64{Int64: 3508, Valid: true},
		ReferenceDpi: sql.NullInt64{Int64: 300, Valid: true}, ReferenceSha256: sql.NullString{String: strings.Repeat("a", 64), Valid: true},
		StudentExamID: studentExamID, Page: page, UserID: userID,
	}
}

func pageReferenceMigration(t *testing.T) (string, string) {
	t.Helper()
	b, err := os.ReadFile("../../db/migrations/0039_add_student_exam_page_reference.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(b), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("migration has no Down")
	}
	return strings.Replace(parts[0], "-- +goose Up", "", 1), parts[1]
}
