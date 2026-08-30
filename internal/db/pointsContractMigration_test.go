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
	assertQuestionOwnershipTriggers(t, conn, 60)

	assertPointValueRejected(t, conn, 0)
	assertPointValueRejected(t, conn, -1)
	if _, err := conn.Exec("INSERT INTO points(point_value,user_id) VALUES(1000,1)"); err != nil {
		t.Fatalf("point value above 100 rejected: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO points(point_value,user_id) VALUES(1,2)"); err != nil {
		t.Fatalf("point value 1 for another owner rejected: %v", err)
	}
	assertSQLRejected(t, conn, "duplicate point value for owner", "INSERT INTO points(point_value,user_id) VALUES(1,1)")
	assertSQLRejected(t, conn, "point with missing owner", "INSERT INTO points(point_value,user_id) VALUES(2,999)")
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
	assertQuestionOwnershipTriggers(t, conn, 70)
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
	assertQuestionOwnershipTriggers(t, conn, 80)
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
		CREATE TABLE subjects(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE themes(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE year_levels(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE skills(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE difficulties(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id));
		CREATE TABLE points(
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			point_value INTEGER NOT NULL DEFAULT 1,
			user_id INTEGER NOT NULL REFERENCES users(id),
			UNIQUE(point_value,user_id)
		);
		CREATE TABLE questions(
			id INTEGER PRIMARY KEY,
			subject_id INTEGER NOT NULL REFERENCES subjects(id),
			theme_id INTEGER NOT NULL REFERENCES themes(id),
			year_level_id INTEGER NOT NULL REFERENCES year_levels(id),
			skill_id INTEGER NOT NULL REFERENCES skills(id),
			difficulty_id INTEGER NOT NULL REFERENCES difficulties(id),
			point_id INTEGER NOT NULL REFERENCES points(id) ON DELETE RESTRICT,
			user_id INTEGER NOT NULL REFERENCES users(id)
		);
		CREATE TRIGGER questions_owner_insert BEFORE INSERT ON questions
		WHEN NOT EXISTS (SELECT 1 FROM subjects WHERE id = NEW.subject_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM themes WHERE id = NEW.theme_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM year_levels WHERE id = NEW.year_level_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM skills WHERE id = NEW.skill_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM difficulties WHERE id = NEW.difficulty_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM points WHERE id = NEW.point_id AND user_id = NEW.user_id)
		BEGIN SELECT RAISE(ABORT, 'question resources must belong to user'); END;
		CREATE TRIGGER questions_owner_update BEFORE UPDATE OF subject_id, theme_id, year_level_id, skill_id, difficulty_id, point_id, user_id ON questions
		WHEN NOT EXISTS (SELECT 1 FROM subjects WHERE id = NEW.subject_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM themes WHERE id = NEW.theme_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM year_levels WHERE id = NEW.year_level_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM skills WHERE id = NEW.skill_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM difficulties WHERE id = NEW.difficulty_id AND user_id = NEW.user_id)
		  OR NOT EXISTS (SELECT 1 FROM points WHERE id = NEW.point_id AND user_id = NEW.user_id)
		BEGIN SELECT RAISE(ABORT, 'question resources must belong to user'); END;
		INSERT INTO users VALUES(1),(2);
		INSERT INTO subjects VALUES(1,1),(2,2);
		INSERT INTO themes VALUES(1,1),(2,2);
		INSERT INTO year_levels VALUES(1,1),(2,2);
		INSERT INTO skills VALUES(1,1),(2,2);
		INSERT INTO difficulties VALUES(1,1),(2,2);
		INSERT INTO points(id,point_value,user_id) VALUES(7,1,1),(12,101,1),(20,2,2);
		INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,user_id)
		VALUES(50,1,1,1,1,1,12,1);
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

func assertQuestionOwnershipTriggers(t *testing.T, conn *sql.DB, questionID int64) {
	t.Helper()
	var count int
	if err := conn.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name IN ('questions_owner_insert','questions_owner_update')`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("question ownership trigger count=%d err=%v, want 2", count, err)
	}
	if _, err := conn.Exec(`INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,user_id) VALUES(?,1,1,1,1,1,7,1)`, questionID); err != nil {
		t.Fatalf("owned question insert rejected: %v", err)
	}
	assertSQLRejected(t, conn, "foreign subject ownership on insert", `INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,user_id) VALUES(998,2,1,1,1,1,7,1)`)
	assertSQLRejected(t, conn, "foreign point ownership on insert", `INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,user_id) VALUES(999,1,1,1,1,1,20,1)`)
	assertSQLRejected(t, conn, "foreign subject ownership on update", `UPDATE questions SET subject_id=2 WHERE id=50`)
	assertSQLRejected(t, conn, "foreign point ownership on update", `UPDATE questions SET point_id=20 WHERE id=50`)
}
