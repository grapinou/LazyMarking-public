-- +goose NO TRANSACTION
-- +goose Up
DROP TRIGGER IF EXISTS student_exams_owner_update;
DROP TRIGGER IF EXISTS student_exams_owner_insert;

PRAGMA foreign_keys = OFF;

CREATE TABLE exams_generated_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exam_id INTEGER NOT NULL,
    processed_students INTEGER NOT NULL DEFAULT 0,
    total_students INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed')),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (exam_id) REFERENCES exams(id) ON DELETE RESTRICT,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(exam_id, user_id)
);

INSERT INTO exams_generated_new (
    id, exam_id, processed_students, total_students, created_at, status, user_id
)
SELECT id, exam_id, processed_students, total_students, created_at, status, user_id
FROM exams_generated;

DROP TABLE exams_generated;
ALTER TABLE exams_generated_new RENAME TO exams_generated;

CREATE TRIGGER generated_exams_owner_insert BEFORE INSERT ON exams_generated
WHEN NOT EXISTS (SELECT 1 FROM exams WHERE id = NEW.exam_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'exam must belong to user'); END;
CREATE TRIGGER generated_exams_owner_update BEFORE UPDATE OF exam_id, user_id ON exams_generated
WHEN NOT EXISTS (SELECT 1 FROM exams WHERE id = NEW.exam_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'exam must belong to user'); END;
CREATE TRIGGER student_exams_owner_insert BEFORE INSERT ON student_exam
WHEN NOT EXISTS (SELECT 1 FROM exams_generated WHERE id = NEW.exam_generated_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM students WHERE id = NEW.student_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'generated exam and student must belong to user'); END;
CREATE TRIGGER student_exams_owner_update BEFORE UPDATE OF exam_generated_id, student_id, user_id ON student_exam
WHEN NOT EXISTS (SELECT 1 FROM exams_generated WHERE id = NEW.exam_generated_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM students WHERE id = NEW.student_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'generated exam and student must belong to user'); END;

PRAGMA foreign_keys = ON;

-- +goose Down
DROP TRIGGER IF EXISTS student_exams_owner_update;
DROP TRIGGER IF EXISTS student_exams_owner_insert;

PRAGMA foreign_keys = OFF;

CREATE TABLE exams_generated_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exam_id INTEGER NOT NULL,
    processed_students INTEGER NOT NULL DEFAULT 0,
    total_students INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed')),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (exam_id) REFERENCES exams(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(exam_id, user_id)
);

INSERT INTO exams_generated_old (
    id, exam_id, processed_students, total_students, created_at, status, user_id
)
SELECT id, exam_id, processed_students, total_students, created_at, status, user_id
FROM exams_generated;

DROP TABLE exams_generated;
ALTER TABLE exams_generated_old RENAME TO exams_generated;

CREATE TRIGGER generated_exams_owner_insert BEFORE INSERT ON exams_generated
WHEN NOT EXISTS (SELECT 1 FROM exams WHERE id = NEW.exam_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'exam must belong to user'); END;
CREATE TRIGGER generated_exams_owner_update BEFORE UPDATE OF exam_id, user_id ON exams_generated
WHEN NOT EXISTS (SELECT 1 FROM exams WHERE id = NEW.exam_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'exam must belong to user'); END;
CREATE TRIGGER student_exams_owner_insert BEFORE INSERT ON student_exam
WHEN NOT EXISTS (SELECT 1 FROM exams_generated WHERE id = NEW.exam_generated_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM students WHERE id = NEW.student_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'generated exam and student must belong to user'); END;
CREATE TRIGGER student_exams_owner_update BEFORE UPDATE OF exam_generated_id, student_id, user_id ON student_exam
WHEN NOT EXISTS (SELECT 1 FROM exams_generated WHERE id = NEW.exam_generated_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM students WHERE id = NEW.student_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'generated exam and student must belong to user'); END;

PRAGMA foreign_keys = ON;
