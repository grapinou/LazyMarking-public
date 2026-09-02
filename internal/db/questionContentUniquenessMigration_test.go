package db

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

func TestQuestionContentUniquenessMigrationContractAndGraph(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec(questionContentMigrationFixture); err != nil {
		t.Fatalf("prepare pre-migration graph: %v", err)
	}
	before := graphCounts(t, conn)
	if _, err := conn.Exec(readQuestionContentMigrationUp(t)); err != nil {
		t.Fatalf("apply 0041: %v", err)
	}
	if got := graphCounts(t, conn); got != before {
		t.Fatalf("graph counts after migration = %v, want %v", got, before)
	}

	mustExec(t, conn, "INSERT INTO questions VALUES(11,1,1,1,1,1,1,'same main',1)")
	mustExec(t, conn, "INSERT INTO questions VALUES(12,1,1,1,1,1,1,'same main',1)")
	mustExec(t, conn, "INSERT INTO questions VALUES(20,2,2,2,2,2,2,'same main',2)")
	mustExec(t, conn, "INSERT INTO alt_questions VALUES(31,10,'same alt',1)")
	mustExec(t, conn, "INSERT INTO alt_questions VALUES(32,11,'same alt',1)")
	mustExec(t, conn, "INSERT INTO alt_questions VALUES(33,10,'same alt',1)")
	mustExec(t, conn, "INSERT INTO alt_questions VALUES(40,20,'same alt',2)")

	assertSQLRejected(t, conn, "empty main", "INSERT INTO questions VALUES(50,1,1,1,1,1,1,'',1)")
	assertSQLRejected(t, conn, "whitespace main", "INSERT INTO questions VALUES(51,1,1,1,1,1,1,'   ',1)")
	assertSQLRejected(t, conn, "empty alternative", "INSERT INTO alt_questions VALUES(50,10,'',1)")
	assertSQLRejected(t, conn, "whitespace alternative", "INSERT INTO alt_questions VALUES(51,10,'   ',1)")
	assertSQLRejected(t, conn, "foreign main resources", "INSERT INTO questions VALUES(60,2,2,2,2,2,2,'forged',1)")
	assertSQLRejected(t, conn, "foreign alternative parent", "INSERT INTO alt_questions VALUES(60,20,'forged',1)")

	for _, trigger := range []string{"questions_owner_insert", "questions_owner_update", "alt_questions_owner_insert", "alt_questions_owner_update"} {
		var count int
		if err := conn.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name=?", trigger).Scan(&count); err != nil || count != 1 {
			t.Fatalf("trigger %s count=%d err=%v", trigger, count, err)
		}
	}
	assertSQLRejected(t, conn, "QCM restrict", "DELETE FROM questions WHERE id=10")
	mustExec(t, conn, "DELETE FROM questions WHERE id=11")
	for table, want := range map[string]int{"answers": 1, "images": 1, "alt_questions": 4, "alt_answers": 1, "alt_images": 1, "qcm_questions": 1} {
		var got int
		if err := conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil || got != want {
			t.Fatalf("%s count=%d want=%d err=%v", table, got, want, err)
		}
	}
	assertDatabaseChecks(t, conn)
}

func TestQuestionContentUniquenessMigrationDownIsSafelyIrreversible(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(readQuestionContentMigrationDown(t)); err == nil {
		t.Fatal("0041 down unexpectedly succeeded")
	}
}

func graphCounts(t *testing.T, conn *sql.DB) string {
	t.Helper()
	var out strings.Builder
	for _, table := range []string{"questions", "answers", "images", "alt_questions", "alt_answers", "alt_images", "qcm_questions", "student_exam_content"} {
		var count int
		if err := conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		out.WriteString(table)
		out.WriteByte('=')
		out.WriteByte(byte('0' + count))
		out.WriteByte(';')
	}
	return out.String()
}

func assertDatabaseChecks(t *testing.T, conn *sql.DB) {
	t.Helper()
	rows, err := conn.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign_key_check reported a violation")
	}
	var integrity string
	if err := conn.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity_check=%q err=%v", integrity, err)
	}
}

func mustExec(t *testing.T, conn *sql.DB, statement string) {
	t.Helper()
	if _, err := conn.Exec(statement); err != nil {
		t.Fatalf("%s: %v", statement, err)
	}
}

func readQuestionContentMigrationUp(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../db/migrations/0041_remove_question_content_uniqueness.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(b), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("0041 has no Down")
	}
	return strings.Replace(strings.Replace(parts[0], "-- +goose NO TRANSACTION", "", 1), "-- +goose Up", "", 1)
}
func readQuestionContentMigrationDown(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../../db/migrations/0041_remove_question_content_uniqueness.sql")
	if err != nil {
		t.Fatal(err)
	}
	return strings.SplitN(string(b), "-- +goose Down", 2)[1]
}

const questionContentMigrationFixture = `
PRAGMA foreign_keys=ON;
CREATE TABLE users(id INTEGER PRIMARY KEY);
CREATE TABLE subjects(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE themes(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE year_levels(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE skills(id INTEGER PRIMARY KEY,user_id INTEGER);
CREATE TABLE difficulties(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE points(id INTEGER PRIMARY KEY,user_id INTEGER);
INSERT INTO users VALUES(1),(2); INSERT INTO subjects VALUES(1,1),(2,2); INSERT INTO themes VALUES(1,1),(2,2);
INSERT INTO year_levels VALUES(1,1),(2,2); INSERT INTO skills VALUES(1,1),(2,2); INSERT INTO difficulties VALUES(1,1),(2,2); INSERT INTO points VALUES(1,1),(2,2);
CREATE TABLE questions(id INTEGER PRIMARY KEY AUTOINCREMENT,subject_id INTEGER NOT NULL,theme_id INTEGER NOT NULL,year_level_id INTEGER NOT NULL,skill_id INTEGER NOT NULL,difficulty_id INTEGER NOT NULL,point_id INTEGER NOT NULL,content TEXT NOT NULL CHECK(length(trim(content))>0),user_id INTEGER NOT NULL,FOREIGN KEY(subject_id) REFERENCES subjects(id) ON DELETE RESTRICT,FOREIGN KEY(theme_id) REFERENCES themes(id) ON DELETE RESTRICT,FOREIGN KEY(year_level_id) REFERENCES year_levels(id) ON DELETE RESTRICT,FOREIGN KEY(skill_id) REFERENCES skills(id) ON DELETE RESTRICT,FOREIGN KEY(difficulty_id) REFERENCES difficulties(id) ON DELETE RESTRICT,FOREIGN KEY(point_id) REFERENCES points(id) ON DELETE RESTRICT,FOREIGN KEY(user_id) REFERENCES users(id),UNIQUE(content,user_id));
CREATE TABLE alt_questions(id INTEGER PRIMARY KEY AUTOINCREMENT,question_id INTEGER NOT NULL,content TEXT NOT NULL CHECK(length(trim(content))>0),user_id INTEGER NOT NULL,FOREIGN KEY(question_id) REFERENCES questions(id) ON DELETE CASCADE,FOREIGN KEY(user_id) REFERENCES users(id),UNIQUE(content,user_id));
CREATE TABLE answers(id INTEGER PRIMARY KEY,question_id INTEGER REFERENCES questions(id) ON DELETE CASCADE,user_id INTEGER); CREATE TABLE images(id INTEGER PRIMARY KEY,question_id INTEGER REFERENCES questions(id) ON DELETE CASCADE,user_id INTEGER); CREATE TABLE alt_answers(id INTEGER PRIMARY KEY,alt_question_id INTEGER REFERENCES alt_questions(id) ON DELETE CASCADE,user_id INTEGER); CREATE TABLE alt_images(id INTEGER PRIMARY KEY,alt_question_id INTEGER REFERENCES alt_questions(id) ON DELETE CASCADE,user_id INTEGER); CREATE TABLE qcm(id INTEGER PRIMARY KEY,user_id INTEGER); CREATE TABLE qcm_questions(id INTEGER PRIMARY KEY,qcm_id INTEGER,question_id INTEGER REFERENCES questions(id) ON DELETE RESTRICT,user_id INTEGER); CREATE TABLE student_exam_content(id INTEGER PRIMARY KEY,content TEXT);
CREATE TRIGGER questions_owner_insert BEFORE INSERT ON questions WHEN NOT EXISTS(SELECT 1 FROM subjects WHERE id=NEW.subject_id AND user_id=NEW.user_id) OR NOT EXISTS(SELECT 1 FROM themes WHERE id=NEW.theme_id AND user_id=NEW.user_id) OR NOT EXISTS(SELECT 1 FROM year_levels WHERE id=NEW.year_level_id AND user_id=NEW.user_id) OR NOT EXISTS(SELECT 1 FROM skills WHERE id=NEW.skill_id AND user_id=NEW.user_id) OR NOT EXISTS(SELECT 1 FROM difficulties WHERE id=NEW.difficulty_id AND user_id=NEW.user_id) OR NOT EXISTS(SELECT 1 FROM points WHERE id=NEW.point_id AND user_id=NEW.user_id) BEGIN SELECT RAISE(ABORT,'question resources must belong to user'); END;
CREATE TRIGGER questions_owner_update BEFORE UPDATE OF subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,user_id ON questions BEGIN SELECT 1; END;
CREATE TRIGGER alt_questions_owner_insert BEFORE INSERT ON alt_questions WHEN NOT EXISTS(SELECT 1 FROM questions WHERE id=NEW.question_id AND user_id=NEW.user_id) BEGIN SELECT RAISE(ABORT,'question must belong to user'); END;
CREATE TRIGGER alt_questions_owner_update BEFORE UPDATE OF question_id,user_id ON alt_questions BEGIN SELECT 1; END;
CREATE TRIGGER answers_owner_insert BEFORE INSERT ON answers BEGIN SELECT 1; END; CREATE TRIGGER answers_owner_update BEFORE UPDATE ON answers BEGIN SELECT 1; END;
CREATE TRIGGER images_owner_insert BEFORE INSERT ON images BEGIN SELECT 1; END; CREATE TRIGGER images_owner_update BEFORE UPDATE ON images BEGIN SELECT 1; END;
CREATE TRIGGER alt_answers_owner_insert BEFORE INSERT ON alt_answers BEGIN SELECT 1; END; CREATE TRIGGER alt_answers_owner_update BEFORE UPDATE ON alt_answers BEGIN SELECT 1; END;
CREATE TRIGGER alt_images_owner_insert BEFORE INSERT ON alt_images BEGIN SELECT 1; END; CREATE TRIGGER alt_images_owner_update BEFORE UPDATE ON alt_images BEGIN SELECT 1; END;
CREATE TRIGGER qcm_questions_owner_insert BEFORE INSERT ON qcm_questions BEGIN SELECT 1; END; CREATE TRIGGER qcm_questions_owner_update BEFORE UPDATE ON qcm_questions BEGIN SELECT 1; END;
INSERT INTO questions VALUES(10,1,1,1,1,1,1,'existing',1); INSERT INTO alt_questions VALUES(30,10,'existing alt',1); INSERT INTO answers VALUES(1,10,1); INSERT INTO images VALUES(1,10,1); INSERT INTO alt_answers VALUES(1,30,1); INSERT INTO alt_images VALUES(1,30,1); INSERT INTO qcm VALUES(1,1); INSERT INTO qcm_questions VALUES(1,1,10,1); INSERT INTO student_exam_content VALUES(1,'snapshot');`
