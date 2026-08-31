-- +goose Up
ALTER TABLE marking_jobs
ADD COLUMN ambiguity_delta REAL CHECK (ambiguity_delta IS NULL OR ambiguity_delta >= 0);

ALTER TABLE marking_jobs
ADD COLUMN review_revision INTEGER NOT NULL DEFAULT 0 CHECK (review_revision >= 0);

ALTER TABLE marking_jobs
ADD COLUMN artifacts_revision INTEGER NOT NULL DEFAULT 0
CHECK (artifacts_revision >= 0 AND artifacts_revision <= review_revision);

-- +goose StatementBegin
CREATE TRIGGER marking_jobs_ambiguity_delta_immutable
BEFORE UPDATE OF ambiguity_delta ON marking_jobs
WHEN OLD.ambiguity_delta IS NOT NULL
 AND OLD.ambiguity_delta IS NOT NEW.ambiguity_delta
BEGIN
    SELECT RAISE(ABORT, 'marking job ambiguity delta is immutable once set');
END;
-- +goose StatementEnd

CREATE TABLE marking_answer_reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    answer_detection_id INTEGER NOT NULL,
    reviewer_user_id INTEGER NOT NULL,
    reviewed_state INTEGER NOT NULL CHECK (reviewed_state IN (0, 1)),
    reviewed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revision INTEGER NOT NULL DEFAULT 1 CHECK (revision >= 1),
    FOREIGN KEY (answer_detection_id) REFERENCES marking_answer_detections(id) ON DELETE CASCADE,
    FOREIGN KEY (reviewer_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    UNIQUE(answer_detection_id)
);

-- +goose StatementBegin
CREATE TRIGGER marking_answer_reviews_owner_insert
BEFORE INSERT ON marking_answer_reviews
WHEN NOT EXISTS (
    SELECT 1
    FROM marking_answer_detections AS mad
    JOIN marking_question_results AS mqr ON mqr.id = mad.question_result_id
    JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
    JOIN marking_jobs AS mj
      ON mj.id = mcr.marking_job_id
     AND mj.user_id = mcr.user_id
    WHERE mad.id = NEW.answer_detection_id
      AND mj.user_id = NEW.reviewer_user_id
)
BEGIN
    SELECT RAISE(ABORT, 'marking review must belong to the result owner');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER marking_answer_reviews_owner_update
BEFORE UPDATE OF answer_detection_id, reviewer_user_id ON marking_answer_reviews
WHEN NOT EXISTS (
    SELECT 1
    FROM marking_answer_detections AS mad
    JOIN marking_question_results AS mqr ON mqr.id = mad.question_result_id
    JOIN marking_copy_results AS mcr ON mcr.id = mqr.copy_result_id
    JOIN marking_jobs AS mj
      ON mj.id = mcr.marking_job_id
     AND mj.user_id = mcr.user_id
    WHERE mad.id = NEW.answer_detection_id
      AND mj.user_id = NEW.reviewer_user_id
)
BEGIN
    SELECT RAISE(ABORT, 'marking review must belong to the result owner');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER marking_answer_reviews_detection_immutable
BEFORE UPDATE OF answer_detection_id ON marking_answer_reviews
WHEN OLD.answer_detection_id IS NOT NEW.answer_detection_id
BEGIN
    SELECT RAISE(ABORT, 'marking review detection is immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER marking_answer_reviews_revision_step
BEFORE UPDATE OF reviewed_state, reviewed_at, revision ON marking_answer_reviews
WHEN NEW.revision != OLD.revision + 1
BEGIN
    SELECT RAISE(ABORT, 'marking review revision must advance by one');
END;
-- +goose StatementEnd

CREATE TABLE marking_aligned_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    copy_result_id INTEGER NOT NULL,
    page_exam INTEGER NOT NULL CHECK (page_exam >= 1),
    storage_key TEXT NOT NULL CHECK (
        storage_key != ''
        AND storage_key NOT LIKE '/%'
        AND instr(storage_key, char(92)) = 0
        AND storage_key NOT LIKE '%//%'
        AND storage_key NOT LIKE '../%'
        AND storage_key NOT LIKE '%/../%'
        AND storage_key NOT LIKE '%/..'
        AND storage_key NOT LIKE './%'
        AND storage_key NOT LIKE '%/./%'
        AND storage_key NOT LIKE '%/.'
        AND lower(substr(storage_key, -4)) = '.png'
    ),
    width INTEGER NOT NULL CHECK (width > 0),
    height INTEGER NOT NULL CHECK (height > 0),
    sha256 TEXT NOT NULL CHECK (
        length(sha256) = 64
        AND sha256 = lower(sha256)
        AND sha256 NOT GLOB '*[^0-9a-f]*'
    ),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (copy_result_id) REFERENCES marking_copy_results(id) ON DELETE CASCADE,
    UNIQUE(copy_result_id, page_exam)
);

-- +goose StatementBegin
CREATE TRIGGER marking_aligned_pages_owner_insert
BEFORE INSERT ON marking_aligned_pages
WHEN NOT EXISTS (
    SELECT 1
    FROM marking_copy_results AS mcr
    JOIN marking_jobs AS mj
      ON mj.id = mcr.marking_job_id
     AND mj.user_id = mcr.user_id
    WHERE mcr.id = NEW.copy_result_id
      AND mcr.user_id = NEW.user_id
      AND mj.user_id = NEW.user_id
      AND mcr.outcome = 'corrected'
      AND NEW.storage_key = printf(
          'aligned/student-exam-%d/page-%d.png',
          mcr.student_exam_id,
          NEW.page_exam
      )
      AND (SELECT COUNT(*)
           FROM student_exam_page_content AS sep
           WHERE sep.student_exam_id = mcr.student_exam_id
             AND sep.user_id = mcr.user_id
             AND sep.page = NEW.page_exam) = 1
)
BEGIN
    SELECT RAISE(ABORT, 'aligned page must match copy owner, student exam, and page');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER marking_aligned_pages_owner_update
BEFORE UPDATE OF user_id, copy_result_id, page_exam, storage_key ON marking_aligned_pages
WHEN NOT EXISTS (
    SELECT 1
    FROM marking_copy_results AS mcr
    JOIN marking_jobs AS mj
      ON mj.id = mcr.marking_job_id
     AND mj.user_id = mcr.user_id
    WHERE mcr.id = NEW.copy_result_id
      AND mcr.user_id = NEW.user_id
      AND mj.user_id = NEW.user_id
      AND mcr.outcome = 'corrected'
      AND NEW.storage_key = printf(
          'aligned/student-exam-%d/page-%d.png',
          mcr.student_exam_id,
          NEW.page_exam
      )
      AND (SELECT COUNT(*)
           FROM student_exam_page_content AS sep
           WHERE sep.student_exam_id = mcr.student_exam_id
             AND sep.user_id = mcr.user_id
             AND sep.page = NEW.page_exam) = 1
)
BEGIN
    SELECT RAISE(ABORT, 'aligned page must match copy owner, student exam, and page');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER marking_aligned_pages_immutable
BEFORE UPDATE ON marking_aligned_pages
WHEN OLD.user_id IS NOT NEW.user_id
  OR OLD.copy_result_id IS NOT NEW.copy_result_id
  OR OLD.page_exam IS NOT NEW.page_exam
  OR OLD.storage_key IS NOT NEW.storage_key
  OR OLD.width IS NOT NEW.width
  OR OLD.height IS NOT NEW.height
  OR OLD.sha256 IS NOT NEW.sha256
  OR OLD.created_at IS NOT NEW.created_at
BEGIN
    SELECT RAISE(ABORT, 'aligned page metadata is immutable');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER marking_aligned_pages_immutable;
DROP TRIGGER marking_aligned_pages_owner_update;
DROP TRIGGER marking_aligned_pages_owner_insert;
DROP TABLE marking_aligned_pages;
DROP TRIGGER marking_answer_reviews_revision_step;
DROP TRIGGER marking_answer_reviews_detection_immutable;
DROP TRIGGER marking_answer_reviews_owner_update;
DROP TRIGGER marking_answer_reviews_owner_insert;
DROP TABLE marking_answer_reviews;
DROP TRIGGER marking_jobs_ambiguity_delta_immutable;
ALTER TABLE marking_jobs DROP COLUMN artifacts_revision;
ALTER TABLE marking_jobs DROP COLUMN review_revision;
ALTER TABLE marking_jobs DROP COLUMN ambiguity_delta;
