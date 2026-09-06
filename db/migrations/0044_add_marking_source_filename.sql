-- +goose Up
ALTER TABLE marking_jobs ADD COLUMN source_pdf_filename TEXT;

-- The upload provenance is immutable once the job has been created.
-- +goose StatementBegin
CREATE TRIGGER marking_jobs_source_pdf_filename_immutable
BEFORE UPDATE OF source_pdf_filename ON marking_jobs
WHEN NEW.source_pdf_filename IS NOT OLD.source_pdf_filename
BEGIN
    SELECT RAISE(ABORT, 'marking source PDF filename is immutable');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER marking_jobs_source_pdf_filename_immutable;
ALTER TABLE marking_jobs DROP COLUMN source_pdf_filename;
