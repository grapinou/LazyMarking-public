-- +goose NO TRANSACTION
-- +goose Up
CREATE TEMP TABLE migration_0034_points_validation (
    point_value INTEGER CONSTRAINT migration_0034_point_value_must_be_positive CHECK (point_value >= 1)
);
INSERT INTO migration_0034_points_validation (point_value)
SELECT point_value FROM points;
DROP TABLE migration_0034_points_validation;

DROP TRIGGER questions_owner_update;
DROP TRIGGER questions_owner_insert;

PRAGMA foreign_keys = OFF;

CREATE TABLE points_with_positive_values (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    point_value INTEGER NOT NULL DEFAULT 1 CHECK (point_value >= 1),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (point_value, user_id)
);
INSERT INTO points_with_positive_values (id, point_value, user_id)
SELECT id, point_value, user_id FROM points;
DROP TABLE points;
ALTER TABLE points_with_positive_values RENAME TO points;

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

PRAGMA foreign_keys = ON;

-- +goose Down
DROP TRIGGER questions_owner_update;
DROP TRIGGER questions_owner_insert;

PRAGMA foreign_keys = OFF;

CREATE TABLE points_without_positive_check (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    point_value INTEGER NOT NULL DEFAULT 1,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (point_value, user_id)
);
INSERT INTO points_without_positive_check (id, point_value, user_id)
SELECT id, point_value, user_id FROM points;
DROP TABLE points;
ALTER TABLE points_without_positive_check RENAME TO points;

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

PRAGMA foreign_keys = ON;
