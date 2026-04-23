CREATE TABLE IF NOT EXISTS ocr_jobs (
    id TEXT PRIMARY KEY,
    source_type TEXT NOT NULL DEFAULT 'quotation',
    file_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    artifact_path TEXT NOT NULL,
    status TEXT NOT NULL,
    provider TEXT NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT 'local-user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ocr_job_results (
    job_id TEXT PRIMARY KEY REFERENCES ocr_jobs(id) ON DELETE CASCADE,
    supplier_id TEXT REFERENCES suppliers(id),
    quotation_number TEXT NOT NULL DEFAULT '',
    issue_date DATE,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ocr_result_lines (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL REFERENCES ocr_jobs(id) ON DELETE CASCADE,
    item_id TEXT REFERENCES items(id),
    manufacturer_name TEXT NOT NULL,
    item_number TEXT NOT NULL,
    item_description TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    lead_time_days INTEGER NOT NULL DEFAULT 0,
    delivery_location TEXT NOT NULL DEFAULT '',
    budget_category_id TEXT REFERENCES external_project_budget_categories(id),
    accounting_category TEXT NOT NULL DEFAULT '',
    supplier_contact TEXT NOT NULL DEFAULT '',
    is_user_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
