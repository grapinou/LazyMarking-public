-- +goose Up
CREATE TABLE marking_copy_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    marking_job_id INTEGER NOT NULL,
    student_exam_id INTEGER NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('corrected', 'incomplete', 'not_seen', 'error')),
    expected_pages INTEGER NOT NULL CHECK (expected_pages >= 1),
    detected_pages INTEGER NOT NULL CHECK (detected_pages >= 0),
    score_half_units INTEGER,
    total_points INTEGER,
    failure_code TEXT,
    failure_detail TEXT,
    completed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (marking_job_id) REFERENCES marking_jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (student_exam_id) REFERENCES student_exam(id) ON DELETE RESTRICT,
    UNIQUE(marking_job_id, student_exam_id),
    CHECK (
        (outcome = 'corrected'
            AND score_half_units IS NOT NULL
            AND total_points IS NOT NULL
            AND total_points >= 1
            AND score_half_units >= 0
            AND score_half_units <= 2 * total_points
            AND failure_code IS NULL
            AND failure_detail IS NULL)
        OR
        (outcome IN ('incomplete', 'not_seen', 'error')
            AND score_half_units IS NULL
            AND total_points IS NULL)
    )
);

CREATE TABLE marking_question_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    copy_result_id INTEGER NOT NULL,
    question_index INTEGER NOT NULL CHECK (question_index >= 0),
    state TEXT NOT NULL CHECK (state IN ('incorrect', 'partial', 'correct')),
    score_half_units INTEGER NOT NULL CHECK (score_half_units >= 0),
    total_points INTEGER NOT NULL CHECK (total_points >= 1),
    FOREIGN KEY (copy_result_id) REFERENCES marking_copy_results(id) ON DELETE CASCADE,
    UNIQUE(copy_result_id, question_index),
    CHECK (
        (state = 'incorrect' AND score_half_units = 0)
        OR (state = 'partial' AND score_half_units = total_points)
        OR (state = 'correct' AND score_half_units = 2 * total_points)
    )
);

CREATE TABLE marking_answer_detections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question_result_id INTEGER NOT NULL,
    answer_index INTEGER NOT NULL CHECK (answer_index >= 0),
    detected_state INTEGER NOT NULL CHECK (detected_state IN (0, 1)),
    mean_gray REAL NOT NULL CHECK (mean_gray >= 0 AND mean_gray <= 255),
    FOREIGN KEY (question_result_id) REFERENCES marking_question_results(id) ON DELETE CASCADE,
    UNIQUE(question_result_id, answer_index)
);

CREATE TRIGGER marking_copy_results_owner_insert
BEFORE INSERT ON marking_copy_results
WHEN NOT EXISTS (
    SELECT 1
    FROM marking_jobs AS mj
    JOIN student_exam AS se
      ON se.exam_generated_id = mj.exam_generated_id
     AND se.user_id = mj.user_id
    WHERE mj.id = NEW.marking_job_id
      AND mj.user_id = NEW.user_id
      AND se.id = NEW.student_exam_id
)
BEGIN
    SELECT RAISE(ABORT, 'marking result must match job user and generation');
END;

CREATE TRIGGER marking_copy_results_owner_update
BEFORE UPDATE OF user_id, marking_job_id, student_exam_id ON marking_copy_results
WHEN NOT EXISTS (
    SELECT 1
    FROM marking_jobs AS mj
    JOIN student_exam AS se
      ON se.exam_generated_id = mj.exam_generated_id
     AND se.user_id = mj.user_id
    WHERE mj.id = NEW.marking_job_id
      AND mj.user_id = NEW.user_id
      AND se.id = NEW.student_exam_id
)
BEGIN
    SELECT RAISE(ABORT, 'marking result must match job user and generation');
END;

CREATE TRIGGER marking_question_results_corrected_parent_insert
BEFORE INSERT ON marking_question_results
WHEN NOT EXISTS (
    SELECT 1 FROM marking_copy_results
    WHERE id = NEW.copy_result_id AND outcome = 'corrected'
)
BEGIN
    SELECT RAISE(ABORT, 'question result requires a corrected copy result');
END;

CREATE TRIGGER marking_question_results_corrected_parent_update
BEFORE UPDATE OF copy_result_id ON marking_question_results
WHEN NOT EXISTS (
    SELECT 1 FROM marking_copy_results
    WHERE id = NEW.copy_result_id AND outcome = 'corrected'
)
BEGIN
    SELECT RAISE(ABORT, 'question result requires a corrected copy result');
END;

-- +goose Down
DROP TRIGGER marking_question_results_corrected_parent_update;
DROP TRIGGER marking_question_results_corrected_parent_insert;
DROP TRIGGER marking_copy_results_owner_update;
DROP TRIGGER marking_copy_results_owner_insert;
DROP TABLE marking_answer_detections;
DROP TABLE marking_question_results;
DROP TABLE marking_copy_results;
