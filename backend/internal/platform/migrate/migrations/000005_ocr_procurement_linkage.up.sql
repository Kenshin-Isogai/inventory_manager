ALTER TABLE supplier_quotations
    ADD COLUMN IF NOT EXISTS source_ocr_job_id TEXT UNIQUE REFERENCES ocr_jobs(id);

ALTER TABLE procurement_batches
    ADD COLUMN IF NOT EXISTS source_ocr_job_id TEXT UNIQUE REFERENCES ocr_jobs(id);
