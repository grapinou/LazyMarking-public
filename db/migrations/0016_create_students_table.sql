-- +goose Up
CREATE TABLE
    students (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        first_name TEXT NOT NULL,
        last_name TEXT NOT NULL,
        class_code_id INTEGER NOT NULL,
        user_id INTEGER NOT NULL,
        FOREIGN KEY (class_code_id) REFERENCES class_codes (id) ON DELETE RESTRICT,
        FOREIGN KEY (user_id) REFERENCES users (id),
        UNIQUE (user_id, first_name, last_name)
    );

-- +goose Down
DROP TABLE students;