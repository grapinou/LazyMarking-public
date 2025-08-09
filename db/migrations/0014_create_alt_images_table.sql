-- +goose Up
CREATE TABLE
    alt_images (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        alt_question_id INTEGER NOT NULL UNIQUE,
        image_name TEXT NOT NULL,
        resize_percentage INTEGER NOT NULL DEFAULT 50,
        user_id INTEGER NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id),
        FOREIGN KEY (alt_question_id) REFERENCES alt_questions (id) ON DELETE CASCADE
    );

-- +goose Down
DROP TABLE alt_images;