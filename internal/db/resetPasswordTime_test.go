package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestResetPasswordTokenTimeSemantics(t *testing.T) {
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY
		);
		CREATE TABLE password_resets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			token TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			used BOOLEAN DEFAULT FALSE,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);
		INSERT INTO users (id) VALUES (1);
	`); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	ctx := context.Background()
	queries := New(conn)
	nowUTC := time.Now().UTC()
	expiredPlusTwo := nowUTC.Add(-5 * time.Minute).In(time.FixedZone("UTC+2", 2*60*60))
	validMinusTwo := nowUTC.Add(5 * time.Minute).In(time.FixedZone("UTC-2", -2*60*60))
	usedFuture := nowUTC.Add(time.Hour)

	for _, token := range []struct {
		value     string
		expiresAt time.Time
	}{
		{value: "expired-plus-two", expiresAt: expiredPlusTwo},
		{value: "valid-minus-two", expiresAt: validMinusTwo},
		{value: "used-future", expiresAt: usedFuture},
	} {
		if err := queries.CreateResetPassword(ctx, CreateResetPasswordParams{
			UserID:    1,
			Token:     token.value,
			ExpiresAt: token.expiresAt,
		}); err != nil {
			t.Fatalf("CreateResetPassword(%s): %v", token.value, err)
		}
	}

	var storedExpired, storedValid string
	if err := conn.QueryRow("SELECT expires_at FROM password_resets WHERE token = ?", "expired-plus-two").Scan(&storedExpired); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow("SELECT expires_at FROM password_resets WHERE token = ?", "valid-minus-two").Scan(&storedValid); err != nil {
		t.Fatal(err)
	}
	t.Logf("expired instant=%s stored=%q", expiredPlusTwo.Format(time.RFC3339Nano), storedExpired)
	t.Logf("valid instant=%s stored=%q", validMinusTwo.Format(time.RFC3339Nano), storedValid)

	if _, err := queries.GetResetPasswordByToken(ctx, "expired-plus-two"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expired +02 token lookup error = %v, want sql.ErrNoRows", err)
	}
	if _, err := queries.GetResetPasswordByToken(ctx, "valid-minus-two"); err != nil {
		t.Errorf("valid -02 token lookup error = %v, want nil", err)
	}
	if err := queries.MarkResetPasswordTokenUsed(ctx, "used-future"); err != nil {
		t.Fatalf("MarkResetPasswordTokenUsed: %v", err)
	}

	if err := queries.DeleteExpiredResetTokens(ctx); err != nil {
		t.Fatalf("DeleteExpiredResetTokens: %v", err)
	}
	if got := resetTokenCount(t, conn, "expired-plus-two"); got != 0 {
		t.Errorf("expired +02 token count after cleanup = %d, want 0", got)
	}
	if got := resetTokenCount(t, conn, "valid-minus-two"); got != 1 {
		t.Errorf("valid -02 token count after cleanup = %d, want 1", got)
	}
	if got := resetTokenCount(t, conn, "used-future"); got != 0 {
		t.Errorf("used future token count after cleanup = %d, want 0", got)
	}

	if !strings.Contains(getResetPasswordByToken, "unixepoch(expires_at) > unixepoch('now')") {
		t.Error("token validity query no longer uses strict > expiration boundary")
	}
	if !strings.Contains(deleteExpiredResetTokens, "unixepoch(expires_at) <= unixepoch('now')") {
		t.Error("token cleanup query no longer uses <= expiration boundary")
	}
}

func resetTokenCount(t *testing.T, conn *sql.DB, token string) int {
	t.Helper()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM password_resets WHERE token = ?", token).Scan(&count); err != nil {
		t.Fatalf("count token %s: %v", token, err)
	}
	return count
}
