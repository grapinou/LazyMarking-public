-- +goose Up
CREATE TABLE
    qcm_questions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        qcm_id INTEGER NOT NULL,
        question_id INTEGER NOT NULL,
        user_id INTEGER NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users (id),
        FOREIGN KEY (qcm_id) REFERENCES qcm (id),
        FOREIGN KEY (question_id) REFERENCES questions (id) ON DELETE RESTRICT,
        UNIQUE (qcm_id, question_id)
    );

-- +goose Down
DROP TABLE qcm_questions