-- 000011_additional_spec_042401.up.sql
-- Additional specification: reservation-order linking, search indexes, and supporting structures.

-- Enable trigram extension for fuzzy text search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

------------------------------------------------------------
-- 1. Reservation-Order linking via reservation_allocations
------------------------------------------------------------

ALTER TABLE reservation_allocations
    ADD COLUMN IF NOT EXISTS purchase_order_line_id TEXT
        REFERENCES purchase_order_lines(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT 'stock';

COMMENT ON COLUMN reservation_allocations.source_type IS 'stock: allocated from existing inventory; incoming_order: allocated from expected PO arrival';
COMMENT ON COLUMN reservation_allocations.purchase_order_line_id IS 'When source_type=incoming_order, references the PO line providing the expected stock';

INSERT INTO locations (code, name, location_type)
VALUES ('INCOMING', 'Incoming Purchase Orders', 'virtual')
ON CONFLICT (code) DO NOTHING;

------------------------------------------------------------
-- 2. Trigram indexes for item search / typeahead
------------------------------------------------------------

CREATE INDEX IF NOT EXISTS idx_items_item_number_trgm
    ON items USING gin (canonical_item_number gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_items_description_trgm
    ON items USING gin (description gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_manufacturers_name_trgm
    ON manufacturers USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_categories_name_trgm
    ON categories USING gin (name gin_trgm_ops);

------------------------------------------------------------
-- 3. Performance indexes for new query patterns
------------------------------------------------------------

-- Item flow: events by item ordered by time
CREATE INDEX IF NOT EXISTS idx_inventory_events_item_occurred
    ON inventory_events (item_id, occurred_at DESC);

-- Arrival calendar: open PO lines with expected dates
CREATE INDEX IF NOT EXISTS idx_po_lines_expected_arrival
    ON purchase_order_lines (expected_arrival_date)
    WHERE expected_arrival_date IS NOT NULL
      AND status NOT IN ('cancelled', 'received');

-- Scope requirements lookup
CREATE INDEX IF NOT EXISTS idx_scope_item_requirements_scope
    ON scope_item_requirements (scope_id);

-- Reservation lookup by scope
CREATE INDEX IF NOT EXISTS idx_reservations_scope
    ON reservations (device_scope_id)
    WHERE status NOT IN ('cancelled', 'released');

-- Reservation allocations by PO line (for order-linked queries)
CREATE INDEX IF NOT EXISTS idx_reservation_alloc_po_line
    ON reservation_allocations (purchase_order_line_id)
    WHERE purchase_order_line_id IS NOT NULL;
