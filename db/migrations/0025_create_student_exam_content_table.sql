-- +goose Up
CREATE TABLE student_exam_content (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    student_exam_id INTEGER NOT NULL,
    page_tot INTEGER NOT NULL,
    content TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (student_exam_id) REFERENCES student_exam(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(student_exam_id, page_tot, content, user_id)
);

-- +goose Down
DROP TABLE student_exam_content;