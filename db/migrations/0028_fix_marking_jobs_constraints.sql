-- +goose Up
CREATE TABLE marking_jobs_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    total_pages INTEGER DEFAULT 0,
    done_pages INTEGER DEFAULT 0,
    total_exams INTEGER DEFAULT 0,
    done_exams INTEGER DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed')),
    status_pdf TEXT NOT NULL DEFAULT 'running' CHECK (status_pdf IN ('running', 'success', 'failed')),
    exam_name TEXT,
    mark_table_name TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO marking_jobs_new (
    id, user_id, total_pages, done_pages, total_exams, done_exams,
    status, status_pdf, exam_name, mark_table_name
)
SELECT
    id, user_id, total_pages, done_pages, total_exams, done_exams,
    status,
    CASE WHEN status_pdf IN ('running', 'success', 'failed') THEN status_pdf ELSE 'failed' END,
    exam_name, mark_table_name
FROM marking_jobs;

DROP TABLE marking_jobs;
ALTER TABLE marking_jobs_new RENAME TO marking_jobs;

-- +goose Down
CREATE TABLE marking_jobs_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    total_pages INTEGER DEFAULT 0,
    done_pages INTEGER DEFAULT 0,
    total_exams INTEGER DEFAULT 0,
    done_exams INTEGER DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed')),
    status_pdf TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed')),
    exam_name TEXT,
    mark_table_name TEXT
);

INSERT INTO marking_jobs_old SELECT * FROM marking_jobs;
DROP TABLE marking_jobs;
ALTER TABLE marking_jobs_old RENAME TO marking_jobs;
