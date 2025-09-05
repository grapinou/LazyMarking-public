-- +goose Up
CREATE TABLE exams_generated (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    exam_id INTEGER NOT NULL,
    processed_students INTEGER NOT NULL DEFAULT 0,
    total_students INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed')),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (exam_id) REFERENCES exams(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(exam_id, user_id)
);

-- +goose Down
DROP TABLE exams_generated;