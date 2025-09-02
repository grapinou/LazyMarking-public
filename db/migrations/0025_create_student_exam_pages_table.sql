-- +goose Up
CREATE TABLE student_exam_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    student_exam_id INTEGER NOT NULL,
    page INTEGER NOT NULL,
    page_info TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (student_exam_id) REFERENCES student_exam(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(student_exam_id, page, user_id)
);

-- +goose Down
DROP TABLE student_exam_pages;