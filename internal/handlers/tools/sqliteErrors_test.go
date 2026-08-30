package tools

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/mattn/go-sqlite3"
)

func TestIsSQLiteForeignKeyConstraint(t *testing.T) {
	foreignKeyError := sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintForeignKey}
	restrictError := sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintTrigger}
	otherConstraint := sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintUnique}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "foreign key", err: foreignKeyError, want: true},
		{name: "wrapped foreign key", err: fmt.Errorf("delete: %w", foreignKeyError), want: true},
		{name: "explicit restrict", err: restrictError, want: true},
		{name: "other SQLite constraint", err: otherConstraint, want: false},
		{name: "non SQLite error", err: fmt.Errorf("database unavailable"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSQLiteForeignKeyConstraint(tc.err); got != tc.want {
				t.Fatalf("IsSQLiteForeignKeyConstraint(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

func TestIsSQLiteForeignKeyConstraintRecognizesDriverError(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`
		CREATE TABLE parent(id INTEGER PRIMARY KEY);
		CREATE TABLE child(parent_id INTEGER REFERENCES parent(id) ON DELETE RESTRICT);
		INSERT INTO parent VALUES(1);
		INSERT INTO child VALUES(1);
	`); err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec("DELETE FROM parent WHERE id=1")
	if !IsSQLiteForeignKeyConstraint(err) {
		t.Fatalf("error type=%T value=%#v is not recognized as a foreign-key constraint", err, err)
	}
}

func TestIsSQLiteUniqueConstraint(t *testing.T) {
	uniqueError := sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintUnique}
	checkError := sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintCheck}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unique", err: uniqueError, want: true},
		{name: "wrapped unique", err: fmt.Errorf("create Exam: %w", uniqueError), want: true},
		{name: "check", err: checkError, want: false},
		{name: "non SQLite error", err: fmt.Errorf("database unavailable"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSQLiteUniqueConstraint(tc.err); got != tc.want {
				t.Fatalf("IsSQLiteUniqueConstraint(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

func TestIsSQLiteCheckConstraint(t *testing.T) {
	checkError := sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintCheck}
	uniqueError := sqlite3.Error{Code: sqlite3.ErrConstraint, ExtendedCode: sqlite3.ErrConstraintUnique}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "check", err: checkError, want: true},
		{name: "wrapped check", err: fmt.Errorf("update student: %w", checkError), want: true},
		{name: "unique", err: uniqueError, want: false},
		{name: "non SQLite error", err: fmt.Errorf("database unavailable"), want: false},
		{name: "nil", err: nil, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSQLiteCheckConstraint(tc.err); got != tc.want {
				t.Fatalf("IsSQLiteCheckConstraint(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

func TestSQLiteUniqueAndCheckHelpersRecognizeDriverErrors(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`
		CREATE TABLE classified(value TEXT NOT NULL CHECK(length(trim(value)) > 0), UNIQUE(value));
		INSERT INTO classified VALUES('known');
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec("INSERT INTO classified VALUES('known')"); !IsSQLiteUniqueConstraint(err) {
		t.Fatalf("driver UNIQUE error type=%T value=%#v was not classified", err, err)
	}
	if _, err := conn.Exec("INSERT INTO classified VALUES('   ')"); !IsSQLiteCheckConstraint(err) {
		t.Fatalf("driver CHECK error type=%T value=%#v was not classified", err, err)
	}
}
