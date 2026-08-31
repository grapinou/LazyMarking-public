package db

import (
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestMarkingJobGenerationMigrationAndCreationContract(t *testing.T) {
	conn, err := InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY);
		CREATE TABLE exams_generated (
			id INTEGER PRIMARY KEY, status TEXT NOT NULL, user_id INTEGER NOT NULL
		);
		CREATE TABLE marking_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			total_pages INTEGER DEFAULT 0, done_pages INTEGER DEFAULT 0,
			total_exams INTEGER DEFAULT 0, done_exams INTEGER DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed')),
			status_pdf TEXT NOT NULL DEFAULT 'running' CHECK (status_pdf IN ('running', 'success', 'failed')),
			exam_name TEXT, mark_table_name TEXT, completed_at TIMESTAMP
		);
		INSERT INTO users(id) VALUES (1), (2);
		INSERT INTO exams_generated(id, status, user_id) VALUES
			(10, 'success', 1), (11, 'running', 1), (12, 'failed', 1), (20, 'success', 2);
		INSERT INTO marking_jobs(id, user_id) VALUES (100, 1);
	`); err != nil {
		t.Fatalf("prepare schema: %v", err)
	}

	up, down := markingGenerationMigration(t)
	if _, err := conn.Exec(up); err != nil {
		t.Fatalf("migration Up: %v", err)
	}
	if _, err := conn.Exec(`
		ALTER TABLE marking_jobs ADD COLUMN result_schema_version INTEGER;
		ALTER TABLE marking_jobs ADD COLUMN marking_algorithm_version TEXT;
		ALTER TABLE marking_jobs ADD COLUMN detection_threshold REAL;
	`); err != nil {
		t.Fatalf("add current metadata columns: %v", err)
	}

	var legacyGeneration sql.NullInt64
	if err := conn.QueryRow("SELECT exam_generated_id FROM marking_jobs WHERE id = 100").Scan(&legacyGeneration); err != nil {
		t.Fatal(err)
	}
	if legacyGeneration.Valid {
		t.Fatalf("legacy generation = %v, want NULL", legacyGeneration)
	}
	if _, err := conn.Exec("UPDATE marking_jobs SET status = 'failed', completed_at = CURRENT_TIMESTAMP WHERE id = 100"); err != nil {
		t.Fatalf("legacy lifecycle update: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO marking_jobs(user_id) VALUES (1)"); err == nil {
		t.Fatal("direct INSERT without generation succeeded")
	}
	if _, err := conn.Exec("INSERT INTO marking_jobs(user_id, exam_generated_id) VALUES (1, 20)"); err == nil {
		t.Fatal("direct cross-user INSERT succeeded")
	}

	queries := New(conn)
	created, err := queries.CreateMarkingJob(t.Context(), CreateMarkingJobParams{
		UserID: 1, ExamGeneratedID: sql.NullInt64{Int64: 10, Valid: true},
		ResultSchemaVersion:     sql.NullInt64{Int64: 1, Valid: true},
		MarkingAlgorithmVersion: sql.NullString{String: "1", Valid: true},
		DetectionThreshold:      sql.NullFloat64{Float64: 150, Valid: true},
	})
	if err != nil {
		t.Fatalf("create owned success job: %v", err)
	}
	var gotUser int64
	var gotGeneration sql.NullInt64
	if err := conn.QueryRow("SELECT user_id, exam_generated_id FROM marking_jobs WHERE id = ?", created).Scan(&gotUser, &gotGeneration); err != nil {
		t.Fatal(err)
	}
	if gotUser != 1 || !gotGeneration.Valid || gotGeneration.Int64 != 10 {
		t.Fatalf("created job user=%d generation=%v", gotUser, gotGeneration)
	}

	for _, tc := range []struct {
		name       string
		userID     int64
		generation int64
	}{
		{name: "running", userID: 1, generation: 11},
		{name: "failed", userID: 1, generation: 12},
		{name: "foreign", userID: 1, generation: 20},
		{name: "missing", userID: 1, generation: 999},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := queries.CreateMarkingJob(t.Context(), CreateMarkingJobParams{
				UserID: tc.userID, ExamGeneratedID: sql.NullInt64{Int64: tc.generation, Valid: true},
				ResultSchemaVersion:     sql.NullInt64{Int64: 1, Valid: true},
				MarkingAlgorithmVersion: sql.NullString{String: "1", Valid: true},
				DetectionThreshold:      sql.NullFloat64{Float64: 150, Valid: true},
			})
			if !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("error=%v, want sql.ErrNoRows", err)
			}
		})
	}

	if _, err := conn.Exec("DELETE FROM exams_generated WHERE id = 10"); err == nil {
		t.Fatal("generation referenced by a marking job was deleted")
	}
	if _, err := conn.Exec("DELETE FROM marking_jobs WHERE id = ?", created); err != nil {
		t.Fatalf("purge linked job: %v", err)
	}
	if _, err := conn.Exec(`
		ALTER TABLE marking_jobs DROP COLUMN detection_threshold;
		ALTER TABLE marking_jobs DROP COLUMN marking_algorithm_version;
		ALTER TABLE marking_jobs DROP COLUMN result_schema_version;
	`); err != nil {
		t.Fatalf("drop current metadata columns: %v", err)
	}

	if _, err := conn.Exec(down); err != nil {
		t.Fatalf("migration Down: %v", err)
	}
	if _, err := conn.Exec("SELECT exam_generated_id FROM marking_jobs LIMIT 1"); err == nil {
		t.Fatal("exam_generated_id still exists after Down")
	}
}

func markingGenerationMigration(t *testing.T) (string, string) {
	t.Helper()
	migration, err := os.ReadFile("../../db/migrations/0036_bind_marking_jobs_to_exam_generation.sql")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(string(migration), "-- +goose Down", 2)
	if len(parts) != 2 {
		t.Fatal("migration has no Down section")
	}
	return strings.Replace(parts[0], "-- +goose Up", "", 1), parts[1]
}
