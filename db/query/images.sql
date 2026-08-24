-- name: CreateImage :exec
INSERT INTO
    images (
        question_id,
        image_name,
        resize_percentage,
        user_id
    )
VALUES
    (
        :question_id,
        :image_name,
        :resize_percentage,
        :user_id
    );

-- name: DeleteImage :exec
DELETE FROM
    images
WHERE
    question_id = :question_id
    AND user_id = :user_id;

-- name: UserOwnsImage :one
SELECT EXISTS (
    SELECT 1 FROM images i WHERE i.image_name = sqlc.arg('requested_image_name') AND i.user_id = sqlc.arg('user_id')
    UNION ALL
    SELECT 1 FROM alt_images ai WHERE ai.image_name = sqlc.arg('requested_image_name') AND ai.user_id = sqlc.arg('user_id')
);

-- name: GetImageByQuestionID :one
SELECT
    *
FROM
    images
WHERE
    question_id = :question_id
    AND user_id = :user_id;

-- name: UpdateSizeImage :exec
UPDATE
    images
SET
    resize_percentage = :resize_percentage
WHERE
    question_id = :question_id
    AND user_id = :user_id;
