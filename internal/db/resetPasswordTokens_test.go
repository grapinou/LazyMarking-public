package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestMarkAllResetPasswordTokensUsedForUser(t *testing.T) {
	conn, queries := newResetPasswordTokenTestDB(t)
	ctx := context.Background()
	expiresAt := time.Now().UTC().Add(time.Hour)
	for _, token := range []struct {
		userID int64
		value  string
	}{
		{userID: 1, value: "token-a"},
		{userID: 1, value: "token-b"},
		{userID: 2, value: "token-c"},
	} {
		if err := queries.CreateResetPassword(ctx, CreateResetPasswordParams{
			UserID:    token.userID,
			Token:     token.value,
			ExpiresAt: expiresAt,
		}); err != nil {
			t.Fatalf("CreateResetPassword(%s): %v", token.value, err)
		}
	}

	if err := queries.MarkAllResetPasswordTokensUsedForUser(ctx, 1); err != nil {
		t.Fatalf("MarkAllResetPasswordTokensUsedForUser: %v", err)
	}
	for _, token := range []string{"token-a", "token-b"} {
		if _, err := queries.GetResetPasswordByToken(ctx, token); !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("GetResetPasswordByToken(%s) error = %v, want sql.ErrNoRows", token, err)
		}
		if got := resetTokenUsed(t, conn, token); !got {
			t.Errorf("used(%s) = %v, want true", token, got)
		}
	}
	if _, err := queries.GetResetPasswordByToken(ctx, "token-c"); err != nil {
		t.Errorf("other user's token is not valid: %v", err)
	}
	if got := resetTokenUsed(t, conn, "token-c"); got {
		t.Errorf("used(token-c) = %v, want false", got)
	}

	if err := queries.DeleteExpiredResetTokens(ctx); err != nil {
		t.Fatalf("DeleteExpiredResetTokens: %v", err)
	}
	if got := resetTokenCount(t, conn, "token-a"); got != 0 {
		t.Errorf("token-a count = %d, want 0", got)
	}
	if got := resetTokenCount(t, conn, "token-b"); got != 0 {
		t.Errorf("token-b count = %d, want 0", got)
	}
	if got := resetTokenCount(t, conn, "token-c"); got != 1 {
		t.Errorf("token-c count = %d, want 1", got)
	}
}

func TestPasswordAndResetTokenUpdatesRollbackTogether(t *testing.T) {
	conn, queries := newResetPasswordTokenTestDB(t)
	ctx := context.Background()
	if err := queries.CreateResetPassword(ctx, CreateResetPasswordParams{
		UserID:    1,
		Token:     "token-a",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	qtx := queries.WithTx(tx)
	rows, err := qtx.UpdateUserPassword(ctx, UpdateUserPasswordParams{ID: 1, Hashpassword: "new-hash"})
	if err != nil {
		t.Fatal(err)
	}
	if rows != 1 {
		t.Fatalf("UpdateUserPassword rows = %d, want 1", rows)
	}
	if err := qtx.MarkAllResetPasswordTokensUsedForUser(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	var password string
	if err := conn.QueryRow("SELECT hashpassword FROM users WHERE id = 1").Scan(&password); err != nil {
		t.Fatal(err)
	}
	if password != "old-hash-1" {
		t.Fatalf("password after rollback = %q, want old hash", password)
	}
	if got := resetTokenUsed(t, conn, "token-a"); got {
		t.Fatalf("used(token-a) after rollback = %v, want false", got)
	}
}

func newResetPasswordTokenTestDB(t *testing.T) (*sql.DB, *Queries) {
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
		CREATE TABLE password_resets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);
		INSERT INTO users (id, hashpassword) VALUES
			(1, 'old-hash-1'),
			(2, 'old-hash-2');
	`); err != nil {
		t.Fatalf("create reset token test schema: %v", err)
	}
	return conn, New(conn)
}

func resetTokenUsed(t *testing.T, conn *sql.DB, token string) bool {
	t.Helper()
	var used bool
	if err := conn.QueryRow("SELECT used FROM password_resets WHERE token = ?", token).Scan(&used); err != nil {
		t.Fatalf("read used(%s): %v", token, err)
	}
	return used
}
