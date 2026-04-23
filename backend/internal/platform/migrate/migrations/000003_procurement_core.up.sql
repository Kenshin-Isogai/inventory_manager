CREATE TABLE IF NOT EXISTS external_projects (
    id TEXT PRIMARY KEY,
    project_key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS external_project_budget_categories (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES external_projects(id) ON DELETE CASCADE,
    category_key TEXT NOT NULL,
    name TEXT NOT NULL,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, category_key)
);

CREATE TABLE IF NOT EXISTS supplier_quotations (
    id TEXT PRIMARY KEY,
    supplier_id TEXT NOT NULL REFERENCES suppliers(id),
    quotation_number TEXT NOT NULL,
    issue_date DATE NOT NULL,
    artifact_path TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quotation_lines (
    id TEXT PRIMARY KEY,
    quotation_id TEXT NOT NULL REFERENCES supplier_quotations(id) ON DELETE CASCADE,
    item_id TEXT REFERENCES items(id),
    manufacturer_name TEXT NOT NULL,
    item_number TEXT NOT NULL,
    item_description TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    lead_time_days INTEGER NOT NULL DEFAULT 0,
    delivery_location TEXT NOT NULL DEFAULT '',
    accounting_category TEXT NOT NULL DEFAULT '',
    supplier_contact TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS procurement_batches (
    id TEXT PRIMARY KEY,
    batch_number TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    project_id TEXT REFERENCES external_projects(id),
    budget_category_id TEXT REFERENCES external_project_budget_categories(id),
    supplier_id TEXT REFERENCES suppliers(id),
    quotation_id TEXT REFERENCES supplier_quotations(id),
    status TEXT NOT NULL,
    normalized_status TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'manual',
    created_by TEXT NOT NULL DEFAULT 'local-user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS procurement_lines (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL REFERENCES procurement_batches(id) ON DELETE CASCADE,
    item_id TEXT REFERENCES items(id),
    quotation_line_id TEXT REFERENCES quotation_lines(id),
    requested_quantity INTEGER NOT NULL CHECK (requested_quantity > 0),
    unit TEXT NOT NULL DEFAULT 'pcs',
    delivery_location TEXT NOT NULL DEFAULT '',
    accounting_category TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS purchase_orders (
    id TEXT PRIMARY KEY,
    procurement_batch_id TEXT NOT NULL REFERENCES procurement_batches(id) ON DELETE CASCADE,
    order_number TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    issued_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS purchase_order_lines (
    id TEXT PRIMARY KEY,
    purchase_order_id TEXT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    procurement_line_id TEXT NOT NULL REFERENCES procurement_lines(id),
    ordered_quantity INTEGER NOT NULL CHECK (ordered_quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS procurement_status_projections (
    batch_id TEXT PRIMARY KEY REFERENCES procurement_batches(id) ON DELETE CASCADE,
    normalized_status TEXT NOT NULL,
    raw_status TEXT NOT NULL,
    quantity_progression JSONB NOT NULL DEFAULT '{}'::jsonb,
    external_request_reference TEXT NOT NULL DEFAULT '',
    last_observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS procurement_status_history (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL REFERENCES procurement_batches(id) ON DELETE CASCADE,
    normalized_status TEXT NOT NULL,
    raw_status TEXT NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    note TEXT NOT NULL DEFAULT ''
);

INSERT INTO external_projects (id, project_key, name, status) VALUES
    ('proj-er2-upgrade', 'ER2-UPGRADE', 'ER2 Production Upgrade', 'active'),
    ('proj-mk4-refresh', 'MK4-REFRESH', 'MK4 Cabinet Refresh', 'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO external_project_budget_categories (id, project_id, category_key, name) VALUES
    ('budget-er2-material', 'proj-er2-upgrade', 'material', 'Material Cost'),
    ('budget-er2-maintenance', 'proj-er2-upgrade', 'maintenance', 'Maintenance'),
    ('budget-mk4-material', 'proj-mk4-refresh', 'material', 'Material Cost')
ON CONFLICT (id) DO NOTHING;

INSERT INTO supplier_quotations (id, supplier_id, quotation_number, issue_date, artifact_path, status) VALUES
    ('quote-001', 'sup-misumi', 'MISUMI-Q-20260422', DATE '2026-04-22', '/artifacts/quotations/misumi-q-20260422.pdf', 'reviewed'),
    ('quote-002', 'sup-thorlabs', 'THORLABS-Q-20260423', DATE '2026-04-23', '/artifacts/quotations/thorlabs-q-20260423.pdf', 'draft')
ON CONFLICT (id) DO NOTHING;

INSERT INTO quotation_lines (id, quotation_id, item_id, manufacturer_name, item_number, item_description, quantity, lead_time_days, delivery_location, accounting_category, supplier_contact) VALUES
    ('quote-line-001', 'quote-001', 'item-er2', 'Omron', 'ER2-P4', 'Control relay pack of 4', 12, 14, 'Tokyo Assembly', 'parts', ''),
    ('quote-line-002', 'quote-002', 'item-mk44', 'Phoenix Contact', 'MK44-BX', 'Terminal block bulk box', 8, 10, 'Tokyo Assembly', 'parts', '')
ON CONFLICT (id) DO NOTHING;

INSERT INTO procurement_batches (id, batch_number, title, project_id, budget_category_id, supplier_id, quotation_id, status, normalized_status, source_type, created_by) VALUES
    ('batch-001', 'PR-20260422-001', 'ER2 shortage replenishment', 'proj-er2-upgrade', 'budget-er2-material', 'sup-misumi', 'quote-001', 'draft', 'draft', 'shortage', 'local-user'),
    ('batch-002', 'PR-20260423-002', 'MK4 cabinet restock', 'proj-mk4-refresh', 'budget-mk4-material', 'sup-thorlabs', 'quote-002', 'submitted', 'submitted', 'manual', 'local-user')
ON CONFLICT (id) DO NOTHING;

INSERT INTO procurement_lines (id, batch_id, item_id, quotation_line_id, requested_quantity, unit, delivery_location, accounting_category, note) VALUES
    ('pline-001', 'batch-001', 'item-er2', 'quote-line-001', 12, 'pcs', 'Tokyo Assembly', 'parts', 'Created from ER2 shortage'),
    ('pline-002', 'batch-002', 'item-mk44', 'quote-line-002', 8, 'pcs', 'Tokyo Assembly', 'parts', 'Restock for cabinet build')
ON CONFLICT (id) DO NOTHING;

INSERT INTO purchase_orders (id, procurement_batch_id, order_number, status, issued_at) VALUES
    ('po-001', 'batch-002', 'PO-20260423-001', 'issued', TIMESTAMPTZ '2026-04-23T09:00:00Z')
ON CONFLICT (id) DO NOTHING;

INSERT INTO purchase_order_lines (id, purchase_order_id, procurement_line_id, ordered_quantity) VALUES
    ('po-line-001', 'po-001', 'pline-002', 8)
ON CONFLICT (id) DO NOTHING;

INSERT INTO procurement_status_projections (batch_id, normalized_status, raw_status, quantity_progression, external_request_reference, last_observed_at, updated_at) VALUES
    ('batch-001', 'draft', 'draft', '{"requested":12,"ordered":0,"received":0}'::jsonb, '', NOW(), NOW()),
    ('batch-002', 'submitted', 'submitted_to_internal_flow', '{"requested":8,"ordered":8,"received":0}'::jsonb, 'LOCAL-SUBMIT-002', NOW(), NOW())
ON CONFLICT (batch_id) DO NOTHING;

INSERT INTO procurement_status_history (id, batch_id, normalized_status, raw_status, observed_at, note) VALUES
    ('psh-001', 'batch-001', 'draft', 'draft', NOW() - INTERVAL '1 day', 'Draft created from shortage'),
    ('psh-002', 'batch-002', 'draft', 'draft', NOW() - INTERVAL '1 day', 'Draft created manually'),
    ('psh-003', 'batch-002', 'submitted', 'submitted_to_internal_flow', NOW(), 'Submitted for local tracking')
ON CONFLICT (id) DO NOTHING;
