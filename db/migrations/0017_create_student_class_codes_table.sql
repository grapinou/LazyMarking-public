-- +goose Up
CREATE TABLE student_class_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    student_id INTEGER NOT NULL,
    class_code_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (student_id) REFERENCES students (id) ON DELETE CASCADE,
    FOREIGN KEY (class_code_id) REFERENCES class_codes (id) ON DELETE RESTRICT,
    FOREIGN KEY (user_id) REFERENCES users (id)
);

-- +goose Down
DROP TABLE student_class_codes;