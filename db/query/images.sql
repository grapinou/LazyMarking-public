-- name: CreateImage :execrows
INSERT INTO
    images (
        question_id,
        image_name,
        resize_percentage,
        user_id
    )
SELECT :question_id, :image_name, :resize_percentage, :user_id
WHERE EXISTS (SELECT 1 FROM questions q WHERE q.id = :question_id AND q.user_id = :user_id);

-- name: DeleteImage :execrows
DELETE FROM
    images
WHERE
    images.question_id = :question_id
    AND images.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = images.question_id AND q.user_id = :user_id);

-- name: UserOwnsImage :one
SELECT EXISTS (
    SELECT 1 FROM images i WHERE i.image_name = sqlc.arg('requested_image_name') AND i.user_id = sqlc.arg('user_id')
    UNION ALL
    SELECT 1 FROM alt_images ai WHERE ai.image_name = sqlc.arg('requested_image_name') AND ai.user_id = sqlc.arg('user_id')
);

-- name: ListAllImageNames :many
SELECT image_name
FROM images
ORDER BY image_name;

-- name: GetImageByQuestionID :one
SELECT
    *
FROM
    images
WHERE
    images.question_id = :question_id
    AND images.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = images.question_id AND q.user_id = :user_id);

-- name: UpdateSizeImage :execrows
UPDATE
    images
SET
    resize_percentage = :resize_percentage
WHERE
    images.question_id = :question_id
    AND images.user_id = :user_id
    AND EXISTS (SELECT 1 FROM questions q WHERE q.id = images.question_id AND q.user_id = :user_id);
