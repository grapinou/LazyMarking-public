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
    id = :id
    AND user_id = :user_id;

-- name: GetAllimages :many
SELECT
    *
FROM
    images
WHERE
    user_id = :user_id
ORDER BY
    image_name;

-- name: Updateimage :exec
UPDATE
    images
SET
    image_name = :image_name
WHERE
    id = :id
    AND user_id = :user_id;