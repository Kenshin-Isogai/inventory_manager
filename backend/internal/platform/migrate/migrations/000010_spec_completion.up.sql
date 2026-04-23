CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    device_key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    device_type TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    planned_start_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE manufacturers
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE departments
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE suppliers
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE items
    ADD COLUMN IF NOT EXISTS primary_department_key TEXT REFERENCES departments(key),
    ADD COLUMN IF NOT EXISTS engineering_domain TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lifecycle_status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE supplier_item_aliases
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE device_scopes
    ADD COLUMN IF NOT EXISTS device_id TEXT REFERENCES devices(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS parent_scope_id TEXT REFERENCES device_scopes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS scope_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS scope_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS owner_department_key TEXT REFERENCES departments(key) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS planned_start_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS planned_end_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE device_scopes
SET scope_name = COALESCE(NULLIF(scope_name, ''), scope_key)
WHERE scope_name = '';

CREATE TABLE IF NOT EXISTS scope_departments (
    id TEXT PRIMARY KEY,
    scope_id TEXT NOT NULL REFERENCES device_scopes(id) ON DELETE CASCADE,
    department_key TEXT NOT NULL REFERENCES departments(key) ON DELETE CASCADE,
    involvement_type TEXT NOT NULL DEFAULT 'owner',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (scope_id, department_key, involvement_type)
);

CREATE TABLE IF NOT EXISTS locations (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    location_type TEXT NOT NULL DEFAULT 'stockroom',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inventory_balances (
    item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    location_code TEXT NOT NULL REFERENCES locations(code) ON DELETE CASCADE,
    on_hand_quantity INTEGER NOT NULL DEFAULT 0,
    reserved_quantity INTEGER NOT NULL DEFAULT 0,
    available_quantity INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (item_id, location_code)
);

ALTER TABLE inventory_events
    ADD COLUMN IF NOT EXISTS from_location_code TEXT,
    ADD COLUMN IF NOT EXISTS to_location_code TEXT,
    ADD COLUMN IF NOT EXISTS actor_id TEXT,
    ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS correlation_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reversed_by_event_id TEXT REFERENCES inventory_events(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE inventory_events
SET to_location_code = COALESCE(NULLIF(to_location_code, ''), location_code)
WHERE COALESCE(to_location_code, '') = '';

CREATE TABLE IF NOT EXISTS inventory_event_links (
    id TEXT PRIMARY KEY,
    inventory_event_id TEXT NOT NULL REFERENCES inventory_events(id) ON DELETE CASCADE,
    linked_entity_type TEXT NOT NULL,
    linked_entity_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS assemblies (
    id TEXT PRIMARY KEY,
    assembly_code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS assembly_components (
    assembly_id TEXT NOT NULL REFERENCES assemblies(id) ON DELETE CASCADE,
    item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (assembly_id, item_id)
);

CREATE TABLE IF NOT EXISTS scope_assembly_requirements (
    id TEXT PRIMARY KEY,
    scope_id TEXT NOT NULL REFERENCES device_scopes(id) ON DELETE CASCADE,
    assembly_id TEXT NOT NULL REFERENCES assemblies(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS scope_item_requirements (
    id TEXT PRIMARY KEY,
    scope_id TEXT NOT NULL REFERENCES device_scopes(id) ON DELETE CASCADE,
    item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (scope_id, item_id)
);

CREATE TABLE IF NOT EXISTS scope_assembly_usage (
    id TEXT PRIMARY KEY,
    scope_id TEXT NOT NULL REFERENCES device_scopes(id) ON DELETE CASCADE,
    location_code TEXT REFERENCES locations(code) ON DELETE SET NULL,
    assembly_id TEXT NOT NULL REFERENCES assemblies(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL CHECK (quantity >= 0),
    note TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (scope_id, location_code, assembly_id)
);

ALTER TABLE reservations
    ADD COLUMN IF NOT EXISTS purpose TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'normal',
    ADD COLUMN IF NOT EXISTS needed_by_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS planned_use_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS hold_until_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS fulfilled_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS released_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS cancellation_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS reservation_allocations (
    id TEXT PRIMARY KEY,
    reservation_id TEXT NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    item_id TEXT NOT NULL REFERENCES items(id),
    location_code TEXT NOT NULL REFERENCES locations(code),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status TEXT NOT NULL DEFAULT 'allocated',
    allocated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ,
    note TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS reservation_events (
    id TEXT PRIMARY KEY,
    reservation_id TEXT NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 0,
    actor_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE import_jobs
    ADD COLUMN IF NOT EXISTS lifecycle_state TEXT NOT NULL DEFAULT 'applied',
    ADD COLUMN IF NOT EXISTS redo_of_job_id TEXT REFERENCES import_jobs(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS created_by TEXT NOT NULL DEFAULT 'local-user',
    ADD COLUMN IF NOT EXISTS undone_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE TABLE IF NOT EXISTS import_rows (
    id TEXT PRIMARY KEY,
    import_job_id TEXT NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    row_number INTEGER NOT NULL CHECK (row_number > 0),
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    normalized_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'pending',
    code TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS import_effects (
    id TEXT PRIMARY KEY,
    import_job_id TEXT NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    import_row_id TEXT REFERENCES import_rows(id) ON DELETE SET NULL,
    target_entity_type TEXT NOT NULL,
    target_entity_id TEXT NOT NULL,
    effect_type TEXT NOT NULL,
    before_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS receipts (
    id TEXT PRIMARY KEY,
    purchase_order_id TEXT REFERENCES purchase_orders(id) ON DELETE SET NULL,
    receipt_number TEXT NOT NULL UNIQUE,
    source_type TEXT NOT NULL DEFAULT 'manual',
    received_by TEXT NOT NULL DEFAULT 'local-user',
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS receipt_lines (
    id TEXT PRIMARY KEY,
    receipt_id TEXT NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
    purchase_order_line_id TEXT REFERENCES purchase_order_lines(id) ON DELETE SET NULL,
    item_id TEXT NOT NULL REFERENCES items(id),
    location_code TEXT NOT NULL REFERENCES locations(code),
    received_quantity INTEGER NOT NULL CHECK (received_quantity > 0),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    key TEXT PRIMARY KEY,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_key TEXT NOT NULL REFERENCES roles(key) ON DELETE CASCADE,
    permission_key TEXT NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_key, permission_key)
);

CREATE TABLE IF NOT EXISTS user_status_history (
    id TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    changed_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE app_users
    ADD COLUMN IF NOT EXISTS username TEXT,
    ADD COLUMN IF NOT EXISTS requested_role TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS registration_memo TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS hosted_domain TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bootstrap BOOLEAN NOT NULL DEFAULT FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS app_users_username_unique
    ON app_users (LOWER(username))
    WHERE username IS NOT NULL;

ALTER TABLE procurement_batches
    ADD COLUMN IF NOT EXISTS device_key TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS scope_key TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE procurement_lines
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'draft',
    ADD COLUMN IF NOT EXISTS lead_time_days INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS budget_category_id TEXT REFERENCES external_project_budget_categories(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS supplier_contact TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE quotation_lines
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE purchase_order_lines
    ADD COLUMN IF NOT EXISTS received_quantity INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS expected_arrival_date DATE,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'ordered',
    ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS purchase_order_line_lineage_events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    source_line_id TEXT REFERENCES purchase_order_lines(id) ON DELETE SET NULL,
    target_line_id TEXT REFERENCES purchase_order_lines(id) ON DELETE SET NULL,
    quantity INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO devices (id, device_key, name, status)
SELECT 'device-' || LOWER(ds.device_key), ds.device_key, ds.device_key, 'active'
FROM (
    SELECT DISTINCT device_key
    FROM device_scopes
    WHERE COALESCE(device_key, '') <> ''
) ds
ON CONFLICT (id) DO NOTHING;

UPDATE device_scopes ds
SET device_id = d.id
FROM devices d
WHERE ds.device_id IS NULL
  AND d.device_key = ds.device_key;

INSERT INTO locations (code, name, location_type)
SELECT DISTINCT location_code, location_code, 'stockroom'
FROM inventory_events
WHERE COALESCE(location_code, '') <> ''
ON CONFLICT (code) DO NOTHING;

INSERT INTO locations (code, name, location_type)
VALUES
    ('RECEIVING', 'Receiving Area', 'receiving'),
    ('INSPECTION', 'Inspection Area', 'inspection')
ON CONFLICT (code) DO NOTHING;

INSERT INTO inventory_balances (item_id, location_code, on_hand_quantity, reserved_quantity, available_quantity, updated_at)
SELECT
    item_id,
    location_code,
    SUM(quantity_delta) AS on_hand_quantity,
    0,
    SUM(quantity_delta) AS available_quantity,
    NOW()
FROM inventory_events
WHERE COALESCE(location_code, '') <> ''
GROUP BY item_id, location_code
ON CONFLICT (item_id, location_code) DO UPDATE
SET on_hand_quantity = EXCLUDED.on_hand_quantity,
    available_quantity = EXCLUDED.available_quantity,
    updated_at = NOW();

INSERT INTO permissions (key, description) VALUES
    ('admin.users.read', 'View user registrations and active users'),
    ('admin.users.write', 'Approve or reject users'),
    ('admin.roles.read', 'View roles and permissions'),
    ('admin.roles.write', 'Manage role permissions'),
    ('admin.master.read', 'View master data'),
    ('admin.master.write', 'Manage master data'),
    ('operator.requirements.read', 'View scope requirements'),
    ('operator.requirements.write', 'Manage scope requirements'),
    ('operator.reservations.read', 'View reservations'),
    ('operator.reservations.write', 'Create and update reservations'),
    ('operator.shortages.read', 'View shortages'),
    ('operator.imports.read', 'View import jobs'),
    ('operator.imports.write', 'Run import jobs'),
    ('inventory.read', 'View inventory balances and events'),
    ('inventory.write', 'Execute inventory commands'),
    ('inventory.undo', 'Undo inventory, reservation, receipt, and import operations'),
    ('procurement.read', 'View procurement projections'),
    ('procurement.write', 'Create and update procurement drafts'),
    ('procurement.sync', 'Reconcile and refresh procurement projections'),
    ('inspector.receive', 'Record arrivals and inspection receipts')
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_key, permission_key) VALUES
    ('admin', 'admin.users.read'),
    ('admin', 'admin.users.write'),
    ('admin', 'admin.roles.read'),
    ('admin', 'admin.roles.write'),
    ('admin', 'admin.master.read'),
    ('admin', 'admin.master.write'),
    ('admin', 'operator.requirements.read'),
    ('admin', 'operator.requirements.write'),
    ('admin', 'operator.reservations.read'),
    ('admin', 'operator.reservations.write'),
    ('admin', 'operator.shortages.read'),
    ('admin', 'operator.imports.read'),
    ('admin', 'operator.imports.write'),
    ('admin', 'inventory.read'),
    ('admin', 'inventory.write'),
    ('admin', 'inventory.undo'),
    ('admin', 'procurement.read'),
    ('admin', 'procurement.write'),
    ('admin', 'procurement.sync'),
    ('admin', 'inspector.receive'),
    ('operator', 'operator.requirements.read'),
    ('operator', 'operator.requirements.write'),
    ('operator', 'operator.reservations.read'),
    ('operator', 'operator.reservations.write'),
    ('operator', 'operator.shortages.read'),
    ('operator', 'operator.imports.read'),
    ('operator', 'operator.imports.write'),
    ('inventory', 'inventory.read'),
    ('inventory', 'inventory.write'),
    ('inventory', 'inventory.undo'),
    ('inventory', 'operator.reservations.read'),
    ('inventory', 'operator.shortages.read'),
    ('procurement', 'procurement.read'),
    ('procurement', 'procurement.write'),
    ('procurement', 'procurement.sync'),
    ('receiving_inspector', 'inspector.receive'),
    ('receiving_inspector', 'inventory.read')
ON CONFLICT DO NOTHING;

INSERT INTO user_status_history (id, user_id, status, reason, changed_by, created_at)
SELECT
    'user-status-' || REPLACE(u.id::text, '-', ''),
    u.id,
    u.status,
    '',
    'migration',
    COALESCE(u.created_at, NOW())
FROM app_users u
ON CONFLICT (id) DO NOTHING;
