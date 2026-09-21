-- +goose Up
CREATE TABLE qcm_shares (
    qcm_id INTEGER PRIMARY KEY REFERENCES qcm(id) ON DELETE CASCADE
);

-- Source identifiers are informational: deleting the source must not affect copies.
CREATE TABLE qcm_copy_origins (
    qcm_id INTEGER PRIMARY KEY REFERENCES qcm(id) ON DELETE CASCADE,
    source_qcm_id INTEGER NOT NULL,
    source_author TEXT NOT NULL
);

-- +goose Down
DROP TABLE qcm_copy_origins;
DROP TABLE qcm_shares;
