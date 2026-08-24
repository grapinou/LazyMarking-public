-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

CREATE TABLE subjects_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (name, user_id)
);
INSERT INTO subjects_new SELECT * FROM subjects;
DROP TABLE subjects;
ALTER TABLE subjects_new RENAME TO subjects;

CREATE TABLE themes_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (name, user_id)
);
INSERT INTO themes_new SELECT * FROM themes;
DROP TABLE themes;
ALTER TABLE themes_new RENAME TO themes;

CREATE TABLE year_levels_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (name, user_id)
);
INSERT INTO year_levels_new SELECT * FROM year_levels;
DROP TABLE year_levels;
ALTER TABLE year_levels_new RENAME TO year_levels;

CREATE TABLE skills_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (name, user_id)
);
INSERT INTO skills_new SELECT * FROM skills;
DROP TABLE skills;
ALTER TABLE skills_new RENAME TO skills;

CREATE TABLE difficulties_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (name, user_id)
);
INSERT INTO difficulties_new SELECT * FROM difficulties;
DROP TABLE difficulties;
ALTER TABLE difficulties_new RENAME TO difficulties;

CREATE TABLE points_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    point_value INTEGER NOT NULL DEFAULT 1,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (point_value, user_id)
);
INSERT INTO points_new SELECT * FROM points;
DROP TABLE points;
ALTER TABLE points_new RENAME TO points;

CREATE TABLE class_codes_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (name, user_id)
);
INSERT INTO class_codes_new SELECT * FROM class_codes;
DROP TABLE class_codes;
ALTER TABLE class_codes_new RENAME TO class_codes;

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
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (content, user_id)
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
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (content, user_id)
);
INSERT INTO alt_questions_new SELECT * FROM alt_questions;
DROP TABLE alt_questions;
ALTER TABLE alt_questions_new RENAME TO alt_questions;

PRAGMA foreign_keys = ON;

-- +goose Down
-- Reverting would make valid cross-user duplicates impossible and could lose data.
-- This data-preserving corrective migration is intentionally irreversible.
SELECT 1;
