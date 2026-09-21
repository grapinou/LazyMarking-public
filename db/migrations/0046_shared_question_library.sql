-- +goose Up
-- Absence of a row means private; no existing question is published.
CREATE TABLE question_shares (
    question_id INTEGER PRIMARY KEY REFERENCES questions(id) ON DELETE CASCADE
);

-- Informational snapshots only: deleting or changing the source cannot affect a copy.
CREATE TABLE question_copy_origins (
    question_id INTEGER PRIMARY KEY REFERENCES questions(id) ON DELETE CASCADE,
    source_question_id INTEGER NOT NULL,
    source_author TEXT NOT NULL
);

-- +goose Down
DROP TABLE question_copy_origins;
DROP TABLE question_shares;
