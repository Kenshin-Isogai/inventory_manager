ALTER TABLE app_users
    ADD COLUMN IF NOT EXISTS identity_provider TEXT,
    ADD COLUMN IF NOT EXISTS identity_subject TEXT,
    ADD COLUMN IF NOT EXISTS rejection_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE UNIQUE INDEX IF NOT EXISTS app_users_identity_subject_unique
    ON app_users (identity_provider, identity_subject)
    WHERE identity_provider IS NOT NULL AND identity_subject IS NOT NULL;

INSERT INTO app_users (id, email, display_name, status, identity_provider, identity_subject, updated_at)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin@example.local', 'Local Admin', 'active', 'local', 'seed-admin', NOW()),
    ('00000000-0000-0000-0000-000000000002', 'operator@example.local', 'Local Operator', 'active', 'local', 'seed-operator', NOW()),
    ('00000000-0000-0000-0000-000000000003', 'inventory@example.local', 'Local Inventory', 'active', 'local', 'seed-inventory', NOW()),
    ('00000000-0000-0000-0000-000000000004', 'procurement@example.local', 'Local Procurement', 'active', 'local', 'seed-procurement', NOW()),
    ('00000000-0000-0000-0000-000000000005', 'inspector@example.local', 'Local Inspector', 'active', 'local', 'seed-inspector', NOW())
ON CONFLICT (email) DO UPDATE
SET display_name = EXCLUDED.display_name,
    identity_provider = COALESCE(app_users.identity_provider, EXCLUDED.identity_provider),
    identity_subject = COALESCE(app_users.identity_subject, EXCLUDED.identity_subject),
    updated_at = NOW();

INSERT INTO user_roles (user_id, role_key) VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin'),
    ('00000000-0000-0000-0000-000000000001', 'operator'),
    ('00000000-0000-0000-0000-000000000001', 'inventory'),
    ('00000000-0000-0000-0000-000000000001', 'procurement'),
    ('00000000-0000-0000-0000-000000000002', 'operator'),
    ('00000000-0000-0000-0000-000000000003', 'inventory'),
    ('00000000-0000-0000-0000-000000000004', 'procurement'),
    ('00000000-0000-0000-0000-000000000005', 'receiving_inspector')
ON CONFLICT DO NOTHING;
