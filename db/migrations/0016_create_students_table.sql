-- +goose Up
CREATE TABLE
    students (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        first_name TEXT NOT NULL CHECK (length(trim(first_name)) > 0),
        last_name TEXT NOT NULL CHECK (length(trim(last_name)) > 0),
        user_id INTEGER NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id),
        UNIQUE (user_id, first_name, last_name)
    );

-- +goose Down
DROP TABLE students;