-- name: CreateAltImage :exec
INSERT INTO
    alt_images (
        alt_question_id,
        image_name,
        resize_percentage,
        user_id
    )
VALUES
    (
        :alt_question_id,
        :image_name,
        :resize_percentage,
        :user_id
    );

-- name: DeleteAltImage :exec
DELETE FROM
    alt_images
WHERE
    alt_question_id = :alt_question_id
    AND user_id = :user_id;

-- name: GetAltImageByAltQuestionID :one
SELECT
    *
FROM
    alt_images
WHERE
    alt_question_id = :alt_question_id
    AND user_id = :user_id;

-- name: UpdateSizeAltImage :exec
UPDATE
    alt_images
SET
    resize_percentage = :resize_percentage
WHERE
    alt_question_id = :alt_question_id
    AND user_id = :user_id;