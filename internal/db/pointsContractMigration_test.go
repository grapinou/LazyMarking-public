package db

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

func TestPointsContractMigrationPreservesDataReferencesAndPositiveDomain(t *testing.T) {
	conn := newPrePointsContractMigrationDB(t, false)
	up, down := readPointsContractMigration(t)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("apply 0034 Up: %v", err)
	}

	var pointValue, questionPointID int64
	if err := conn.QueryRow("SELECT point_value FROM points WHERE id=12 AND user_id=1").Scan(&pointValue); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow("SELECT point_id FROM questions WHERE id=50").Scan(&questionPointID); err != nil {
		t.Fatal(err)
	}
	if pointValue != 101 || questionPointID != 12 {
		t.Fatalf("preserved point value=%d question point_id=%d, want 101 and 12", pointValue, questionPointID)
	}

	assertPointValueRejected(t, conn, 0)
	assertPointValueRejected(t, conn, -1)
	if _, err := conn.Exec("INSERT INTO points(point_value,user_id) VALUES(1000,1)"); err != nil {
		t.Fatalf("point value above 100 rejected: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO points(point_value,user_id) VALUES(1,2)"); err != nil {
		t.Fatalf("point value 1 for another owner rejected: %v", err)
	}
	assertPointsForeignKeysValid(t, conn)

	if _, err := conn.Exec(down); err != nil {
		t.Fatalf("apply 0034 Down: %v", err)
	}
	if err := conn.QueryRow("SELECT point_id FROM questions WHERE id=50").Scan(&questionPointID); err != nil || questionPointID != 12 {
		t.Fatalf("question point after Down=%d err=%v, want 12", questionPointID, err)
	}
	if _, err := conn.Exec("INSERT INTO points(point_value,user_id) VALUES(0,1)"); err != nil {
		t.Fatalf("Down did not restore pre-0034 contract: %v", err)
	}
	assertPointsForeignKeysValid(t, conn)
}

func TestPointsContractMigrationRejectsHistoricalInvalidValueBeforeRebuild(t *testing.T) {
	conn := newPrePointsContractMigrationDB(t, true)
	up, _ := readPointsContractMigration(t)
	if _, err := conn.Exec(up); err == nil || !strings.Contains(err.Error(), "migration_0034_point_value_must_be_positive") {
		t.Fatalf("apply 0034 Up error=%v, want named validation constraint", err)
	}

	var pointValue int64
	if err := conn.QueryRow("SELECT point_value FROM points WHERE id=13").Scan(&pointValue); err != nil || pointValue != 0 {
		t.Fatalf("historical invalid point=%d err=%v, want preserved 0", pointValue, err)
	}
	var foreignKeys int
	if err := conn.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("foreign_keys=%d err=%v, want still enabled", foreignKeys, err)
	}
}

func newPrePointsContractMigrationDB(t *testing.T, includeInvalid bool) *sql.DB {
	t.Helper()
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY);
		CREATE TABLE points(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			point_value INTEGER NOT NULL DEFAULT 1,
			user_id INTEGER NOT NULL REFERENCES users(id),
			UNIQUE(point_value,user_id)
		);
		CREATE TABLE questions(
			id INTEGER PRIMARY KEY,
			point_id INTEGER NOT NULL REFERENCES points(id) ON DELETE RESTRICT
		);
		INSERT INTO users VALUES(1),(2);
		INSERT INTO points(id,point_value,user_id) VALUES(7,1,1),(12,101,1);
		INSERT INTO questions(id,point_id) VALUES(50,12);
	`); err != nil {
		t.Fatal(err)
	}
	if includeInvalid {
		if _, err := conn.Exec("INSERT INTO points(id,point_value,user_id) VALUES(13,0,1)"); err != nil {
			t.Fatal(err)
		}
	}
	return conn
}

func readPointsContractMigration(t *testing.T) (string, string) {
	t.Helper()
	migration, err := os.ReadFile("../../db/migrations/0034_require_positive_point_values.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(migration), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("0034 migration has no Down section")
	}
	up := strings.Replace(parts[0], "-- +goose Up", "", 1)
	return up, parts[1]
}

func assertPointValueRejected(t *testing.T, conn *sql.DB, value int64) {
	t.Helper()
	if _, err := conn.Exec("INSERT INTO points(point_value,user_id) VALUES(?,1)", value); err == nil {
		t.Fatalf("point value %d accepted", value)
	}
}

func assertPointsForeignKeysValid(t *testing.T, conn *sql.DB) {
	t.Helper()
	rows, err := conn.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign_key_check reported a violation")
	}
}
