ALTER TABLE ocr_jobs
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_ocr_jobs_deleted_at ON ocr_jobs (deleted_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ocr_jobs_created_by ON ocr_jobs (created_by);
