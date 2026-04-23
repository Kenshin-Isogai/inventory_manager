ALTER TABLE procurement_status_projections
    ADD COLUMN IF NOT EXISTS last_reconciled_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS sync_source TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sync_error TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS procurement_master_sync_runs (
    id TEXT PRIMARY KEY,
    sync_type TEXT NOT NULL,
    project_id TEXT REFERENCES external_projects(id) ON DELETE SET NULL,
    project_key TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    row_count INTEGER NOT NULL DEFAULT 0 CHECK (row_count >= 0),
    source TEXT NOT NULL DEFAULT '',
    triggered_by TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS procurement_webhook_events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    external_request_reference TEXT NOT NULL DEFAULT '',
    project_key TEXT NOT NULL DEFAULT '',
    normalized_status TEXT NOT NULL DEFAULT '',
    raw_status TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    processing_error TEXT NOT NULL DEFAULT ''
);
