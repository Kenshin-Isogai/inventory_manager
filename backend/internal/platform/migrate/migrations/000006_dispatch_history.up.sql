ALTER TABLE supplier_quotations
    ADD COLUMN IF NOT EXISTS artifact_delete_status TEXT NOT NULL DEFAULT 'retained',
    ADD COLUMN IF NOT EXISTS artifact_deleted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS artifact_delete_error TEXT NOT NULL DEFAULT '';

ALTER TABLE ocr_jobs
    ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_retry_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS procurement_dispatch_outbox (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL UNIQUE REFERENCES procurement_batches(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL DEFAULT 'submit_procurement_request',
    status TEXT NOT NULL DEFAULT 'pending',
    idempotency_key TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    last_attempt_at TIMESTAMPTZ,
    next_attempt_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS procurement_dispatch_history (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL REFERENCES procurement_batches(id) ON DELETE CASCADE,
    normalized_status TEXT NOT NULL,
    external_request_reference TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_response JSONB NOT NULL DEFAULT '{}'::jsonb,
    evidence_file_references JSONB NOT NULL DEFAULT '[]'::jsonb,
    retryable BOOLEAN NOT NULL DEFAULT FALSE,
    normalized_error_code TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
