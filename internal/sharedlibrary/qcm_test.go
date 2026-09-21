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
	"github.com/mattn/go-sqlite3"
)

func qcmFixture(t *testing.T) (*sql.DB, string) {
	t.Helper()
	conn, dir := fixture(t)
	exec(t, conn, `INSERT INTO qcm(id,name,user_id) VALUES(1,'Contrôle énergie',1);
INSERT INTO questions(id,subject_id,theme_id,year_level_id,skill_id,difficulty_id,point_id,content,instruction,user_id)
VALUES(2,1,1,1,1,1,1,'Famille privée B','Consigne B',1),(3,1,1,1,1,1,1,'Famille privée C','',1),(4,1,1,1,1,1,1,'Hors QCM','',1);
INSERT INTO answers(question_id,content,state,user_id) VALUES(2,'Réponse B',1,1),(3,'Réponse C',1,1);
INSERT INTO images(question_id,image_name,resize_percentage,user_id) VALUES(2,'main.png',20,1);
INSERT INTO qcm_questions(qcm_id,question_id,user_id,position) VALUES(1,1,1,1),(1,3,1,2),(1,2,1,3);`)
	return conn, dir
}

func TestSharedQCMCompleteCopyAndIndependence(t *testing.T) {
	conn, dir := qcmFixture(t)
	ctx := context.Background()
	q := db.New(conn)
	if rows, err := q.GetSharedQCMs(ctx); err != nil || len(rows) != 0 {
		t.Fatalf("QCM not private by default: %v %v", rows, err)
	}
	if _, err := CopySharedQCM(ctx, conn, dir, 1, 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("private QCM copied: %v", err)
	}
	if n, err := q.ShareQCM(ctx, db.ShareQCMParams{QcmID: 1, UserID: 2}); err != nil || n != 0 {
		t.Fatalf("foreign share: %d %v", n, err)
	}
	if _, err := q.ShareQCM(ctx, db.ShareQCMParams{QcmID: 1, UserID: 1}); err != nil {
		t.Fatal(err)
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM question_shares") != 1 {
		t.Fatal("QCM implicitly published families")
	}
	for _, id := range []int64{2, 3} {
		if _, err := LoadShared(ctx, q, id); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("private family visible outside QCM: %v", err)
		}
		if _, err := LoadSharedQCMFamily(ctx, q, 1, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadSharedQCMFamily(ctx, q, 1, 4); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("non-member family visible: %v", err)
	}
	// Every text classification reuses Bob's label via the P6.1.1 key.
	for _, table := range []string{"subjects", "themes", "year_levels", "skills", "difficulties"} {
		exec(t, conn, fmt.Sprintf("INSERT INTO %s(id,name,user_id) VALUES(20,' SCIENCES ',2)", table))
	}
	exec(t, conn, "INSERT INTO points(id,point_value,user_id) VALUES(20,3,2)")
	id, err := CopySharedQCM(ctx, conn, dir, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM qcm WHERE id=? AND user_id=2 AND name='Contrôle énergie'", id) != 1 {
		t.Fatal("wrong QCM copy")
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM qcm_shares") != 1 || scalar(t, conn, "SELECT COUNT(*) FROM question_shares") != 1 {
		t.Fatal("copy published implicitly")
	}
	if scalar(t, conn, "SELECT source_qcm_id FROM qcm_copy_origins WHERE qcm_id=? AND source_author='Alice'", id) != 1 {
		t.Fatal("missing QCM provenance")
	}
	for _, table := range []string{"subjects", "themes", "year_levels", "skills", "difficulties", "points"} {
		if scalar(t, conn, "SELECT COUNT(*) FROM "+table+" WHERE user_id=2") != 1 {
			t.Fatalf("duplicate classification %s", table)
		}
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM questions WHERE user_id=2 AND subject_id=20 AND theme_id=20 AND year_level_id=20 AND skill_id=20 AND difficulty_id=20 AND point_id=20") != 3 {
		t.Fatal("classification ownership/mapping")
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM questions WHERE user_id=2") != 3 {
		t.Fatal("family copied more than once or outside composition")
	}
	// Publication in this test ONLY lets the same reader compare the new QCM.
	exec(t, conn, "INSERT INTO qcm_shares VALUES(?)", id)
	source, err := LoadSharedQCM(ctx, q, 1)
	if err != nil {
		t.Fatal(err)
	}
	copy, err := LoadSharedQCM(ctx, q, id)
	if err != nil {
		t.Fatal(err)
	}
	var copiedImages []string
	if len(copy.Families) != 3 {
		t.Fatalf("composition: %+v", copy)
	}
	for i, entry := range source.Families {
		got := copy.Families[i]
		a, b := entry.Family, got.Family
		if entry.Position != got.Position || a.Metadata.ID == b.Metadata.ID || b.Metadata.UserID != 2 ||
			a.Metadata.Content != b.Metadata.Content || a.Metadata.Instruction != b.Metadata.Instruction ||
			len(a.Versions) != len(b.Versions) || a.Metadata.PointValue != b.Metadata.PointValue {
			t.Fatalf("family/position not preserved: %+v / %+v", entry, got)
		}
		if scalar(t, conn, "SELECT source_question_id FROM question_copy_origins WHERE question_id=?", b.Metadata.ID) != a.Metadata.ID {
			t.Fatal("family provenance")
		}
		for j, av := range a.Versions {
			bv := b.Versions[j]
			if av.Content != bv.Content || !reflect.DeepEqual(av.Answers, bv.Answers) || (av.Image == nil) != (bv.Image == nil) {
				t.Fatalf("version differs: %+v / %+v", av, bv)
			}
			if j > 0 && av.ID == bv.ID {
				t.Fatal("variant ID reused")
			}
			if av.Image != nil {
				if av.Image.Name == bv.Image.Name || av.Image.Resize != bv.Image.Resize {
					t.Fatal("image shared or resize lost")
				}
				content, err := os.ReadFile(filepath.Join(dir, bv.Image.Name))
				if err != nil || string(content) != av.Image.Name {
					t.Fatalf("image bytes: %q %v", content, err)
				}
				copiedImages = append(copiedImages, bv.Image.Name)
			}
		}
	}
	for _, table := range []string{"exams", "exams_generated", "students", "student_exam", "student_exam_content", "student_exam_page_content"} {
		if scalar(t, conn, "SELECT COUNT(*) FROM "+table) != 0 {
			t.Fatalf("unexpected exam data in %s", table)
		}
	}
	exec(t, conn, `UPDATE qcm SET name='Alice edited' WHERE id=1;
UPDATE questions SET content='Alice edited',instruction='Alice new instruction' WHERE user_id=1;
UPDATE alt_questions SET content='Alice new variant' WHERE user_id=1;
UPDATE answers SET content='Alice new answer' WHERE user_id=1;
DELETE FROM qcm_questions WHERE qcm_id=1 AND position=2;`)
	updated, err := LoadSharedQCM(ctx, q, 1)
	if err != nil || len(updated.Families) != 2 || updated.Families[0].Family.Metadata.Instruction != "Alice new instruction" {
		t.Fatalf("preview stale: %+v %v", updated, err)
	}
	after, err := LoadSharedQCM(ctx, q, id)
	if err != nil || !reflect.DeepEqual(copy, after) {
		t.Fatal("source edit changed copy")
	}
	exec(t, conn, "UPDATE qcm SET name='Bob edited' WHERE id=?", id)
	exec(t, conn, "UPDATE questions SET content='Bob edited' WHERE user_id=2")
	if scalar(t, conn, "SELECT COUNT(*) FROM qcm WHERE id=1 AND name='Alice edited'") != 1 ||
		scalar(t, conn, "SELECT COUNT(*) FROM questions WHERE user_id=1 AND content='Alice edited'") != 4 {
		t.Fatal("copy edit affected original")
	}
	if _, err := q.UnshareQCM(ctx, db.UnshareQCMParams{QcmID: 1, UserID: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSharedQCM(ctx, q, 1); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("withdrawal did not revoke access: %v", err)
	}
	if _, err := LoadSharedQCMFamily(ctx, q, 1, 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("withdrawal did not revoke private image context: %v", err)
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM qcm_questions WHERE qcm_id=?", id) != 3 {
		t.Fatal("withdrawal changed copy")
	}
	// Delete source QCM, families and files: copy and provenance survive.
	exec(t, conn, "DELETE FROM qcm WHERE id=1; DELETE FROM questions WHERE user_id=1")
	for _, name := range []string{"main.png", "variant.png"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range copiedImages {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM qcm_copy_origins WHERE qcm_id=? AND source_author='Alice'", id) != 1 {
		t.Fatal("provenance depends on source")
	}
	if _, err := LoadSharedQCM(ctx, q, id); err != nil {
		t.Fatal(err)
	}
}

func TestCopyQCMRollback(t *testing.T) {
	for _, mode := range []string{"private", "missing", "own", "later family", "later image", "composition", "commit"} {
		t.Run(mode, func(t *testing.T) {
			conn, dir := qcmFixture(t)
			exec(t, conn, "INSERT INTO qcm_shares VALUES(1)")
			id, user := int64(1), int64(2)
			switch mode {
			case "private":
				exec(t, conn, "DELETE FROM qcm_shares")
			case "missing":
				id = 999
			case "own":
				user = 1
			case "later family":
				exec(t, conn, `CREATE TRIGGER fail_answer BEFORE INSERT ON answers WHEN NEW.user_id=2 AND NEW.content='Réponse B' BEGIN SELECT RAISE(ABORT,'injected failure'); END`)
			case "later image":
				exec(t, conn, "UPDATE images SET image_name='missing.png' WHERE question_id=2")
			case "composition":
				exec(t, conn, `CREATE TRIGGER fail_link BEFORE INSERT ON qcm_questions WHEN NEW.user_id=2 AND NEW.position=3 BEGIN SELECT RAISE(ABORT,'injected failure'); END`)
			case "commit":
				exec(t, conn, `CREATE TABLE deferred_failure(user_id INTEGER REFERENCES users(id) DEFERRABLE INITIALLY DEFERRED);
CREATE TRIGGER fail_commit AFTER INSERT ON qcm_copy_origins BEGIN INSERT INTO deferred_failure VALUES(999); END`)
			}
			before, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			_, err = CopySharedQCM(context.Background(), conn, dir, id, user)
			if err == nil || mode == "own" && !errors.Is(err, ErrOwnQCM) {
				t.Fatalf("copy error: %v", err)
			}
			for _, table := range []string{"qcm", "qcm_questions", "questions", "alt_questions", "answers", "alt_answers", "images", "alt_images", "subjects", "themes", "year_levels", "skills", "difficulties", "points"} {
				if scalar(t, conn, "SELECT COUNT(*) FROM "+table+" WHERE user_id=2") != 0 {
					t.Fatalf("partial copy in %s", table)
				}
			}
			if scalar(t, conn, "SELECT COUNT(*) FROM qcm") != 1 || scalar(t, conn, "SELECT COUNT(*) FROM questions") != 4 ||
				scalar(t, conn, "SELECT COUNT(*) FROM qcm_copy_origins") != 0 || scalar(t, conn, "SELECT COUNT(*) FROM question_copy_origins") != 0 {
				t.Fatal("source modified or orphan provenance")
			}
			after, err := os.ReadDir(dir)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("copied files survived rollback: %v %v", after, err)
			}
		})
	}
}

func TestCopyQCMNamesAndEmptyComposition(t *testing.T) {
	conn, dir := qcmFixture(t)
	exec(t, conn, "DELETE FROM qcm_questions; INSERT INTO qcm_shares VALUES(1)")
	for _, want := range []string{"Contrôle énergie", "Contrôle énergie — copie", "Contrôle énergie — copie 2"} {
		id, err := CopySharedQCM(context.Background(), conn, dir, 1, 2)
		if err != nil {
			t.Fatal(err)
		}
		var name string
		if err := conn.QueryRow("SELECT name FROM qcm WHERE id=?", id).Scan(&name); err != nil || name != want {
			t.Fatalf("name=%q want=%q err=%v", name, want, err)
		}
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM questions WHERE user_id=2") != 0 {
		t.Fatal("empty QCM copied unrelated families")
	}
}

func TestCopyQCMLeavesExistingExamHistoryUntouched(t *testing.T) {
	conn, dir := qcmFixture(t)
	for _, table := range []string{"class_codes", "periods", "years"} {
		exec(t, conn, "INSERT INTO "+table+"(id,name,user_id) VALUES(1,'Historique Alice',1)")
	}
	exec(t, conn, `INSERT INTO students(id,first_name,last_name,user_id) VALUES(1,'Élève','Alice',1);
INSERT INTO exams(id,name,qcm_id,class_code_id,period_id,year_id,user_id) VALUES(1,'Historique',1,1,1,1,1);
INSERT INTO exams_generated(id,exam_id,total_students,user_id) VALUES(1,1,1,1);
INSERT INTO student_exam(id,exam_generated_id,student_id,user_id) VALUES(1,1,1,1);
INSERT INTO student_exam_content(student_exam_id,page_tot,content,user_id) VALUES(1,1,'{"historical":"snapshot"}',1);
INSERT INTO student_exam_page_content(student_exam_id,page,content,user_id) VALUES(1,1,'{"historical":"coordinates"}',1);
INSERT INTO qcm_shares VALUES(1);`)
	if _, err := CopySharedQCM(context.Background(), conn, dir, 1, 2); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"class_codes", "periods", "years", "students", "exams", "exams_generated", "student_exam", "student_exam_content", "student_exam_page_content"} {
		if scalar(t, conn, "SELECT COUNT(*) FROM "+table) != 1 || scalar(t, conn, "SELECT COUNT(*) FROM "+table+" WHERE user_id=2") != 0 {
			t.Fatalf("exam history copied or modified: %s", table)
		}
	}
	if scalar(t, conn, `SELECT COUNT(*) FROM student_exam_content WHERE content='{"historical":"snapshot"}'`) != 1 ||
		scalar(t, conn, `SELECT COUNT(*) FROM student_exam_page_content WHERE content='{"historical":"coordinates"}'`) != 1 {
		t.Fatal("historical snapshot changed")
	}
}

func TestCopyQCMConcurrentSourceChangeIsCoherent(t *testing.T) {
	conn, dir := qcmFixture(t)
	exec(t, conn, "PRAGMA journal_mode=WAL; INSERT INTO qcm_shares VALUES(1)")
	// WAL allows another writer between source reads and the first copy write.
	// SQLite must reject promotion of this stale snapshot instead of mixing it
	// with a newer composition. A fresh copy then uses the whole new state.
	tx, err := conn.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	source, err := LoadSharedQCM(context.Background(), db.New(tx), 1)
	if err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "UPDATE qcm SET name='Changed during copy' WHERE id=1; DELETE FROM qcm_questions WHERE qcm_id=1 AND position=3")
	var files []string
	if _, err := cloneQCM(context.Background(), tx, dir, source, 2, &files); err != nil {
		var sqliteErr sqlite3.Error
		if !errors.As(err, &sqliteErr) || sqliteErr.ExtendedCode != sqlite3.ErrBusySnapshot {
			t.Fatalf("expected SQLITE_BUSY_SNAPSHOT, got %v", err)
		}
	} else {
		t.Fatal("stale SQLite snapshot unexpectedly promoted")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 || scalar(t, conn, "SELECT COUNT(*) FROM qcm WHERE user_id=2") != 0 {
		t.Fatal("concurrent conflict left a partial copy")
	}
	id, err := CopySharedQCM(context.Background(), conn, dir, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if scalar(t, conn, "SELECT COUNT(*) FROM qcm WHERE id=? AND name='Changed during copy'", id) != 1 ||
		scalar(t, conn, "SELECT COUNT(*) FROM qcm_questions WHERE qcm_id=?", id) != 2 {
		t.Fatal("fresh copy mixed states")
	}
}
