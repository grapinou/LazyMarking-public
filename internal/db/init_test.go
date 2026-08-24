package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestInitDBEnablesForeignKeysOnEveryPooledConnection(t *testing.T) {
	conn, err := InitDB(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(8)

	connections := make([]interface{ Close() error }, 0, 8)
	for i := 0; i < 8; i++ {
		connection, err := conn.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		connections = append(connections, connection)
		var enabled int
		if err := connection.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&enabled); err != nil {
			t.Fatal(err)
		}
		if enabled != 1 {
			t.Fatalf("connection %d has foreign_keys=%d, want 1", i, enabled)
		}
	}
	for _, connection := range connections {
		if err := connection.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSQLiteDSNPreservesExistingQuery(t *testing.T) {
	got := sqliteDSN("file:test.db?mode=rwc")
	want := "file:test.db?mode=rwc&_foreign_keys=on"
	if got != want {
		t.Fatalf("sqliteDSN() = %q, want %q", got, want)
	}
}
