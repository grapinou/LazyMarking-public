-- +goose Up
CREATE TABLE
    exams (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL CHECK (length(trim(name)) > 0),
        qcm_id INTEGER NOT NULL,
        class_code_id INTEGER NOT NULL,
        period_id INTEGER NOT NULL,
        year_id INTEGER NOT NULL,
        user_id INTEGER NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id),
        FOREIGN KEY (qcm_id) REFERENCES qcm (id),
        FOREIGN KEY (year_id) REFERENCES years (id),
        FOREIGN KEY (class_code_id) REFERENCES class_codes (id),
        FOREIGN KEY (period_id) REFERENCES periods (id),
        UNIQUE (name, qcm_id, class_code_id, user_id)
    );

-- +goose Down
DROP TABLE exams;