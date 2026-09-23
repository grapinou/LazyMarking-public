-- +goose Up
CREATE TABLE student_copy_access (
    student_exam_id INTEGER PRIMARY KEY REFERENCES student_exam(id) ON DELETE CASCADE,
    public_token TEXT NOT NULL UNIQUE CHECK (length(public_token) = 43),
    code_hash TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'unpublished' CHECK (state IN ('unpublished', 'published', 'revoked')),
    published_at TIMESTAMP,
    expires_at TIMESTAMP,
    failed_attempts INTEGER NOT NULL DEFAULT 0 CHECK (failed_attempts >= 0),
    locked_until TIMESTAMP,
    CHECK ((state IN ('unpublished', 'revoked') AND published_at IS NULL AND expires_at IS NULL)
        OR (state IN ('published', 'revoked') AND published_at IS NOT NULL AND expires_at IS NOT NULL))
);

-- +goose Down
DROP TABLE student_copy_access;
