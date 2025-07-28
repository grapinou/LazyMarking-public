-- +goose Up
CREATE TABLE points (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    point_value INTEGER NOT NULL UNIQUE DEFAULT 1,
    user_id INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (point_value, user_id)
);

-- +goose Down
DROP TABLE points;