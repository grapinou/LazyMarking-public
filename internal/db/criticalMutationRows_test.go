package db

import (
	"context"
	"database/sql"
	"testing"
)

func TestCriticalMutationRowsAffected(t *testing.T) {
	t.Run("DeleteImage", func(t *testing.T) {
		_, queries := newCriticalMutationTestDB(t)
		ctx := context.Background()
		params := DeleteImageParams{QuestionID: 10, UserID: 1}

		assertMutationRows(t, 1, func() (int64, error) { return queries.DeleteImage(ctx, params) })
		assertMutationRows(t, 0, func() (int64, error) { return queries.DeleteImage(ctx, params) })
	})

	t.Run("DeleteAltImage", func(t *testing.T) {
		_, queries := newCriticalMutationTestDB(t)
		ctx := context.Background()
		params := DeleteAltImageParams{AltQuestionID: 20, QuestionID: 10, UserID: 1}

		assertMutationRows(t, 1, func() (int64, error) { return queries.DeleteAltImage(ctx, params) })
		assertMutationRows(t, 0, func() (int64, error) { return queries.DeleteAltImage(ctx, params) })
	})

	t.Run("DeleteQuestionCountsOnlyParent", func(t *testing.T) {
		conn, queries := newCriticalMutationTestDB(t)
		ctx := context.Background()

		assertMutationRows(t, 1, func() (int64, error) {
			return queries.DeleteQuestion(ctx, DeleteQuestionParams{ID: 10, UserID: 1})
		})
		assertTableCount(t, conn, "images", 0)
		assertTableCount(t, conn, "alt_questions", 0)
		assertTableCount(t, conn, "alt_images", 0)
		assertMutationRows(t, 0, func() (int64, error) {
			return queries.DeleteQuestion(ctx, DeleteQuestionParams{ID: 10, UserID: 1})
		})
	})

	t.Run("DeleteAltQuestionCountsOnlyParent", func(t *testing.T) {
		conn, queries := newCriticalMutationTestDB(t)
		ctx := context.Background()

		assertMutationRows(t, 1, func() (int64, error) {
			return queries.DeleteAltQuestion(ctx, DeleteAltQuestionParams{ID: 20, QuestionID: 10, UserID: 1})
		})
		assertTableCount(t, conn, "alt_images", 0)
		assertMutationRows(t, 0, func() (int64, error) {
			return queries.DeleteAltQuestion(ctx, DeleteAltQuestionParams{ID: 20, QuestionID: 10, UserID: 1})
		})
	})

	t.Run("UpdateUserPassword", func(t *testing.T) {
		_, queries := newCriticalMutationTestDB(t)
		ctx := context.Background()

		assertMutationRows(t, 1, func() (int64, error) {
			return queries.UpdateUserPassword(ctx, UpdateUserPasswordParams{ID: 1, Hashpassword: "new-hash"})
		})
		assertMutationRows(t, 0, func() (int64, error) {
			return queries.UpdateUserPassword(ctx, UpdateUserPasswordParams{ID: 999, Hashpassword: "unused"})
		})
	})
}

func assertMutationRows(t *testing.T, want int64, mutate func() (int64, error)) {
	t.Helper()
	rows, err := mutate()
	if err != nil {
		t.Fatalf("mutation error: %v", err)
	}
	if rows != want {
		t.Fatalf("rows affected = %d, want %d", rows, want)
	}
}

func assertTableCount(t *testing.T, conn *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}

func newCriticalMutationTestDB(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			hashpassword TEXT NOT NULL
		);
		CREATE TABLE questions (
			id INTEGER PRIMARY KEY,
			user_id INTEGER NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
		CREATE TABLE images (
			id INTEGER PRIMARY KEY,
			question_id INTEGER NOT NULL UNIQUE,
			user_id INTEGER NOT NULL,
			FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
		CREATE TABLE alt_questions (
			id INTEGER PRIMARY KEY,
			question_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
		CREATE TABLE alt_images (
			id INTEGER PRIMARY KEY,
			alt_question_id INTEGER NOT NULL UNIQUE,
			user_id INTEGER NOT NULL,
			FOREIGN KEY (alt_question_id) REFERENCES alt_questions(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		INSERT INTO users (id, hashpassword) VALUES (1, 'old-hash');
		INSERT INTO questions (id, user_id) VALUES (10, 1);
		INSERT INTO images (id, question_id, user_id) VALUES (30, 10, 1);
		INSERT INTO alt_questions (id, question_id, user_id) VALUES (20, 10, 1);
		INSERT INTO alt_images (id, alt_question_id, user_id) VALUES (40, 20, 1);
	`); err != nil {
		t.Fatalf("create critical mutation test schema: %v", err)
	}

	return conn, New(conn)
}
