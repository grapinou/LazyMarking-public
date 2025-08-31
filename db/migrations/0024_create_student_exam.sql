-- +goose Up
CREATE TABLE student_exam (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exam_generated_id INTEGER NOT NULL,
    student_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (exam_generated_id) REFERENCES exams_generated(id) ON DELETE CASCADE,
    FOREIGN KEY (student_id) REFERENCES students(id),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(exam_generated_id, student_id, user_id)
);

-- +goose Down
DROP TABLE student_exam;