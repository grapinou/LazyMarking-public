package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestOwnedReferenceMutationRows(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		table  string
		column string
		update func(*Queries, int64, int64) (int64, error)
		delete func(*Queries, int64, int64) (int64, error)
	}{
		{"subjects", "subjects", "name", func(q *Queries, id, userID int64) (int64, error) {
			return q.UpdateSubject(ctx, UpdateSubjectParams{Name: "updated", ID: id, UserID: userID})
		}, func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteSubject(ctx, DeleteSubjectParams{ID: id, UserID: userID})
		}},
		{"themes", "themes", "name", func(q *Queries, id, userID int64) (int64, error) {
			return q.UpdateTheme(ctx, UpdateThemeParams{Name: "updated", ID: id, UserID: userID})
		}, func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteTheme(ctx, DeleteThemeParams{ID: id, UserID: userID})
		}},
		{"skills", "skills", "name", func(q *Queries, id, userID int64) (int64, error) {
			return q.UpdateSkill(ctx, UpdateSkillParams{Name: "updated", ID: id, UserID: userID})
		}, func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteSkill(ctx, DeleteSkillParams{ID: id, UserID: userID})
		}},
		{"difficulties", "difficulties", "name", func(q *Queries, id, userID int64) (int64, error) {
			return q.UpdateDifficulty(ctx, UpdateDifficultyParams{Name: "updated", ID: id, UserID: userID})
		}, func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteDifficulty(ctx, DeleteDifficultyParams{ID: id, UserID: userID})
		}},
		{"points", "points", "point_value", func(q *Queries, id, userID int64) (int64, error) {
			return q.UpdatePoint(ctx, UpdatePointParams{PointValue: 99, ID: id, UserID: userID})
		}, func(q *Queries, id, userID int64) (int64, error) {
			return q.DeletePoint(ctx, DeletePointParams{ID: id, UserID: userID})
		}},
		{"year_levels", "year_levels", "name", func(q *Queries, id, userID int64) (int64, error) {
			return q.UpdateYearLevel(ctx, UpdateYearLevelParams{Name: "updated", ID: id, UserID: userID})
		}, func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteYearLevel(ctx, DeleteYearLevelParams{ID: id, UserID: userID})
		}},
		{"years", "years", "name", func(q *Queries, id, userID int64) (int64, error) {
			return q.UpdateYear(ctx, UpdateYearParams{Name: "updated", ID: id, UserID: userID})
		}, func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteYear(ctx, DeleteYearParams{ID: id, UserID: userID})
		}},
		{"periods", "periods", "name", func(q *Queries, id, userID int64) (int64, error) {
			return q.UpdatePeriod(ctx, UpdatePeriodParams{Name: "updated", ID: id, UserID: userID})
		}, func(q *Queries, id, userID int64) (int64, error) {
			return q.DeletePeriod(ctx, DeletePeriodParams{ID: id, UserID: userID})
		}},
		{"class_codes", "class_codes", "name", func(q *Queries, id, userID int64) (int64, error) {
			return q.UpdateClassCode(ctx, UpdateClassCodeParams{Name: "updated", ID: id, UserID: userID})
		}, func(q *Queries, id, userID int64) (int64, error) {
			return q.DeleteClassCode(ctx, DeleteClassCodeParams{ID: id, UserID: userID})
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			conn := newReferenceMutationDB(t, tc.table, tc.column)
			queries := New(conn)

			assertMutationRows(t, 1, func() (int64, error) { return tc.update(queries, 1, 1) })
			assertMutationRows(t, 0, func() (int64, error) { return tc.update(queries, 999, 1) })
			assertMutationRows(t, 0, func() (int64, error) { return tc.update(queries, 2, 1) })
			assertReferenceOwnerValue(t, conn, tc.table, tc.column, 2, tc.column == "point_value", "owner-two")

			assertMutationRows(t, 1, func() (int64, error) { return tc.delete(queries, 1, 1) })
			assertMutationRows(t, 0, func() (int64, error) { return tc.delete(queries, 999, 1) })
			assertMutationRows(t, 0, func() (int64, error) { return tc.delete(queries, 2, 1) })
			var count int
			if err := conn.QueryRow("SELECT COUNT(*) FROM " + tc.table + " WHERE id = 2 AND user_id = 2").Scan(&count); err != nil || count != 1 {
				t.Fatalf("foreign-owned row count = %d, err = %v; want 1", count, err)
			}
		})
	}
}

func TestOwnedReferenceMutationConstraintErrorIsNotZeroRows(t *testing.T) {
	conn := newReferenceMutationDB(t, "subjects", "name")
	if _, err := conn.Exec("PRAGMA foreign_keys = ON; CREATE TABLE subject_references (subject_id INTEGER REFERENCES subjects(id));"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec("INSERT INTO subjects (id, user_id, name) VALUES (3, 1, 'duplicate')"); err != nil {
		t.Fatal(err)
	}
	queries := New(conn)
	if rows, err := queries.UpdateSubject(context.Background(), UpdateSubjectParams{Name: "duplicate", ID: 1, UserID: 1}); err == nil || rows != 0 {
		t.Fatalf("constraint update rows = %d, err = %v; want SQL error and zero rows", rows, err)
	}
	assertMutationRows(t, 0, func() (int64, error) {
		return queries.UpdateSubject(context.Background(), UpdateSubjectParams{Name: "duplicate", ID: 2, UserID: 1})
	})
	if _, err := conn.Exec("INSERT INTO subject_references (subject_id) VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	if rows, err := queries.DeleteSubject(context.Background(), DeleteSubjectParams{ID: 1, UserID: 1}); err == nil || rows != 0 {
		t.Fatalf("foreign-key delete rows = %d, err = %v; want SQL error and zero rows", rows, err)
	}
}

func newReferenceMutationDB(t *testing.T, table, column string) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	columnType := "TEXT"
	ownerOne, ownerTwo := "'owner-one'", "'owner-two'"
	if column == "point_value" {
		columnType, ownerOne, ownerTwo = "INTEGER", "1", "2"
	}
	statement := "CREATE TABLE " + table + " (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, " + column + " " + columnType + " NOT NULL, UNIQUE (" + column + ", user_id));" +
		"INSERT INTO " + table + " (id, user_id, " + column + ") VALUES (1, 1, " + ownerOne + "), (2, 2, " + ownerTwo + ");"
	if _, err := conn.Exec(statement); err != nil {
		t.Fatal(err)
	}
	return conn
}

func assertReferenceOwnerValue(t *testing.T, conn *sql.DB, table, column string, id int64, numeric bool, want string) {
	t.Helper()
	if numeric {
		var got int64
		if err := conn.QueryRow("SELECT "+column+" FROM "+table+" WHERE id = ?", id).Scan(&got); err != nil || got != 2 {
			t.Fatalf("foreign-owned value = %d, err = %v; want 2", got, err)
		}
		return
	}
	var got string
	if err := conn.QueryRow("SELECT "+column+" FROM "+table+" WHERE id = ?", id).Scan(&got); err != nil || got != want {
		t.Fatalf("foreign-owned value = %q, err = %v; want %q", got, err, want)
	}
}
