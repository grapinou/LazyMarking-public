package sharedlibrary

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/grapinou/LazyMarking/internal/db"
)

func fixture(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	conn, _, err := db.OpenMigratedDB(context.Background(), filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	exec(t, conn, `INSERT INTO users(id,username,email,hashpassword) VALUES(1,'Alice','a@example.test','hash'),(2,'Bob','b@example.test','hash');`)
	for _, table := range []string{"subjects", "themes", "year_levels", "skills", "difficulties"} {
		exec(t, conn, fmt.Sprintf("INSERT INTO %s(id,name,user_id) VALUES(1,'Sciences',1)", table))
	}
	exec(t, conn, `INSERT INTO points(id,point_value,user_id) VALUES(1,3,1);
INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,content,instruction,user_id) VALUES(1,1,1,1,1,1,1,'Question Alice','Justifier.',1);
INSERT INTO answers(id,question_id,content,state,user_id) VALUES(1,1,'Oui',1,1),(2,1,'Non',0,1);
INSERT INTO alt_questions(id,question_id,content,user_id) VALUES(1,1,'Variante Alice',1),(2,1,'Sans image',1);
INSERT INTO alt_answers(id,alt_question_id,content,state,user_id) VALUES(1,1,'Alternative correcte',1,1),(2,1,'Alternative fausse',0,1);
INSERT INTO images(question_id,image_name,resize_percentage,user_id) VALUES(1,'main.png',60,1);
INSERT INTO alt_images(alt_question_id,image_name,resize_percentage,user_id) VALUES(1,'variant.png',35,1);
INSERT INTO question_shares VALUES(1);`)
	for _, name := range []string{"main.png", "variant.png"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return conn, dir
}

func exec(t *testing.T, conn *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}

func scalar(t *testing.T, conn *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := conn.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCopyCompleteFamilyAndIndependence(t *testing.T) {
	conn, dir := fixture(t)
	ctx := context.Background()
	// Exact matching personal classifications are reused, never Alice's rows.
	exec(t, conn, `INSERT INTO subjects(id,name,user_id) VALUES(20,'Sciences',2)`)
	id, err := CopyShared(ctx, conn, dir, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	q := db.New(conn)
	copy, err := q.GetQuestionByID(ctx, db.GetQuestionByIDParams{ID: id, UserID: 2})
	if err != nil {
		t.Fatal(err)
	}
	if id == 1 || copy.Content != "Question Alice" || copy.Instruction != "Justifier." || copy.SubjectID != 20 {
		t.Fatalf("copy: %+v", copy)
	}
	for _, table := range []string{"subjects", "themes", "year_levels", "skills", "difficulties", "points", "questions", "images"} {
		if n := scalar(t, conn, "SELECT COUNT(*) FROM "+table+" WHERE user_id=2"); n != 1 {
			t.Fatalf("%s owned by Bob=%d", table, n)
		}
	}
	for _, table := range []string{"answers", "alt_questions", "alt_answers"} {
		if n := scalar(t, conn, "SELECT COUNT(*) FROM "+table+" WHERE user_id=2"); n != 2 {
			t.Fatalf("%s owned by Bob=%d", table, n)
		}
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM question_shares WHERE question_id=?", id) != 0 {
		t.Fatal("copy was published")
	}
	if scalar(t, conn, "SELECT source_question_id FROM question_copy_origins WHERE question_id=? AND source_author='Alice'", id) != 1 {
		t.Fatal("missing provenance")
	}
	if scalar(t, conn, "SELECT point_value FROM points WHERE user_id=2") != 3 {
		t.Fatal("points changed")
	}
	// Publish temporarily to compare every field through the read-only loader.
	exec(t, conn, "INSERT INTO question_shares VALUES(?)", id)
	a, err := LoadShared(ctx, q, 1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := LoadShared(ctx, q, id)
	if err != nil {
		t.Fatal(err)
	}
	var copiedFiles []string
	for i := range a.Versions {
		av, bv := a.Versions[i], b.Versions[i]
		if av.Content != bv.Content || !reflect.DeepEqual(av.Answers, bv.Answers) {
			t.Fatalf("version not preserved: %+v / %+v", av, bv)
		}
		if av.Image != nil {
			if av.Image.Name == bv.Image.Name || av.Image.Resize != bv.Image.Resize {
				t.Fatal("image reference reused or size changed")
			}
			bytes, err := os.ReadFile(filepath.Join(dir, bv.Image.Name))
			if err != nil || string(bytes) != av.Image.Name {
				t.Fatalf("physical image copy: %q %v", bytes, err)
			}
			copiedFiles = append(copiedFiles, bv.Image.Name)
		}
	}
	exec(t, conn, "DELETE FROM question_shares WHERE question_id=?", id)
	exec(t, conn, `UPDATE questions SET content='Alice edited',instruction='New instruction' WHERE id=1;
UPDATE answers SET content='Alice answer edited' WHERE user_id=1;
UPDATE alt_questions SET content='Alice variant edited' WHERE user_id=1;
UPDATE subjects SET name='Alice subject edited' WHERE user_id=1;`)
	copy, err = q.GetQuestionByID(ctx, db.GetQuestionByIDParams{ID: id, UserID: 2})
	if err != nil || copy.Content != "Question Alice" || copy.Instruction != "Justifier." {
		t.Fatalf("source edits changed copy: %+v %v", copy, err)
	}
	exec(t, conn, `UPDATE questions SET content='Bob edited' WHERE id=?`, id)
	if scalar(t, conn, `SELECT COUNT(*) FROM questions WHERE id=1 AND content='Alice edited'`) != 1 {
		t.Fatal("copy edit changed original")
	}
	exec(t, conn, `DELETE FROM question_shares WHERE question_id=1`)
	if _, err := LoadShared(ctx, q, 1); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("withdrawn source visible: %v", err)
	}
	// Delete source rows and files exactly as the original's deletion would.
	exec(t, conn, `DELETE FROM questions WHERE id=1`)
	for _, name := range []string{"main.png", "variant.png"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range copiedFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM questions WHERE id=? AND user_id=2", id) != 1 {
		t.Fatal("copy lost after source deletion")
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM question_copy_origins WHERE question_id=? AND source_author='Alice'", id) != 1 {
		t.Fatal("provenance lost after source deletion")
	}
}

func TestCopyFailureRollsBackRowsAndFiles(t *testing.T) {
	for _, mode := range []string{"missing variant image", "database failure after images", "private", "missing", "symlink"} {
		t.Run(mode, func(t *testing.T) {
			conn, dir := fixture(t)
			id := int64(1)
			switch mode {
			case "missing variant image":
				if err := os.Remove(filepath.Join(dir, "variant.png")); err != nil {
					t.Fatal(err)
				}
			case "database failure after images":
				exec(t, conn, `CREATE TRIGGER fail_origin BEFORE INSERT ON question_copy_origins BEGIN SELECT RAISE(ABORT,'injected failure'); END;`)
			case "private":
				exec(t, conn, `DELETE FROM question_shares`)
			case "missing":
				id = 999
			case "symlink":
				if err := os.Remove(filepath.Join(dir, "variant.png")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(dir, "main.png"), filepath.Join(dir, "variant.png")); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := os.ReadDir(dir)
			if _, err := CopyShared(context.Background(), conn, dir, id, 2); err == nil {
				t.Fatal("copy unexpectedly succeeded")
			}
			for _, table := range []string{"subjects", "themes", "year_levels", "skills", "difficulties", "points", "questions", "answers", "alt_questions", "alt_answers", "images", "alt_images"} {
				if n := scalar(t, conn, "SELECT COUNT(*) FROM "+table+" WHERE user_id=2"); n != 0 {
					t.Fatalf("partial copy in %s: %d", table, n)
				}
			}
			after, _ := os.ReadDir(dir)
			if len(after) != len(before) {
				t.Fatalf("orphan copy files: %v", after)
			}
		})
	}
}

func TestOpenImageRejectsUnsafeFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"../secret.png", "/tmp/image.png", "sub/image.png", "..", "", "bad\\path.png"} {
		if f, err := OpenImage(dir, name); err == nil {
			f.Close()
			t.Fatalf("accepted %q", name)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "directory.png"), 0700); err != nil {
		t.Fatal(err)
	}
	if f, err := OpenImage(dir, "directory.png"); err == nil {
		f.Close()
		t.Fatal("directory accepted")
	}
}

func TestCopyFamilyWithoutInstructionImagesOrVariants(t *testing.T) {
	conn, dir := fixture(t)
	exec(t, conn, `DELETE FROM images; DELETE FROM alt_questions; UPDATE questions SET instruction=''`)
	id, err := CopyShared(context.Background(), conn, dir, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM questions WHERE id=? AND instruction='' AND user_id=2", id) != 1 {
		t.Fatal("empty instruction not preserved")
	}
	for _, table := range []string{"alt_questions", "alt_images", "images"} {
		if scalar(t, conn, "SELECT COUNT(*) FROM "+table+" WHERE user_id=2") != 0 {
			t.Fatalf("unexpected %s", table)
		}
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM answers WHERE question_id=? AND user_id=2", id) != 2 {
		t.Fatal("answers not copied")
	}
}

func TestCopyReusesNormalizedClassifications(t *testing.T) {
	conn, dir := fixture(t)
	for _, c := range []struct{ table, source, existing string }{
		{"subjects", "physique-chimie", "Physique-Chimie"},
		{"year_levels", "seconde", "Seconde"},
		{"themes", "Électricité", "Electricite"},
		{"skills", "Physique  Chimie", "Physique Chimie"},
		{"difficulties", " FACILE ", "Facile"},
	} {
		exec(t, conn, "UPDATE "+c.table+" SET name=? WHERE user_id=1", c.source)
		exec(t, conn, "INSERT INTO "+c.table+"(id,name,user_id) VALUES(20,?,2)", c.existing)
	}
	exec(t, conn, "INSERT INTO points(id,point_value,user_id) VALUES(20,3,2)")
	id, err := CopyShared(context.Background(), conn, dir, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	row, err := db.New(conn).GetQuestionByID(context.Background(), db.GetQuestionByIDParams{ID: id, UserID: 2})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []int64{row.SubjectID, row.ThemeID, row.YearLevelID, row.SkillID, row.DifficultyID, row.PointID} {
		if value != 20 {
			t.Fatalf("feature not reused: %+v", row)
		}
	}
	for _, table := range []string{"subjects", "year_levels", "themes", "skills", "difficulties", "points"} {
		if scalar(t, conn, "SELECT COUNT(*) FROM "+table+" WHERE user_id=2") != 1 {
			t.Fatalf("duplicate %s", table)
		}
	}
	var name string
	if err := conn.QueryRow("SELECT name FROM year_levels WHERE id=20").Scan(&name); err != nil || name != "Seconde" {
		t.Fatalf("label rewritten: %q %v", name, err)
	}
	exec(t, conn, "UPDATE year_levels SET name='2nde' WHERE user_id=1")
	id, err = CopyShared(context.Background(), conn, dir, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if scalar(t, conn, "SELECT year_level_id FROM questions WHERE id=?", id) == 20 {
		t.Fatal("semantic labels merged")
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM year_levels WHERE user_id=2") != 2 {
		t.Fatal("semantic label not created")
	}
}

func TestCopyOwnFamilyRefusedWithoutWrites(t *testing.T) {
	conn, dir := fixture(t)
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CopyShared(context.Background(), conn, dir, 1, 1); !errors.Is(err, ErrOwnFamily) {
		t.Fatalf("own copy: %v", err)
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM questions") != 1 || scalar(t, conn, "SELECT COUNT(*) FROM question_copy_origins") != 0 {
		t.Fatal("self copy wrote rows")
	}
	after, err := os.ReadDir(dir)
	if err != nil || len(after) != len(before) {
		t.Fatalf("self copy wrote files: %v %v", after, err)
	}
}
