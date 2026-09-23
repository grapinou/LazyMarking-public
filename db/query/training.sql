-- name: ListTrainingDecks :many
SELECT id, title, state, generation_id, public_token, created_at FROM training_decks WHERE user_id = ? ORDER BY created_at DESC, id DESC;

-- name: GetTrainingDeck :one
SELECT id, user_id, generation_id, title, public_token, state, created_at FROM training_decks WHERE id = ? AND user_id = ?;

-- name: GetPublishedTrainingDeck :one
SELECT id, title, public_token FROM training_decks WHERE public_token = ? AND state = 'published';

-- name: InsertTrainingDeck :one
INSERT INTO training_decks (user_id,generation_id,title,public_token) VALUES (?,?,?,?) RETURNING id;

-- name: InsertTrainingCard :exec
INSERT INTO training_cards (deck_id,position,selected,source_question_id,source_variant_type,source_variant_id,content_json) VALUES (?,?,?,?,?,?,?);

-- name: ListTrainingCards :many
SELECT id, deck_id, position, selected, source_question_id, source_variant_type, source_variant_id, content_json, rendered_json FROM training_cards WHERE deck_id = ? ORDER BY position,id;

-- name: UpdateTrainingCard :execrows
UPDATE training_cards SET selected = ?, position = ? WHERE training_cards.id = ? AND training_cards.deck_id = ? AND EXISTS (SELECT 1 FROM training_decks WHERE training_decks.id = training_cards.deck_id AND training_decks.user_id = ? AND training_decks.state = 'draft');

-- name: UpdateTrainingTitle :execrows
UPDATE training_decks SET title = ? WHERE id = ? AND user_id = ? AND state = 'draft';

-- name: SetTrainingState :execrows
UPDATE training_decks SET state = ? WHERE id = ? AND user_id = ? AND state = ?;

-- name: SetTrainingRenderedCard :execrows
UPDATE training_cards SET rendered_json = ? WHERE training_cards.id = ? AND training_cards.deck_id = ? AND EXISTS (SELECT 1 FROM training_decks WHERE training_decks.id = training_cards.deck_id AND training_decks.user_id = ? AND training_decks.state = 'draft');
