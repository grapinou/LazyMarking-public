-- +goose NO TRANSACTION
-- +goose Up
CREATE TEMP TABLE migration_0034_points_validation (
    point_value INTEGER CONSTRAINT migration_0034_point_value_must_be_positive CHECK (point_value >= 1)
);
INSERT INTO migration_0034_points_validation (point_value)
SELECT point_value FROM points;
DROP TABLE migration_0034_points_validation;

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

PRAGMA foreign_keys = ON;

-- +goose Down
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

PRAGMA foreign_keys = ON;
