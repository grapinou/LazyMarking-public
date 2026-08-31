-- +goose Up
ALTER TABLE student_exam_page_content ADD COLUMN reference_storage_key TEXT;
ALTER TABLE student_exam_page_content ADD COLUMN reference_width INTEGER;
ALTER TABLE student_exam_page_content ADD COLUMN reference_height INTEGER;
ALTER TABLE student_exam_page_content ADD COLUMN reference_dpi INTEGER;
ALTER TABLE student_exam_page_content ADD COLUMN reference_sha256 TEXT;

CREATE TRIGGER student_exam_page_reference_metadata_insert
BEFORE INSERT ON student_exam_page_content
WHEN NOT COALESCE((
    (NEW.reference_storage_key IS NULL
        AND NEW.reference_width IS NULL
        AND NEW.reference_height IS NULL
        AND NEW.reference_dpi IS NULL
        AND NEW.reference_sha256 IS NULL)
    OR
    (NEW.reference_storage_key IS NOT NULL
        AND NEW.reference_storage_key != ''
        AND NEW.reference_storage_key NOT LIKE '/%'
        AND instr(NEW.reference_storage_key, char(92)) = 0
        AND NEW.reference_storage_key NOT LIKE '%//%'
        AND NEW.reference_storage_key NOT LIKE '../%'
        AND NEW.reference_storage_key NOT LIKE '%/../%'
        AND NEW.reference_storage_key NOT LIKE '%/..'
        AND NEW.reference_storage_key NOT LIKE './%'
        AND NEW.reference_storage_key NOT LIKE '%/./%'
        AND NEW.reference_storage_key NOT LIKE '%/.'
        AND lower(substr(NEW.reference_storage_key, -4)) = '.png'
        AND NEW.reference_width > 0
        AND NEW.reference_height > 0
        AND NEW.reference_dpi = 300
        AND length(NEW.reference_sha256) = 64
        AND NEW.reference_sha256 = lower(NEW.reference_sha256)
        AND NEW.reference_sha256 NOT GLOB '*[^0-9a-f]*')
), 0)
BEGIN
    SELECT RAISE(ABORT, 'page reference metadata must be all null or complete and valid');
END;

CREATE TRIGGER student_exam_page_reference_metadata_update
BEFORE UPDATE OF reference_storage_key, reference_width, reference_height, reference_dpi, reference_sha256
ON student_exam_page_content
WHEN NOT COALESCE((
    NEW.reference_storage_key IS NOT NULL
    AND NEW.reference_storage_key != ''
    AND NEW.reference_storage_key NOT LIKE '/%'
    AND instr(NEW.reference_storage_key, char(92)) = 0
    AND NEW.reference_storage_key NOT LIKE '%//%'
    AND NEW.reference_storage_key NOT LIKE '../%'
    AND NEW.reference_storage_key NOT LIKE '%/../%'
    AND NEW.reference_storage_key NOT LIKE '%/..'
    AND NEW.reference_storage_key NOT LIKE './%'
    AND NEW.reference_storage_key NOT LIKE '%/./%'
    AND NEW.reference_storage_key NOT LIKE '%/.'
    AND lower(substr(NEW.reference_storage_key, -4)) = '.png'
    AND NEW.reference_width > 0
    AND NEW.reference_height > 0
    AND NEW.reference_dpi = 300
    AND length(NEW.reference_sha256) = 64
    AND NEW.reference_sha256 = lower(NEW.reference_sha256)
    AND NEW.reference_sha256 NOT GLOB '*[^0-9a-f]*'
), 0)
BEGIN
    SELECT RAISE(ABORT, 'page reference metadata must be complete and valid');
END;

CREATE TRIGGER student_exam_page_reference_immutable
BEFORE UPDATE OF reference_storage_key, reference_width, reference_height, reference_dpi, reference_sha256
ON student_exam_page_content
WHEN OLD.reference_storage_key IS NOT NULL
 AND (
    OLD.reference_storage_key IS NOT NEW.reference_storage_key
    OR OLD.reference_width IS NOT NEW.reference_width
    OR OLD.reference_height IS NOT NEW.reference_height
    OR OLD.reference_dpi IS NOT NEW.reference_dpi
    OR OLD.reference_sha256 IS NOT NEW.reference_sha256
 )
BEGIN
    SELECT RAISE(ABORT, 'page reference metadata is immutable');
END;

-- The historical UNIQUE includes content and therefore does not make a page
-- unique. Preserve possible legacy duplicates, but reject new ambiguous pages.
CREATE TRIGGER student_exam_page_identity_insert
BEFORE INSERT ON student_exam_page_content
WHEN EXISTS (
    SELECT 1
    FROM student_exam_page_content AS existing
    WHERE existing.student_exam_id = NEW.student_exam_id
      AND existing.page = NEW.page
      AND existing.user_id = NEW.user_id
)
BEGIN
    SELECT RAISE(ABORT, 'student exam page already exists');
END;

-- +goose Down
DROP TRIGGER student_exam_page_identity_insert;
DROP TRIGGER student_exam_page_reference_immutable;
DROP TRIGGER student_exam_page_reference_metadata_update;
DROP TRIGGER student_exam_page_reference_metadata_insert;
ALTER TABLE student_exam_page_content DROP COLUMN reference_sha256;
ALTER TABLE student_exam_page_content DROP COLUMN reference_dpi;
ALTER TABLE student_exam_page_content DROP COLUMN reference_height;
ALTER TABLE student_exam_page_content DROP COLUMN reference_width;
ALTER TABLE student_exam_page_content DROP COLUMN reference_storage_key;
