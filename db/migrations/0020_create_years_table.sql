-- +goose Up
CREATE TABLE
    years (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        value INTEGER NOT NULL,
        user_id INTEGER NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id),
        UNIQUE (value, user_id)
    );

-- +goose Down
DROP TABLE years;