INSERT INTO departments (key, name) VALUES
    ('optics', 'Optics'),
    ('mechanical', 'Mechanical'),
    ('controls', 'Controls')
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS scope_systems (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE device_scopes
    ADD COLUMN IF NOT EXISTS system_key TEXT REFERENCES scope_systems(key) ON DELETE RESTRICT;

INSERT INTO scope_systems (key, name, description, status) VALUES
    ('optics', 'Optical System', 'Optical engineering system boundary', 'active'),
    ('mechanical', 'Mechanical System', 'Mechanical engineering system boundary', 'active'),
    ('controls', 'Control System', 'Control engineering system boundary', 'active')
ON CONFLICT (key) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    status = EXCLUDED.status,
    updated_at = NOW();

UPDATE device_scopes
SET scope_type = 'assembly',
    scope_name = COALESCE(NULLIF(scope_name, ''), scope_key)
WHERE COALESCE(scope_type, '') = '';

INSERT INTO device_scopes (id, device_id, device_key, parent_scope_id, system_key, scope_key, scope_name, scope_type, owner_department_key, status, description, updated_at)
SELECT
    'ds-' || LOWER(d.device_key) || '-optics',
    d.id,
    d.device_key,
    NULL,
    'optics',
    'optics',
    'Optical System',
    'system',
    'optics',
    'active',
    'Optical engineering system boundary',
    NOW()
FROM devices d
ON CONFLICT (device_key, scope_key) DO UPDATE
SET device_id = EXCLUDED.device_id,
    system_key = EXCLUDED.system_key,
    scope_name = EXCLUDED.scope_name,
    scope_type = EXCLUDED.scope_type,
    owner_department_key = EXCLUDED.owner_department_key,
    status = EXCLUDED.status,
    updated_at = NOW();

INSERT INTO device_scopes (id, device_id, device_key, parent_scope_id, system_key, scope_key, scope_name, scope_type, owner_department_key, status, description, updated_at)
SELECT
    'ds-' || LOWER(d.device_key) || '-mechanical',
    d.id,
    d.device_key,
    NULL,
    'mechanical',
    'mechanical',
    'Mechanical System',
    'system',
    'mechanical',
    'active',
    'Mechanical engineering system boundary',
    NOW()
FROM devices d
ON CONFLICT (device_key, scope_key) DO UPDATE
SET device_id = EXCLUDED.device_id,
    system_key = EXCLUDED.system_key,
    scope_name = EXCLUDED.scope_name,
    scope_type = EXCLUDED.scope_type,
    owner_department_key = EXCLUDED.owner_department_key,
    status = EXCLUDED.status,
    updated_at = NOW();

INSERT INTO device_scopes (id, device_id, device_key, parent_scope_id, system_key, scope_key, scope_name, scope_type, owner_department_key, status, description, updated_at)
SELECT
    'ds-' || LOWER(d.device_key) || '-controls',
    d.id,
    d.device_key,
    NULL,
    'controls',
    'controls',
    'Control System',
    'system',
    'controls',
    'active',
    'Control engineering system boundary',
    NOW()
FROM devices d
ON CONFLICT (device_key, scope_key) DO UPDATE
SET device_id = EXCLUDED.device_id,
    system_key = EXCLUDED.system_key,
    scope_name = EXCLUDED.scope_name,
    scope_type = EXCLUDED.scope_type,
    owner_department_key = EXCLUDED.owner_department_key,
    status = EXCLUDED.status,
    updated_at = NOW();

UPDATE device_scopes
SET owner_department_key = CASE
        WHEN scope_key ILIKE '%power%' OR scope_key ILIKE '%control%' THEN 'controls'
        WHEN scope_key ILIKE '%optic%' OR scope_key ILIKE '%lens%' OR scope_key ILIKE '%camera%' THEN 'optics'
        WHEN COALESCE(owner_department_key, '') = '' THEN 'mechanical'
        ELSE owner_department_key
    END
WHERE scope_type <> 'system';

UPDATE device_scopes
SET system_key = CASE
        WHEN scope_type = 'system' AND scope_key IN ('optics', 'mechanical', 'controls') THEN scope_key
        WHEN owner_department_key IN ('optics', 'mechanical', 'controls') THEN owner_department_key
        ELSE 'mechanical'
    END
WHERE system_key IS NULL;

UPDATE device_scopes child
SET parent_scope_id = parent.id
FROM device_scopes parent
WHERE child.parent_scope_id IS NULL
  AND child.scope_type <> 'system'
  AND parent.device_key = child.device_key
  AND parent.scope_key = COALESCE(NULLIF(child.system_key, ''), 'mechanical');

ALTER TABLE device_scopes
    ADD CONSTRAINT device_scopes_system_requires_key_chk
    CHECK (scope_type <> 'system' OR COALESCE(system_key, '') <> '');

ALTER TABLE device_scopes
    ADD CONSTRAINT device_scopes_system_is_root_chk
    CHECK (scope_type <> 'system' OR parent_scope_id IS NULL);

ALTER TABLE device_scopes
    ADD CONSTRAINT device_scopes_non_system_requires_parent_chk
    CHECK (scope_type = 'system' OR parent_scope_id IS NOT NULL);

CREATE OR REPLACE FUNCTION enforce_device_scope_hierarchy()
RETURNS TRIGGER AS $$
DECLARE
    parent_record device_scopes%ROWTYPE;
BEGIN
    IF COALESCE(NEW.scope_type, '') = 'system' THEN
        IF COALESCE(NEW.system_key, '') = '' THEN
            RAISE EXCEPTION 'system scopes require system_key';
        END IF;
        IF NEW.parent_scope_id IS NOT NULL THEN
            RAISE EXCEPTION 'system scopes cannot have parent_scope_id';
        END IF;
        IF NEW.scope_key <> NEW.system_key THEN
            RAISE EXCEPTION 'system scope key must match system_key';
        END IF;
        RETURN NEW;
    END IF;

    IF NEW.parent_scope_id IS NULL THEN
        RAISE EXCEPTION 'non-system scopes require parent_scope_id';
    END IF;

    SELECT *
    INTO parent_record
    FROM device_scopes
    WHERE id = NEW.parent_scope_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'parent scope not found: %', NEW.parent_scope_id;
    END IF;

    IF parent_record.device_key <> NEW.device_key THEN
        RAISE EXCEPTION 'parent scope device_key % does not match child device_key %', parent_record.device_key, NEW.device_key;
    END IF;

    IF COALESCE(parent_record.system_key, '') = '' THEN
        RAISE EXCEPTION 'parent scope % is missing system_key', parent_record.id;
    END IF;

    IF COALESCE(NEW.system_key, '') = '' THEN
        NEW.system_key := parent_record.system_key;
    ELSIF NEW.system_key <> parent_record.system_key THEN
        RAISE EXCEPTION 'child scope system_key % must match parent system_key %', NEW.system_key, parent_record.system_key;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_device_scopes_enforce_hierarchy ON device_scopes;

CREATE TRIGGER trg_device_scopes_enforce_hierarchy
BEFORE INSERT OR UPDATE ON device_scopes
FOR EACH ROW
EXECUTE FUNCTION enforce_device_scope_hierarchy();
