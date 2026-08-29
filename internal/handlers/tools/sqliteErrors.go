package tools

import (
	"errors"

	"github.com/mattn/go-sqlite3"
)

// IsSQLiteForeignKeyConstraint reports whether err is a SQLite foreign-key
// constraint violation, including when the driver error is wrapped. SQLite
// reports an explicit ON DELETE RESTRICT action as ErrConstraintTrigger rather
// than ErrConstraintForeignKey; both codes therefore represent FK enforcement
// for the schema used by LazyMarking.
func IsSQLiteForeignKeyConstraint(err error) bool {
	var sqliteError sqlite3.Error
	if !errors.As(err, &sqliteError) {
		return false
	}
	return sqliteError.ExtendedCode == sqlite3.ErrConstraintForeignKey ||
		sqliteError.ExtendedCode == sqlite3.ErrConstraintTrigger
}
