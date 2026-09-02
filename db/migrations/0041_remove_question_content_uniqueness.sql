-- +goose NO TRANSACTION
-- +goose Up
-- Question text is content, not identity. The same wording may legitimately be
-- reused when the stimulus, image, parent, or other pedagogical context differs.
PRAGMA foreign_keys = OFF;

-- These triggers are bound to the tables being rebuilt. Drop them explicitly
-- before either parent disappears, then recreate the same ownership contract.
DROP TRIGGER questions_owner_insert;
DROP TRIGGER questions_owner_update;
DROP TRIGGER alt_questions_owner_insert;
DROP TRIGGER alt_questions_owner_update;
DROP TRIGGER answers_owner_insert;
DROP TRIGGER answers_owner_update;
DROP TRIGGER images_owner_insert;
DROP TRIGGER images_owner_update;
DROP TRIGGER alt_answers_owner_insert;
DROP TRIGGER alt_answers_owner_update;
DROP TRIGGER alt_images_owner_insert;
DROP TRIGGER alt_images_owner_update;
DROP TRIGGER qcm_questions_owner_insert;
DROP TRIGGER qcm_questions_owner_update;

CREATE TABLE questions_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    subject_id INTEGER NOT NULL,
    theme_id INTEGER NOT NULL,
    year_level_id INTEGER NOT NULL,
    skill_id INTEGER NOT NULL,
    difficulty_id INTEGER NOT NULL,
    point_id INTEGER NOT NULL,
    content TEXT NOT NULL CHECK (length(trim(content)) > 0),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE RESTRICT,
    FOREIGN KEY (theme_id) REFERENCES themes(id) ON DELETE RESTRICT,
    FOREIGN KEY (year_level_id) REFERENCES year_levels(id) ON DELETE RESTRICT,
    FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE RESTRICT,
    FOREIGN KEY (difficulty_id) REFERENCES difficulties(id) ON DELETE RESTRICT,
    FOREIGN KEY (point_id) REFERENCES points(id) ON DELETE RESTRICT,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
INSERT INTO questions_new SELECT * FROM questions;
DROP TABLE questions;
ALTER TABLE questions_new RENAME TO questions;

CREATE TABLE alt_questions_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question_id INTEGER NOT NULL,
    content TEXT NOT NULL CHECK (length(trim(content)) > 0),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
INSERT INTO alt_questions_new SELECT * FROM alt_questions;
DROP TABLE alt_questions;
ALTER TABLE alt_questions_new RENAME TO alt_questions;

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

CREATE TRIGGER alt_questions_owner_insert BEFORE INSERT ON alt_questions
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;
CREATE TRIGGER alt_questions_owner_update BEFORE UPDATE OF question_id, user_id ON alt_questions
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;

CREATE TRIGGER answers_owner_insert BEFORE INSERT ON answers
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;
CREATE TRIGGER answers_owner_update BEFORE UPDATE OF question_id, user_id ON answers
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;

CREATE TRIGGER images_owner_insert BEFORE INSERT ON images
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;
CREATE TRIGGER images_owner_update BEFORE UPDATE OF question_id, user_id ON images
WHEN NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'question must belong to user'); END;

CREATE TRIGGER alt_answers_owner_insert BEFORE INSERT ON alt_answers
WHEN NOT EXISTS (SELECT 1 FROM alt_questions WHERE id = NEW.alt_question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'alternative question must belong to user'); END;
CREATE TRIGGER alt_answers_owner_update BEFORE UPDATE OF alt_question_id, user_id ON alt_answers
WHEN NOT EXISTS (SELECT 1 FROM alt_questions WHERE id = NEW.alt_question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'alternative question must belong to user'); END;

CREATE TRIGGER alt_images_owner_insert BEFORE INSERT ON alt_images
WHEN NOT EXISTS (SELECT 1 FROM alt_questions WHERE id = NEW.alt_question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'alternative question must belong to user'); END;
CREATE TRIGGER alt_images_owner_update BEFORE UPDATE OF alt_question_id, user_id ON alt_images
WHEN NOT EXISTS (SELECT 1 FROM alt_questions WHERE id = NEW.alt_question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'alternative question must belong to user'); END;

CREATE TRIGGER qcm_questions_owner_insert BEFORE INSERT ON qcm_questions
WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;
CREATE TRIGGER qcm_questions_owner_update BEFORE UPDATE OF qcm_id, question_id, user_id ON qcm_questions
WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;

PRAGMA foreign_keys = ON;

-- +goose Down
-- Restoring the removed UNIQUE constraints could discard or reject valid
-- duplicates created under this contract. This corrective migration is
-- intentionally irreversible and fails without changing data.
CREATE TEMP TABLE migration_0041_cannot_rollback (
    value INTEGER CONSTRAINT migration_0041_is_irreversible CHECK (value = 0)
);
INSERT INTO migration_0041_cannot_rollback(value) VALUES (1);
