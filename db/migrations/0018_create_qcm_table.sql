-- +goose Up
CREATE TABLE
    qcm (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL CHECK (length(trim(name)) > 0),
        user_id INTEGER NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id),
        UNIQUE (name, user_id)
    );

-- +goose Down
DROP TABLE qcm;