-- name: CreateExamGenerated :one
INSERT INTO
    exams_generated (exam_id, user_id)
VALUES
    (:exam_id, :user_id) RETURNING id;