-- +goose Up
CREATE TABLE qcm_questions_with_qcm_cascade (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    qcm_id INTEGER NOT NULL,
    question_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 1),
    FOREIGN KEY (user_id) REFERENCES users (id),
    FOREIGN KEY (qcm_id) REFERENCES qcm (id) ON DELETE CASCADE,
    FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE RESTRICT,
    UNIQUE (qcm_id, question_id),
    UNIQUE (qcm_id, position)
);

INSERT INTO qcm_questions_with_qcm_cascade (id, qcm_id, question_id, user_id, position)
SELECT id, qcm_id, question_id, user_id, position
FROM qcm_questions;

DROP TABLE qcm_questions;
ALTER TABLE qcm_questions_with_qcm_cascade RENAME TO qcm_questions;

CREATE TRIGGER qcm_questions_owner_insert BEFORE INSERT ON qcm_questions
WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;
CREATE TRIGGER qcm_questions_owner_update BEFORE UPDATE OF qcm_id, question_id, user_id ON qcm_questions
WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;

-- +goose Down
CREATE TABLE qcm_questions_without_qcm_cascade (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    qcm_id INTEGER NOT NULL,
    question_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 1),
    FOREIGN KEY (user_id) REFERENCES users (id),
    FOREIGN KEY (qcm_id) REFERENCES qcm (id),
    FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE RESTRICT,
    UNIQUE (qcm_id, question_id),
    UNIQUE (qcm_id, position)
);

INSERT INTO qcm_questions_without_qcm_cascade (id, qcm_id, question_id, user_id, position)
SELECT id, qcm_id, question_id, user_id, position
FROM qcm_questions;

DROP TABLE qcm_questions;
ALTER TABLE qcm_questions_without_qcm_cascade RENAME TO qcm_questions;

CREATE TRIGGER qcm_questions_owner_insert BEFORE INSERT ON qcm_questions
WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;
CREATE TRIGGER qcm_questions_owner_update BEFORE UPDATE OF qcm_id, question_id, user_id ON qcm_questions
WHEN NOT EXISTS (SELECT 1 FROM qcm WHERE id = NEW.qcm_id AND user_id = NEW.user_id)
  OR NOT EXISTS (SELECT 1 FROM questions WHERE id = NEW.question_id AND user_id = NEW.user_id)
BEGIN SELECT RAISE(ABORT, 'QCM and question must belong to user'); END;
