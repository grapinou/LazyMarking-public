-- +goose Up
CREATE TABLE training_decks (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 generation_id INTEGER REFERENCES exams_generated(id) ON DELETE SET NULL,
 title TEXT NOT NULL CHECK (length(trim(title)) BETWEEN 1 AND 160),
 public_token TEXT NOT NULL UNIQUE CHECK (length(public_token) = 43),
 state TEXT NOT NULL DEFAULT 'draft' CHECK (state IN ('draft','published','closed')),
 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX training_decks_owner ON training_decks(user_id, created_at DESC);
CREATE TABLE training_cards (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 deck_id INTEGER NOT NULL REFERENCES training_decks(id) ON DELETE CASCADE,
 position INTEGER NOT NULL CHECK (position >= 0),
 selected INTEGER NOT NULL DEFAULT 0 CHECK (selected IN (0,1)),
 source_question_id INTEGER,
 source_variant_type TEXT,
 source_variant_id INTEGER,
 content_json TEXT NOT NULL,
 rendered_json TEXT,
 UNIQUE(deck_id,position)
);
-- +goose Down
DROP TABLE training_cards;
DROP TABLE training_decks;
