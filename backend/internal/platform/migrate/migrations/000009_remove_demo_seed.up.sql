DELETE FROM procurement_dispatch_history
WHERE batch_id IN ('batch-001', 'batch-002');

DELETE FROM procurement_dispatch_outbox
WHERE batch_id IN ('batch-001', 'batch-002');

DELETE FROM purchase_order_lines
WHERE id IN ('po-line-001');

DELETE FROM purchase_orders
WHERE id IN ('po-001');

DELETE FROM procurement_status_history
WHERE id IN ('psh-001', 'psh-002', 'psh-003');

DELETE FROM procurement_status_projections
WHERE batch_id IN ('batch-001', 'batch-002');

DELETE FROM procurement_lines
WHERE id IN ('pline-001', 'pline-002');

DELETE FROM procurement_batches
WHERE id IN ('batch-001', 'batch-002');

DELETE FROM quotation_lines
WHERE id IN ('quote-line-001', 'quote-line-002');

DELETE FROM supplier_quotations
WHERE id IN ('quote-001', 'quote-002');

DELETE FROM external_project_budget_categories
WHERE id IN ('budget-er2-material', 'budget-er2-maintenance', 'budget-mk4-material');

DELETE FROM external_projects
WHERE id IN ('proj-er2-upgrade', 'proj-mk4-refresh');

DELETE FROM import_jobs
WHERE id IN ('imp-001', 'imp-002');

DELETE FROM reservations
WHERE id IN ('res-001', 'res-002');

DELETE FROM inventory_events
WHERE id IN ('evt-001', 'evt-002', 'evt-003', 'evt-004');

DELETE FROM supplier_item_aliases
WHERE id IN ('alias-er2-pack4', 'alias-mk44-bulk');

DELETE FROM device_scopes
WHERE id IN ('ds-er2-powerboard', 'ds-mk4-cabinet');

DELETE FROM items
WHERE id IN ('item-er2', 'item-mk44', 'item-cn88');

DELETE FROM suppliers
WHERE id IN ('sup-thorlabs', 'sup-misumi');

DELETE FROM manufacturers
WHERE key IN ('omron', 'phoenix', 'molex');

DELETE FROM categories
WHERE key IN ('relay', 'terminal', 'connector');

DELETE FROM departments
WHERE key IN ('production', 'procurement');

DELETE FROM app_users
WHERE email IN (
    'admin@example.local',
    'operator@example.local',
    'inventory@example.local',
    'procurement@example.local',
    'inspector@example.local'
);
