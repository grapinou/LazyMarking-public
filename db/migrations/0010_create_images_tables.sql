-- +goose Up
CREATE TABLE images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question_id INTEGER NOT NULL UNIQUE,
    image_name TEXT NOT NULL,
    resize_percentage INTEGER NOT NULL DEFAULT 50,
    user_id INTEGER NOT NULL UNIQUE,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (question_id) REFERENCES questions(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE images;