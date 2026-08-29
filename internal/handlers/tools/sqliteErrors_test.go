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
