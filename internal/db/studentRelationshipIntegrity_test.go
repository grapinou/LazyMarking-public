package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestOwnedStudentMutationRows(t *testing.T) {
	conn, queries := newStudentRelationshipTestDB(t)
	ctx := context.Background()

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.UpdateStudent(ctx, UpdateStudentParams{FirstName: "Updated", LastName: "Owner", ID: 1, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateStudent(ctx, UpdateStudentParams{FirstName: "Missing", LastName: "Student", ID: 999, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateStudent(ctx, UpdateStudentParams{FirstName: "Forged", LastName: "Change", ID: 4, UserID: 1})
	})

	var firstName, lastName string
	if err := conn.QueryRow("SELECT first_name, last_name FROM students WHERE id = 4 AND user_id = 2").Scan(&firstName, &lastName); err != nil {
		t.Fatal(err)
	}
	if firstName != "Foreign" || lastName != "Owner" {
		t.Fatalf("foreign student changed to %q %q", firstName, lastName)
	}

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.DeleteStudent(ctx, DeleteStudentParams{ID: 1, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.DeleteStudent(ctx, DeleteStudentParams{ID: 999, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.DeleteStudent(ctx, DeleteStudentParams{ID: 4, UserID: 1})
	})
}

func TestOwnedStudentClassRelationshipMutations(t *testing.T) {
	_, queries := newStudentRelationshipTestDB(t)
	ctx := context.Background()

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.CreateStudentWithClassCode(ctx, CreateStudentWithClassCodeParams{StudentID: 3, ClassCodeID: 10, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.CreateStudentWithClassCode(ctx, CreateStudentWithClassCodeParams{StudentID: 3, ClassCodeID: 20, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.CreateStudentWithClassCode(ctx, CreateStudentWithClassCodeParams{StudentID: 4, ClassCodeID: 10, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.CreateStudentWithClassCode(ctx, CreateStudentWithClassCodeParams{StudentID: 3, ClassCodeID: 10, UserID: 2})
	})
	if rows, err := queries.CreateStudentWithClassCode(ctx, CreateStudentWithClassCodeParams{StudentID: 3, ClassCodeID: 10, UserID: 1}); err == nil || rows != 0 {
		t.Fatalf("duplicate relation rows = %d, err = %v; want constraint error", rows, err)
	}
	assertMutationRows(t, 1, func() (int64, error) {
		return queries.DeleteStudentClassCodeByStudentID(ctx, DeleteStudentClassCodeByStudentIDParams{StudentID: 3, ClassCodeID: 10, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.DeleteStudentClassCodeByStudentID(ctx, DeleteStudentClassCodeByStudentIDParams{StudentID: 3, ClassCodeID: 10, UserID: 1})
	})
}

func TestDeleteStudentsFromClassOwnershipAndCardinality(t *testing.T) {
	conn, queries := newStudentRelationshipTestDB(t)
	ctx := context.Background()

	assertMutationRows(t, 1, func() (int64, error) {
		return queries.DeleteStudentsOnlyInOneClass(ctx, DeleteStudentsOnlyInOneClassParams{ClassCodeID: 10, UserID: 1})
	})
	assertMutationRows(t, 1, func() (int64, error) {
		return queries.DeleteStudentsWithSeveralClass(ctx, DeleteStudentsWithSeveralClassParams{ClassCodeID: 10, UserID: 1})
	})

	assertStudentExists(t, conn, 1, false)
	assertStudentExists(t, conn, 2, true)
	assertStudentExists(t, conn, 3, true)
	assertStudentExists(t, conn, 4, true)
	assertMembershipExists(t, conn, 2, 10, 1, false)
	assertMembershipExists(t, conn, 2, 11, 1, true)
	assertMembershipExists(t, conn, 4, 20, 2, true)
	assertMembershipExists(t, conn, 4, 10, 1, true)

	assertMutationRows(t, 0, func() (int64, error) {
		return queries.DeleteStudentsOnlyInOneClass(ctx, DeleteStudentsOnlyInOneClassParams{ClassCodeID: 20, UserID: 1})
	})
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.DeleteStudentsWithSeveralClass(ctx, DeleteStudentsWithSeveralClassParams{ClassCodeID: 20, UserID: 1})
	})
}

func newStudentRelationshipTestDB(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE students (id INTEGER PRIMARY KEY, first_name TEXT NOT NULL, last_name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE class_codes (id INTEGER PRIMARY KEY, name TEXT NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE student_class_codes (
			id INTEGER PRIMARY KEY, student_id INTEGER NOT NULL, class_code_id INTEGER NOT NULL, user_id INTEGER NOT NULL,
			UNIQUE(student_id, class_code_id, user_id)
		);
		INSERT INTO students VALUES
			(1, 'Only', 'Class', 1), (2, 'Multi', 'Class', 1), (3, 'No', 'Class', 1), (4, 'Foreign', 'Owner', 2);
		INSERT INTO class_codes VALUES (10, 'Owner A', 1), (11, 'Owner B', 1), (20, 'Foreign', 2);
		INSERT INTO student_class_codes(student_id, class_code_id, user_id) VALUES
			(1, 10, 1), (2, 10, 1), (2, 11, 1), (4, 20, 2), (4, 10, 1);
	`); err != nil {
		t.Fatal(err)
	}
	return conn, New(conn)
}

func assertStudentExists(t *testing.T, conn *sql.DB, id int64, want bool) {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM students WHERE id = ?", id).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if got := count == 1; got != want {
		t.Fatalf("student %d exists = %v, want %v", id, got, want)
	}
}

func assertMembershipExists(t *testing.T, conn *sql.DB, studentID, classID, userID int64, want bool) {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM student_class_codes WHERE student_id = ? AND class_code_id = ? AND user_id = ?", studentID, classID, userID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if got := count == 1; got != want {
		t.Fatalf("membership (%d, %d, %d) exists = %v, want %v", studentID, classID, userID, got, want)
	}
}
