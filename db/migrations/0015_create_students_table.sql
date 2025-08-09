-- +goose Up
CREATE TABLE
    students (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        first_name TEXT NOT NULL,
        last_name TEXT NOT NULL,
        user_id INTEGER NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id),
        CONSTRAINT unique_student_per_user UNIQUE (user_id, first_name, last_name)
    );

-- +goose Down
DROP TABLE students;