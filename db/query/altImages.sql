-- name: CreateAltImage :execrows
INSERT INTO
    alt_images (
        alt_question_id,
        image_name,
        resize_percentage,
        user_id
    )
SELECT :alt_question_id, :image_name, :resize_percentage, :user_id
WHERE EXISTS (SELECT 1 FROM alt_questions a
              WHERE a.id = :alt_question_id AND a.question_id = :question_id AND a.user_id = :user_id);

-- name: DeleteAltImage :execrows
DELETE FROM
    alt_images
WHERE
    alt_images.alt_question_id = :alt_question_id
    AND alt_images.user_id = :user_id
    AND EXISTS (SELECT 1 FROM alt_questions a
                WHERE a.id = alt_images.alt_question_id AND a.question_id = :question_id AND a.user_id = :user_id);

-- name: GetAltImageByAltQuestionID :one
SELECT
    *
FROM
    alt_images
WHERE
    alt_images.alt_question_id = :alt_question_id
    AND alt_images.user_id = :user_id
    AND EXISTS (SELECT 1 FROM alt_questions a
                WHERE a.id = alt_images.alt_question_id AND a.question_id = :question_id AND a.user_id = :user_id);

-- name: UpdateSizeAltImage :execrows
UPDATE
    alt_images
SET
    resize_percentage = :resize_percentage
WHERE
    alt_images.alt_question_id = :alt_question_id
    AND alt_images.user_id = :user_id
    AND EXISTS (SELECT 1 FROM alt_questions a
                WHERE a.id = alt_images.alt_question_id AND a.question_id = :question_id AND a.user_id = :user_id);
