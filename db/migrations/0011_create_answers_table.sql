-- +goose Up
CREATE TABLE answers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question_id INTEGER NOT NULL,
    state INTEGER NOT NULL DEFAULT 0 CHECK (state IN (0, 1)),
    content TEXT NOT NULL UNIQUE CHECK (length(trim(content)) > 0),
    user_id INTEGER NOT NULL,
    FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE answers