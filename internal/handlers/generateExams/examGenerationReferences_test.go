package generateexams

import (
	"bytes"
	"context"
	"database/sql"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/handlers/tools"
)

func TestCompleteExamGenerationWithReferencesRequiresValidatedFiles(t *testing.T) {
	t.Run("missing metadata", func(t *testing.T) {
		conn, queries, workspace := generationReferenceTestDB(t)
		defer conn.Close()
		storeGenerationTestReference(t, queries, workspace, 100, 1)
		storeGenerationTestReference(t, queries, workspace, 101, 1)
		if err := completeExamGenerationWithReferences(1, "alice", 10, context.Background(), queries); err == nil {
			t.Fatal("generation completed with one missing page reference")
		}
		assertGenerationReferenceStatus(t, conn, "running")
	})

	t.Run("missing file", func(t *testing.T) {
		conn, queries, workspace := generationReferenceTestDB(t)
		defer conn.Close()
		for _, expected := range generationReferenceTestPages() {
			stored := storeGenerationTestReference(t, queries, workspace, expected.studentExamID, expected.page)
			if expected.studentExamID == 100 && expected.page == 2 {
				if err := os.Remove(stored.Path); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := completeExamGenerationWithReferences(1, "alice", 10, context.Background(), queries); err == nil {
			t.Fatal("generation completed with missing reference file")
		}
		assertGenerationReferenceStatus(t, conn, "running")
	})

	t.Run("corrupt file", func(t *testing.T) {
		conn, queries, workspace := generationReferenceTestDB(t)
		defer conn.Close()
		for _, expected := range generationReferenceTestPages() {
			stored := storeGenerationTestReference(t, queries, workspace, expected.studentExamID, expected.page)
			if expected.studentExamID == 100 && expected.page == 2 {
				if err := os.WriteFile(stored.Path, []byte("corrupt"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := completeExamGenerationWithReferences(1, "alice", 10, context.Background(), queries); err == nil {
			t.Fatal("generation completed with corrupt reference file")
		}
		assertGenerationReferenceStatus(t, conn, "running")
	})

	t.Run("complete", func(t *testing.T) {
		conn, queries, workspace := generationReferenceTestDB(t)
		defer conn.Close()
		for _, expected := range generationReferenceTestPages() {
			storeGenerationTestReference(t, queries, workspace, expected.studentExamID, expected.page)
		}
		if err := completeExamGenerationWithReferences(1, "alice", 10, context.Background(), queries); err != nil {
			t.Fatal(err)
		}
		assertGenerationReferenceStatus(t, conn, "success")
	})

	t.Run("ambiguous page", func(t *testing.T) {
		conn, queries, workspace := generationReferenceTestDB(t)
		defer conn.Close()
		for _, expected := range generationReferenceTestPages() {
			storeGenerationTestReference(t, queries, workspace, expected.studentExamID, expected.page)
		}
		if _, err := conn.Exec(`
			INSERT INTO student_exam_page_content(
				id, student_exam_id, page, content, user_id,
				reference_storage_key, reference_width, reference_height,
				reference_dpi, reference_sha256)
			SELECT 4, student_exam_id, page, '{"legacy":"duplicate"}', user_id,
				reference_storage_key, reference_width, reference_height,
				reference_dpi, reference_sha256
			FROM student_exam_page_content WHERE id = 1
		`); err != nil {
			t.Fatal(err)
		}
		if err := completeExamGenerationWithReferences(1, "alice", 10, context.Background(), queries); err == nil {
			t.Fatal("generation completed with an ambiguous historical page")
		}
		assertGenerationReferenceStatus(t, conn, "running")
	})
}

type generationReferenceTestPage struct {
	studentExamID int64
	page          int64
}

func generationReferenceTestPages() []generationReferenceTestPage {
	return []generationReferenceTestPage{{studentExamID: 100, page: 1}, {studentExamID: 100, page: 2}, {studentExamID: 101, page: 1}}
}

func generationReferenceTestDB(t *testing.T) (*sql.DB, *db.Queries, string) {
	t.Helper()
	t.Chdir(t.TempDir())
	conn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(`
		CREATE TABLE users(id INTEGER PRIMARY KEY, username TEXT NOT NULL);
		CREATE TABLE exams_generated(id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, status TEXT NOT NULL DEFAULT 'running');
		CREATE TABLE student_exam(id INTEGER PRIMARY KEY, exam_generated_id INTEGER NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE student_exam_page_content(
			id INTEGER PRIMARY KEY, student_exam_id INTEGER NOT NULL, page INTEGER NOT NULL,
			content TEXT NOT NULL, user_id INTEGER NOT NULL,
			reference_storage_key TEXT, reference_width INTEGER, reference_height INTEGER,
			reference_dpi INTEGER, reference_sha256 TEXT);
		INSERT INTO users VALUES(1,'alice');
		INSERT INTO exams_generated VALUES(10,1,'running');
		INSERT INTO student_exam VALUES(100,10,1),(101,10,1);
		INSERT INTO student_exam_page_content(id,student_exam_id,page,content,user_id) VALUES
			(1,100,1,'{}',1),(2,100,2,'{}',1),(3,101,1,'{}',1);
	`); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	workspace, ok := tools.CreateOperationTempDir("alice", "exam-10")
	if !ok {
		conn.Close()
		t.Fatal("create generation workspace")
	}
	return conn, db.New(conn), workspace
}

func storeGenerationTestReference(t *testing.T, queries *db.Queries, workspace string, studentExamID, page int64) tools.ResolvedStudentExamPageReference {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, int(page)+2, int(page)+3))); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(workspace, "native-student-"+stringInt64(studentExamID)+"-page-"+stringInt64(page)+".png")
	if err := os.WriteFile(source, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	stored, err := tools.StoreStudentExamPageReference(context.Background(), queries, 1, "alice", 10, studentExamID, page, source)
	if err != nil {
		t.Fatal(err)
	}
	return stored
}

func assertGenerationReferenceStatus(t *testing.T, conn *sql.DB, want string) {
	t.Helper()
	var got string
	if err := conn.QueryRow(`SELECT status FROM exams_generated WHERE id=10`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("status=%q, want %q", got, want)
	}
}
