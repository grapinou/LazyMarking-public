-- +goose Up
CREATE TABLE marking_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    total_pages INTEGER DEFAULT 0,
    done_pages INTEGER DEFAULT 0,
    total_exams INTEGER DEFAULT 0,
    done_exams INTEGER DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed')),
    status_pdf TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed'))
);

-- +goose Down
DROP TABLE marking_jobs;