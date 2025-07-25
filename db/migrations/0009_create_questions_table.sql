-- +goose Up
CREATE TABLE questions(
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        subject_id INTEGER NOT NULL,
        theme_id INTEGER NOT NULL,
        year_level_id INTEGER NOT NULL,
        skill_id INTEGER NOT NULL,
        difficulty_id INTEGER NOT NULL,
        point_id INTEGER NOT NULL,
        content TEXT NOT NULL UNIQUE CHECK (length(trim(content)) > 0),
        user_id INTEGER NOT NULL,
        FOREIGN KEY (subject_id) REFERENCES subjects(id) ON DELETE RESTRICT,
        FOREIGN KEY (theme_id) REFERENCES themes(id) ON DELETE RESTRICT,
        FOREIGN KEY (year_level_id) REFERENCES year_levels(id) ON DELETE RESTRICT,
        FOREIGN KEY (skill_id) REFERENCES skills(id) ON DELETE RESTRICT,
        FOREIGN KEY (difficulty_id) REFERENCES difficulties(id) ON DELETE RESTRICT,
        FOREIGN KEY (point_id) REFERENCES points(id) ON DELETE RESTRICT,
        FOREIGN KEY (user_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE questions;