-- +goose Up
CREATE TABLE alt_answers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alt_question_id INTEGER NOT NULL,
    state INTEGER NOT NULL DEFAULT 0 CHECK (state IN (0, 1)),
    content TEXT NOT NULL UNIQUE CHECK (length(trim(content)) > 0),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (alt_question_id) REFERENCES alt_questions(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE alt_answers;