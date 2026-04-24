CREATE TABLE IF NOT EXISTS manufacturers (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS categories (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS departments (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS suppliers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    contact_name TEXT,
    contact_email TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS items (
    id TEXT PRIMARY KEY,
    manufacturer_key TEXT NOT NULL REFERENCES manufacturers(key),
    category_key TEXT NOT NULL REFERENCES categories(key),
    canonical_item_number TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    default_supplier_id TEXT REFERENCES suppliers(id),
    note TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS supplier_item_aliases (
    id TEXT PRIMARY KEY,
    item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    supplier_id TEXT NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    supplier_item_number TEXT NOT NULL,
    units_per_order INTEGER NOT NULL DEFAULT 1 CHECK (units_per_order > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS device_scopes (
    id TEXT PRIMARY KEY,
    device_key TEXT NOT NULL,
    scope_key TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (device_key, scope_key)
);

CREATE TABLE IF NOT EXISTS inventory_events (
    id TEXT PRIMARY KEY,
    item_id TEXT NOT NULL REFERENCES items(id),
    location_code TEXT NOT NULL,
    event_type TEXT NOT NULL,
    quantity_delta INTEGER NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    device_scope_id TEXT REFERENCES device_scopes(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reservations (
    id TEXT PRIMARY KEY,
    item_id TEXT NOT NULL REFERENCES items(id),
    device_scope_id TEXT NOT NULL REFERENCES device_scopes(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status TEXT NOT NULL,
    requested_by TEXT NOT NULL DEFAULT 'local-user',
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS import_jobs (
    id TEXT PRIMARY KEY,
    import_type TEXT NOT NULL,
    status TEXT NOT NULL,
    file_name TEXT NOT NULL,
    summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO manufacturers (key, name) VALUES
    ('omron', 'Omron'),
    ('phoenix', 'Phoenix Contact'),
    ('molex', 'Molex')
ON CONFLICT (key) DO NOTHING;

INSERT INTO categories (key, name) VALUES
    ('relay', 'Relay'),
    ('terminal', 'Terminal Block'),
    ('connector', 'Connector')
ON CONFLICT (key) DO NOTHING;

INSERT INTO departments (key, name) VALUES
    ('production', 'Production'),
    ('procurement', 'Procurement')
ON CONFLICT (key) DO NOTHING;

INSERT INTO suppliers (id, name, contact_name, contact_email) VALUES
    ('sup-thorlabs', 'Thorlabs Japan', 'Sales Desk', 'sales@thorlabs.example'),
    ('sup-misumi', 'MISUMI', 'Support', 'support@misumi.example')
ON CONFLICT (id) DO NOTHING;

INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description, default_supplier_id, note) VALUES
    ('item-er2', 'omron', 'relay', 'ER2', 'Control relay', 'sup-misumi', 'Standard relay used in powerboard assemblies'),
    ('item-mk44', 'phoenix', 'terminal', 'MK-44', 'Terminal block 4P', 'sup-thorlabs', 'Common terminal block'),
    ('item-cn88', 'molex', 'connector', 'CN-88', 'I/O connector housing', 'sup-misumi', 'Used in cabinet harness')
ON CONFLICT (id) DO NOTHING;

INSERT INTO supplier_item_aliases (id, item_id, supplier_id, supplier_item_number, units_per_order) VALUES
    ('alias-er2-pack4', 'item-er2', 'sup-misumi', 'ER2-P4', 4),
    ('alias-mk44-bulk', 'item-mk44', 'sup-thorlabs', 'MK44-BX', 10)
ON CONFLICT (id) DO NOTHING;

INSERT INTO device_scopes (id, device_key, scope_key, description) VALUES
    ('ds-er2-powerboard', 'ER2', 'powerboard', 'ER2 powerboard assembly'),
    ('ds-mk4-cabinet', 'MK4', 'cabinet', 'MK4 cabinet assembly')
ON CONFLICT (id) DO NOTHING;

INSERT INTO inventory_events (id, item_id, location_code, event_type, quantity_delta, note, device_scope_id) VALUES
    ('evt-001', 'item-er2', 'TOKYO-A1', 'receive', 10, 'Initial local seed', 'ds-er2-powerboard'),
    ('evt-002', 'item-er2', 'TOKYO-A1', 'adjust', -1, 'Damaged unit', 'ds-er2-powerboard'),
    ('evt-003', 'item-mk44', 'TOKYO-B2', 'receive', 5, 'Initial local seed', 'ds-mk4-cabinet'),
    ('evt-004', 'item-cn88', 'TOKYO-C1', 'receive', 20, 'Initial local seed', NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO reservations (id, item_id, device_scope_id, quantity, status, requested_by, note) VALUES
    ('res-001', 'item-er2', 'ds-er2-powerboard', 12, 'reserved', 'local-user', 'Assembly request'),
    ('res-002', 'item-mk44', 'ds-mk4-cabinet', 8, 'awaiting_stock', 'local-user', 'Shortage follow-up')
ON CONFLICT (id) DO NOTHING;

INSERT INTO import_jobs (id, import_type, status, file_name, summary) VALUES
    ('imp-001', 'items_with_aliases', 'completed', 'items_with_aliases_20260422.csv', '{"item_inserted": 3, "item_updated": 0, "alias_inserted": 2, "alias_updated": 0, "alias_only": 0}'::jsonb),
    ('imp-002', 'items_with_aliases', 'completed', 'alias_updates_20260422.csv', '{"item_inserted": 0, "item_updated": 0, "alias_inserted": 1, "alias_updated": 13, "alias_only": 14}'::jsonb)
ON CONFLICT (id) DO NOTHING;
