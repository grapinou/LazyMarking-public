package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	migrationfiles "github.com/grapinou/LazyMarking/db/migrations"
)

func TestOpenMigratedDBFreshAndAlreadyCurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fresh.db")
	conn, report, err := OpenMigratedDB(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if report.FromVersion != 0 || report.ToVersion != 46 || len(report.Applied) != 46 {
		t.Fatalf("fresh migration report = %+v", report)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	conn, report, err = OpenMigratedDB(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if report.FromVersion != 46 || report.ToVersion != 46 || len(report.Applied) != 0 {
		t.Fatalf("current migration report = %+v", report)
	}
	assertMigrationVersion(t, conn, 46)
}

func TestOpenMigratedDBUpgradesVersion44AndPreservesData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "version-44.db")
	conn, err := InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := newMigrationProvider(conn, migrationfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(context.Background(), 44); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`
		INSERT INTO users(id, username, email, hashpassword)
		VALUES(1, 'migration-test', 'migration@example.test', 'hash');
		INSERT INTO subjects(id, name, user_id) VALUES(1, 'Sciences', 1);
		INSERT INTO themes(id, name, user_id) VALUES(1, 'Matière', 1);
		INSERT INTO year_levels(id, name, user_id) VALUES(1, '6e', 1);
		INSERT INTO skills(id, name, user_id) VALUES(1, 'Analyser', 1);
		INSERT INTO difficulties(id, name, user_id) VALUES(1, 'Simple', 1);
		INSERT INTO points(id, point_value, user_id) VALUES(1, 1, 1);
		INSERT INTO questions(
			id, subject_id, theme_id, year_level_id, skill_id,
			difficulty_id, point_id, content, user_id
		) VALUES(1, 1, 1, 1, 1, 1, 1, 'Donnée existante', 1);
	`); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	conn, report, err := OpenMigratedDB(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if report.FromVersion != 44 || report.ToVersion != 46 || fmt.Sprint(report.Applied) != "[45 46]" {
		t.Fatalf("migration report = %+v", report)
	}
	var content, instruction string
	if err := conn.QueryRow(`SELECT content, instruction FROM questions WHERE id = 1`).Scan(&content, &instruction); err != nil {
		t.Fatal(err)
	}
	if content != "Donnée existante" || instruction != "" {
		t.Fatalf("migrated question = content %q, instruction %q", content, instruction)
	}
}

func TestOpenMigratedDBReturnsNoConnectionWhenMigrationFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failing.db")
	migrations := fstest.MapFS{
		"0001_preserve.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE preserved(value TEXT NOT NULL);
INSERT INTO preserved VALUES('still here');
-- +goose Down
DROP TABLE preserved;
`)},
		"0002_fail.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE should_rollback(value TEXT);
THIS IS NOT VALID SQL;
-- +goose Down
DROP TABLE should_rollback;
`)},
	}
	conn, report, err := openMigratedDB(context.Background(), path, migrations)
	if err == nil {
		t.Fatal("migration unexpectedly succeeded")
	}
	if conn != nil {
		conn.Close()
		t.Fatal("failed startup returned a usable database connection")
	}
	if report.FromVersion != 0 || report.ToVersion != 2 {
		t.Fatalf("failure report = %+v", report)
	}

	inspection, err := InitDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer inspection.Close()
	assertMigrationVersion(t, inspection, 1)
	var value string
	if err := inspection.QueryRow(`SELECT value FROM preserved`).Scan(&value); err != nil || value != "still here" {
		t.Fatalf("preserved data = %q, err = %v", value, err)
	}
	var rolledBackTable int
	if err := inspection.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = 'should_rollback'
	`).Scan(&rolledBackTable); err != nil {
		t.Fatal(err)
	}
	if rolledBackTable != 0 {
		t.Fatal("failed migration was not rolled back")
	}
}

func TestOpenMigratedDBRejectsSchemaNewerThanBinary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.db")
	conn, _, err := OpenMigratedDB(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`
		INSERT INTO goose_db_version(version_id, is_applied)
		VALUES(47, 1)
	`); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	conn, report, err := OpenMigratedDB(context.Background(), path)
	if err == nil {
		if conn != nil {
			conn.Close()
		}
		t.Fatal("newer schema unexpectedly accepted")
	}
	if conn != nil {
		conn.Close()
		t.Fatal("newer schema returned a usable database connection")
	}
	if report.FromVersion != 47 || report.ToVersion != 46 {
		t.Fatalf("newer schema report = %+v", report)
	}
}

func TestOpenMigratedDBRuntimeCopies(t *testing.T) {
	for _, environment := range []string{"smoke", "real", "2026-2027"} {
		t.Run(environment, func(t *testing.T) {
			source, err := filepath.Abs(filepath.Join("..", "..", "testdata", environment, "app.db"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(source); os.IsNotExist(err) {
				t.Skip("local runtime database is not available")
			} else if err != nil {
				t.Fatal(err)
			}
			sourceBefore := fileDigest(t, source)
			copyPath := filepath.Join(t.TempDir(), "app.db")
			copyFile(t, source, copyPath)

			conn, err := InitDB(copyPath)
			if err != nil {
				t.Fatal(err)
			}
			provider, err := newMigrationProvider(conn, migrationfiles.Files)
			if err != nil {
				t.Fatal(err)
			}
			version, err := provider.GetDBVersion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			before := stableRuntimeDataDigest(t, conn)
			sharingSQL := `SELECT COALESCE(group_concat(question_id, ','), '') FROM (SELECT question_id FROM question_shares ORDER BY question_id)`
			var sharingBefore string
			if version >= 46 {
				if err := conn.QueryRow(sharingSQL).Scan(&sharingBefore); err != nil {
					t.Fatal(err)
				}
			}
			if err := conn.Close(); err != nil {
				t.Fatal(err)
			}

			conn, report, err := OpenMigratedDB(context.Background(), copyPath)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if report.FromVersion != version || report.ToVersion != 46 {
				t.Fatalf("migration report = %+v, source copy version = %d", report, version)
			}
			if after := stableRuntimeDataDigest(t, conn); after != before {
				t.Fatalf("existing runtime data changed: before %x, after %x", before, after)
			}
			assertMigrationVersion(t, conn, 46)
			var sharingAfter string
			if err := conn.QueryRow(sharingSQL).Scan(&sharingAfter); err != nil || sharingAfter != sharingBefore {
				t.Fatalf("sharing changed on startup: before=%q after=%q err=%v", sharingBefore, sharingAfter, err)
			}
			t.Logf("runtime copy %s: %d -> %d; existing sharing preserved", environment, version, report.ToVersion)
			assertSQLiteIntegrity(t, conn)
			if sourceAfter := fileDigest(t, source); sourceAfter != sourceBefore {
				t.Fatal("source runtime database was modified")
			}
		})
	}
}

func assertMigrationVersion(t *testing.T, conn *sql.DB, want int64) {
	t.Helper()
	var got int64
	if err := conn.QueryRow(`
		SELECT COALESCE(MAX(version_id), 0)
		FROM goose_db_version
		WHERE is_applied = 1
	`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("database version = %d, want %d", got, want)
	}
}

func stableRuntimeDataDigest(t *testing.T, conn *sql.DB) [sha256.Size]byte {
	t.Helper()
	hash := sha256.New()
	for _, query := range []string{
		`SELECT id, content, user_id FROM questions ORDER BY id`,
		`SELECT student_exam_id, page_tot, content, user_id FROM student_exam_content ORDER BY student_exam_id`,
		`SELECT student_exam_id, page, content, user_id FROM student_exam_page_content ORDER BY student_exam_id, page`,
	} {
		rows, err := conn.Query(query)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]sql.RawBytes, len(columns))
			destinations := make([]any, len(columns))
			for i := range values {
				destinations[i] = &values[i]
			}
			if err := rows.Scan(destinations...); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			for _, value := range values {
				hash.Write(value)
				hash.Write([]byte{0})
			}
			hash.Write([]byte{1})
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
		hash.Write([]byte{2})
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return digest
}

func assertSQLiteIntegrity(t *testing.T, conn *sql.DB) {
	t.Helper()
	var integrity string
	if err := conn.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatal(err)
	}
	if integrity != "ok" {
		t.Fatalf("integrity_check = %q", integrity)
	}
	rows, err := conn.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign_key_check reported a violation")
	}
}

func copyFile(t *testing.T, source, destination string) {
	t.Helper()
	in, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func fileDigest(t *testing.T, path string) [sha256.Size]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(data)
}
