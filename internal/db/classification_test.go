package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestClassificationReuseAndHistoricalDuplicates(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "app.db")
	conn, _, err := OpenMigratedDB(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`INSERT INTO users(id,username,email,hashpassword) VALUES(1,'Alice','a@example.test','h'),(2,'Bob','b@example.test','h');
INSERT INTO year_levels(id,name,user_id) VALUES(1,'Seconde',1),(2,'seconde',1),(3,'SECONDE',2);`); err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	conn, report, err := OpenMigratedDB(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if len(report.Applied) != 0 {
		t.Fatalf("unexpected new migration: %+v", report)
	}
	q := New(conn)
	result, err := q.GetOrCreateClassification(ctx, "year_levels", " SECONDE ", 1)
	if err != nil || !result.Existing || result.ID != 1 || result.Name != "Seconde" {
		t.Fatalf("historical reuse: %+v %v", result, err)
	}
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM year_levels").Scan(&count); err != nil || count != 3 {
		t.Fatalf("historical rows changed: %d %v", count, err)
	}
	result, err = q.GetOrCreateClassification(ctx, "year_levels", "seconde", 2)
	if err != nil || result.ID != 3 || result.Name != "SECONDE" {
		t.Fatalf("owner scoped reuse: %+v %v", result, err)
	}
	result, err = q.GetOrCreateClassification(ctx, "year_levels", "2nde", 1)
	if err != nil || result.Existing || result.Name != "2nde" {
		t.Fatalf("semantic distinction: %+v %v", result, err)
	}
	for _, table := range []string{"subjects", "themes", "skills", "difficulties"} {
		first, err := q.GetOrCreateClassification(ctx, table, "Électricité  appliquée", 1)
		if err != nil {
			t.Fatal(err)
		}
		second, err := q.GetOrCreateClassification(ctx, table, " electricite appliquee ", 1)
		if err != nil || !second.Existing || first.ID != second.ID || second.Name != "Électricité  appliquée" {
			t.Fatalf("%s: %+v %v", table, second, err)
		}
	}
	if _, err := q.GetOrCreateClassification(ctx, "users", "Invalid", 1); err == nil {
		t.Fatal("unexpected table accepted")
	}
	if _, err := q.GetOrCreateClassification(ctx, "themes", " \t ", 1); err == nil {
		t.Fatal("empty label accepted")
	}
}
