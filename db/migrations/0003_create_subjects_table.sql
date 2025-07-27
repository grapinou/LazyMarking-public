-- +goose Up
CREATE TABLE subjects(
   id INTEGER PRIMARY KEY AUTOINCREMENT,
   name TEXT NOT NULL UNIQUE CHECK (length(trim(content)) > 0),
   user_id INTEGER NOT NULL,
   FOREIGN KEY (user_id) REFERENCES users(id),
   UNIQUE (name, user_id)
);

-- +goose Down
DROP TABLE subjects;