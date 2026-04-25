package inventory

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"backend/internal/testseed"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ResetTestData(ctx context.Context) error {
	return testseed.ResetDatabase(ctx, r.db)
}

func (r *Repository) Dashboard(ctx context.Context) (DashboardData, error) {
	var shortageCount int
	if err := r.db.QueryRowContext(ctx, `
		WITH reservation_totals AS (
			SELECT item_id, SUM(quantity) AS reserved_quantity
			FROM reservations
			WHERE status IN ('requested', 'reserved', 'allocated', 'partially_allocated', 'awaiting_stock')
			GROUP BY item_id
		),
		inventory_totals AS (
			SELECT item_id, COALESCE(SUM(quantity_delta), 0) AS on_hand_quantity
			FROM inventory_events
			GROUP BY item_id
		)
		SELECT COUNT(*)
		FROM reservation_totals r
		LEFT JOIN inventory_totals i ON i.item_id = r.item_id
		WHERE r.reserved_quantity > COALESCE(i.on_hand_quantity, 0)
	`).Scan(&shortageCount); err != nil {
		return DashboardData{}, fmt.Errorf("query shortage count: %w", err)
	}

	var reservationCount int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reservations WHERE status IN ('requested', 'reserved', 'allocated', 'partially_allocated', 'awaiting_stock')
	`).Scan(&reservationCount); err != nil {
		return DashboardData{}, fmt.Errorf("query reservation count: %w", err)
	}

	var importPending int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM import_jobs WHERE status = 'pending'
	`).Scan(&importPending); err != nil {
		return DashboardData{}, fmt.Errorf("query import pending count: %w", err)
	}

	shortages, err := r.Shortages(ctx, "", "")
	if err != nil {
		return DashboardData{}, err
	}

	alerts := make([]string, 0, len(shortages.Rows))
	for _, row := range shortages.Rows {
		alerts = append(alerts, fmt.Sprintf("Device %s / Scope %s is short %d of %s.", row.Device, row.Scope, row.Quantity, row.ItemNumber))
	}
	if len(alerts) == 0 {
		alerts = append(alerts, "No open shortages in the local projection.")
	}

	return DashboardData{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Metrics: []DashboardMetric{
			{Label: "Open shortages", Value: fmt.Sprintf("%d", shortageCount), Delta: "projection-based"},
			{Label: "Pending reservations", Value: fmt.Sprintf("%d", reservationCount), Delta: "local DB"},
			{Label: "Pending imports", Value: fmt.Sprintf("%d", importPending), Delta: "waiting for review"},
		},
		Alerts: alerts,
	}, nil
}

func (r *Repository) Reservations(ctx context.Context, device, scope string) (ReservationList, error) {
	query := `
		SELECT
			r.id,
			i.canonical_item_number,
			i.description,
			r.quantity,
			ds.device_key,
			ds.scope_key,
			r.status
		FROM reservations r
		JOIN items i ON i.id = r.item_id
		JOIN device_scopes ds ON ds.id = r.device_scope_id
		WHERE ($1 = '' OR ds.device_key = $1)
		  AND ($2 = '' OR ds.scope_key = $2)
		ORDER BY r.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, device, scope)
	if err != nil {
		return ReservationList{}, fmt.Errorf("query reservations: %w", err)
	}
	defer rows.Close()

	out := ReservationList{Rows: []ReservationSummary{}}
	for rows.Next() {
		var row ReservationSummary
		if err := rows.Scan(
			&row.ID,
			&row.ItemNumber,
			&row.Description,
			&row.Quantity,
			&row.Device,
			&row.Scope,
			&row.Status,
		); err != nil {
			return ReservationList{}, fmt.Errorf("scan reservation: %w", err)
		}
		out.Rows = append(out.Rows, row)
	}
	return out, rows.Err()
}

func (r *Repository) InventoryOverview(ctx context.Context) (InventoryOverview, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH inventory_totals AS (
			SELECT item_id, location_code, SUM(quantity_delta) AS on_hand_quantity
			FROM inventory_events
			GROUP BY item_id, location_code
		),
		reservation_totals AS (
			SELECT item_id, SUM(quantity) AS reserved_quantity
			FROM reservations
			WHERE status IN ('requested', 'reserved', 'allocated', 'partially_allocated', 'awaiting_stock')
			GROUP BY item_id
		)
		SELECT
			i.id,
			i.canonical_item_number,
			i.description,
			m.name,
			c.name,
			it.location_code,
			it.on_hand_quantity,
			COALESCE(rt.reserved_quantity, 0) AS reserved_quantity,
			it.on_hand_quantity - COALESCE(rt.reserved_quantity, 0) AS available_quantity
		FROM inventory_totals it
		JOIN items i ON i.id = it.item_id
		JOIN manufacturers m ON m.key = i.manufacturer_key
		JOIN categories c ON c.key = i.category_key
		LEFT JOIN reservation_totals rt ON rt.item_id = i.id
		ORDER BY i.canonical_item_number, it.location_code
	`)
	if err != nil {
		return InventoryOverview{}, fmt.Errorf("query inventory overview: %w", err)
	}
	defer rows.Close()

	out := InventoryOverview{Balances: []InventoryBalance{}}
	for rows.Next() {
		var row InventoryBalance
		if err := rows.Scan(
			&row.ItemID,
			&row.ItemNumber,
			&row.Description,
			&row.Manufacturer,
			&row.Category,
			&row.LocationCode,
			&row.OnHandQuantity,
			&row.ReservedQuantity,
			&row.AvailableQuantity,
		); err != nil {
			return InventoryOverview{}, fmt.Errorf("scan inventory overview: %w", err)
		}
		out.Balances = append(out.Balances, row)
	}
	return out, rows.Err()
}

func (r *Repository) Shortages(ctx context.Context, device, scope string) (ShortageList, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH reservation_totals AS (
			SELECT
				r.item_id,
				ds.device_key,
				ds.scope_key,
				SUM(r.quantity) AS reserved_quantity
			FROM reservations r
			JOIN device_scopes ds ON ds.id = r.device_scope_id
			WHERE r.status IN ('requested', 'reserved', 'allocated', 'partially_allocated', 'awaiting_stock')
			  AND ($1 = '' OR ds.device_key = $1)
			  AND ($2 = '' OR ds.scope_key = $2)
			GROUP BY r.item_id, ds.device_key, ds.scope_key
		),
		inventory_totals AS (
			SELECT item_id, COALESCE(SUM(quantity_delta), 0) AS on_hand_quantity
			FROM inventory_events
			GROUP BY item_id
		)
		SELECT
			rt.device_key,
			rt.scope_key,
			m.name,
			i.canonical_item_number,
			i.description,
			rt.reserved_quantity - COALESCE(it.on_hand_quantity, 0) AS shortage_quantity
		FROM reservation_totals rt
		JOIN items i ON i.id = rt.item_id
		JOIN manufacturers m ON m.key = i.manufacturer_key
		LEFT JOIN inventory_totals it ON it.item_id = rt.item_id
		WHERE rt.reserved_quantity > COALESCE(it.on_hand_quantity, 0)
		ORDER BY rt.device_key, rt.scope_key, i.canonical_item_number
	`, device, scope)
	if err != nil {
		return ShortageList{}, fmt.Errorf("query shortages: %w", err)
	}
	defer rows.Close()

	out := ShortageList{Rows: []ShortageRow{}}
	for rows.Next() {
		var row ShortageRow
		if err := rows.Scan(
			&row.Device,
			&row.Scope,
			&row.Manufacturer,
			&row.ItemNumber,
			&row.Description,
			&row.Quantity,
		); err != nil {
			return ShortageList{}, fmt.Errorf("scan shortage row: %w", err)
		}
		out.Rows = append(out.Rows, row)
	}
	return out, rows.Err()
}

func (r *Repository) Imports(ctx context.Context) (ImportHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, import_type, status, file_name, summary::text, created_at
		FROM import_jobs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return ImportHistory{}, fmt.Errorf("query import jobs: %w", err)
	}
	defer rows.Close()

	out := ImportHistory{Rows: []ImportJob{}}
	for rows.Next() {
		var row ImportJob
		var createdAt time.Time
		if err := rows.Scan(&row.ID, &row.ImportType, &row.Status, &row.FileName, &row.Summary, &createdAt); err != nil {
			return ImportHistory{}, fmt.Errorf("scan import job: %w", err)
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out.Rows = append(out.Rows, row)
	}
	return out, rows.Err()
}

func (r *Repository) MasterSummary(ctx context.Context) (MasterDataSummary, error) {
	var summary MasterDataSummary

	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM items`).Scan(&summary.ItemCount); err != nil {
		return MasterDataSummary{}, fmt.Errorf("count items: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM suppliers`).Scan(&summary.SupplierCount); err != nil {
		return MasterDataSummary{}, fmt.Errorf("count suppliers: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM supplier_item_aliases`).Scan(&summary.AliasCount); err != nil {
		return MasterDataSummary{}, fmt.Errorf("count aliases: %w", err)
	}

	manufacturerRows, err := r.db.QueryContext(ctx, `SELECT name FROM manufacturers ORDER BY name`)
	if err != nil {
		return MasterDataSummary{}, fmt.Errorf("list manufacturers: %w", err)
	}
	defer manufacturerRows.Close()
	summary.Manufacturers = []string{}
	for manufacturerRows.Next() {
		var name string
		if err := manufacturerRows.Scan(&name); err != nil {
			return MasterDataSummary{}, fmt.Errorf("scan manufacturer: %w", err)
		}
		summary.Manufacturers = append(summary.Manufacturers, name)
	}

	categoryRows, err := r.db.QueryContext(ctx, `SELECT key, name FROM categories ORDER BY name`)
	if err != nil {
		return MasterDataSummary{}, fmt.Errorf("list categories: %w", err)
	}
	defer categoryRows.Close()
	summary.Categories = []CategorySummary{}
	for categoryRows.Next() {
		var category CategorySummary
		if err := categoryRows.Scan(&category.Key, &category.Name); err != nil {
			return MasterDataSummary{}, fmt.Errorf("scan category: %w", err)
		}
		summary.Categories = append(summary.Categories, category)
	}

	supplierRows, err := r.db.QueryContext(ctx, `SELECT id, name FROM suppliers ORDER BY name`)
	if err != nil {
		return MasterDataSummary{}, fmt.Errorf("list suppliers: %w", err)
	}
	defer supplierRows.Close()
	summary.Suppliers = []SupplierSummary{}
	for supplierRows.Next() {
		var supplier SupplierSummary
		if err := supplierRows.Scan(&supplier.ID, &supplier.Name); err != nil {
			return MasterDataSummary{}, fmt.Errorf("scan supplier: %w", err)
		}
		summary.Suppliers = append(summary.Suppliers, supplier)
	}

	itemRows, err := r.db.QueryContext(ctx, `
		SELECT i.canonical_item_number, i.description, m.name, c.name, COALESCE(s.name, '')
		FROM items i
		JOIN manufacturers m ON m.key = i.manufacturer_key
		JOIN categories c ON c.key = i.category_key
		LEFT JOIN suppliers s ON s.id = i.default_supplier_id
		ORDER BY i.created_at DESC
		LIMIT 5
	`)
	if err != nil {
		return MasterDataSummary{}, fmt.Errorf("list items: %w", err)
	}
	defer itemRows.Close()
	summary.RecentItems = []MasterItem{}
	for itemRows.Next() {
		var item MasterItem
		if err := itemRows.Scan(&item.ItemNumber, &item.Description, &item.Manufacturer, &item.Category, &item.Supplier); err != nil {
			return MasterDataSummary{}, fmt.Errorf("scan item: %w", err)
		}
		summary.RecentItems = append(summary.RecentItems, item)
	}

	aliasRows, err := r.db.QueryContext(ctx, `
		SELECT sia.id, s.id, s.name, i.id, i.canonical_item_number, sia.supplier_item_number, sia.units_per_order
		FROM supplier_item_aliases sia
		JOIN suppliers s ON s.id = sia.supplier_id
		JOIN items i ON i.id = sia.item_id
		ORDER BY sia.created_at DESC
		LIMIT 20
	`)
	if err != nil {
		return MasterDataSummary{}, fmt.Errorf("list aliases: %w", err)
	}
	defer aliasRows.Close()
	summary.Aliases = []SupplierAliasSummary{}
	for aliasRows.Next() {
		var alias SupplierAliasSummary
		if err := aliasRows.Scan(
			&alias.ID,
			&alias.SupplierID,
			&alias.SupplierName,
			&alias.ItemID,
			&alias.CanonicalItemNumber,
			&alias.SupplierItemNumber,
			&alias.UnitsPerOrder,
		); err != nil {
			return MasterDataSummary{}, fmt.Errorf("scan alias: %w", err)
		}
		summary.Aliases = append(summary.Aliases, alias)
	}

	importRows, err := r.db.QueryContext(ctx, `
		SELECT file_name FROM import_jobs ORDER BY created_at DESC LIMIT 5
	`)
	if err != nil {
		return MasterDataSummary{}, fmt.Errorf("list import files: %w", err)
	}
	defer importRows.Close()
	summary.RecentImportFiles = []string{}
	for importRows.Next() {
		var fileName string
		if err := importRows.Scan(&fileName); err != nil {
			return MasterDataSummary{}, fmt.Errorf("scan import file: %w", err)
		}
		summary.RecentImportFiles = append(summary.RecentImportFiles, fileName)
	}

	return summary, nil
}

func (r *Repository) ShortageCSV(ctx context.Context, device, scope string) (string, error) {
	shortages, err := r.Shortages(ctx, device, scope)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"device", "scope", "manufacturer", "item_number", "description", "quantity"}); err != nil {
		return "", fmt.Errorf("write shortage csv header: %w", err)
	}
	for _, row := range shortages.Rows {
		if err := writer.Write([]string{
			row.Device,
			row.Scope,
			row.Manufacturer,
			row.ItemNumber,
			row.Description,
			strconv.Itoa(row.Quantity),
		}); err != nil {
			return "", fmt.Errorf("write shortage csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("flush shortage csv: %w", err)
	}
	return buffer.String(), nil
}

func (r *Repository) ExportMasterCSV(ctx context.Context, exportType string) (string, error) {
	switch strings.TrimSpace(exportType) {
	case "items_with_aliases":
		return r.exportItemsWithAliasesCSV(ctx)
	case "items":
		return r.exportItemsCSV(ctx)
	case "aliases":
		return r.exportAliasesCSV(ctx)
	default:
		return "", fmt.Errorf("unsupported export type: %s", exportType)
	}
}

func (r *Repository) ImportMasterCSV(ctx context.Context, importType, fileName string, body io.Reader) (ImportJob, error) {
	records, err := csv.NewReader(body).ReadAll()
	if err != nil {
		job, jobErr := r.recordImportFailure(ctx, importType, fileName, err)
		if jobErr != nil {
			return ImportJob{}, jobErr
		}
		return job, fmt.Errorf("read import csv: %w", err)
	}
	if len(records) < 1 {
		job, jobErr := r.recordImportFailure(ctx, importType, fileName, fmt.Errorf("csv header is required"))
		if jobErr != nil {
			return ImportJob{}, jobErr
		}
		return job, fmt.Errorf("csv header is required")
	}

	headers := normalizeCSVHeaders(records[0])
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ImportJob{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	inserted, updated, err := 0, 0, error(nil)
	summary := map[string]int{}
	switch strings.TrimSpace(importType) {
	case "items_with_aliases":
		summary, err = r.importItemsWithAliasesCSVTx(ctx, tx, headers, records[1:])
	case "items":
		inserted, updated, err = r.importItemsCSVTx(ctx, tx, headers, records[1:])
		summary = map[string]int{"inserted": inserted, "updated": updated}
	case "aliases":
		inserted, updated, err = r.importAliasesCSVTx(ctx, tx, headers, records[1:])
		summary = map[string]int{"inserted": inserted, "updated": updated}
	default:
		err = fmt.Errorf("unsupported import type: %s", importType)
	}
	if err != nil {
		_ = tx.Rollback()
		job, jobErr := r.recordImportFailure(ctx, importType, fileName, err)
		if jobErr != nil {
			return ImportJob{}, jobErr
		}
		return job, err
	}

	summaryBytes, _ := json.Marshal(summary)
	jobID := fmt.Sprintf("imp-%d", time.Now().UnixNano())
	createdAt := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO import_jobs (id, import_type, status, file_name, summary, created_at)
		VALUES ($1, $2, 'completed', $3, $4::jsonb, $5)
	`, jobID, importType, fileName, string(summaryBytes), createdAt); err != nil {
		return ImportJob{}, fmt.Errorf("insert import job: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ImportJob{}, fmt.Errorf("commit import tx: %w", err)
	}
	return ImportJob{
		ID:         jobID,
		ImportType: importType,
		Status:     "completed",
		FileName:   fileName,
		Summary:    string(summaryBytes),
		CreatedAt:  createdAt.Format(time.RFC3339),
	}, nil
}

func (r *Repository) exportItemsCSV(ctx context.Context) (string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			i.canonical_item_number,
			i.description,
			m.name,
			c.name,
			COALESCE(s.id, ''),
			i.note
		FROM items i
		JOIN manufacturers m ON m.key = i.manufacturer_key
		JOIN categories c ON c.key = i.category_key
		LEFT JOIN suppliers s ON s.id = i.default_supplier_id
		ORDER BY i.canonical_item_number
	`)
	if err != nil {
		return "", fmt.Errorf("query items export: %w", err)
	}
	defer rows.Close()

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"canonical_item_number", "description", "manufacturer", "category", "default_supplier_id", "note"}); err != nil {
		return "", fmt.Errorf("write item csv header: %w", err)
	}
	for rows.Next() {
		var itemNumber, description, manufacturer, category, supplierID, note string
		if err := rows.Scan(&itemNumber, &description, &manufacturer, &category, &supplierID, &note); err != nil {
			return "", fmt.Errorf("scan items export row: %w", err)
		}
		if err := writer.Write([]string{itemNumber, description, manufacturer, category, supplierID, note}); err != nil {
			return "", fmt.Errorf("write items export row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("flush items export: %w", err)
	}
	return buffer.String(), rows.Err()
}

func (r *Repository) exportItemsWithAliasesCSV(ctx context.Context) (string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			i.canonical_item_number,
			i.description,
			m.name,
			c.name,
			COALESCE(i.default_supplier_id, ''),
			COALESCE(sia.supplier_id, ''),
			COALESCE(s.name, ''),
			COALESCE(sia.supplier_item_number, ''),
			COALESCE(sia.units_per_order, 1),
			i.note
		FROM items i
		JOIN manufacturers m ON m.key = i.manufacturer_key
		JOIN categories c ON c.key = i.category_key
		LEFT JOIN supplier_item_aliases sia ON sia.item_id = i.id
		LEFT JOIN suppliers s ON s.id = sia.supplier_id
		ORDER BY i.canonical_item_number, s.name, sia.supplier_item_number
	`)
	if err != nil {
		return "", fmt.Errorf("query items with aliases export: %w", err)
	}
	defer rows.Close()

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{
		"canonical_item_number",
		"description",
		"manufacturer",
		"category",
		"default_supplier_id",
		"supplier_id",
		"supplier_name",
		"supplier_item_number",
		"units_per_order",
		"note",
	}); err != nil {
		return "", fmt.Errorf("write items with aliases csv header: %w", err)
	}
	for rows.Next() {
		var itemNumber, description, manufacturer, category, defaultSupplierID, supplierID, supplierName, aliasNumber, note string
		var unitsPerOrder int
		if err := rows.Scan(&itemNumber, &description, &manufacturer, &category, &defaultSupplierID, &supplierID, &supplierName, &aliasNumber, &unitsPerOrder, &note); err != nil {
			return "", fmt.Errorf("scan items with aliases export row: %w", err)
		}
		units := ""
		if aliasNumber != "" {
			units = strconv.Itoa(unitsPerOrder)
		}
		if err := writer.Write([]string{itemNumber, description, manufacturer, category, defaultSupplierID, supplierID, supplierName, aliasNumber, units, note}); err != nil {
			return "", fmt.Errorf("write items with aliases export row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("flush items with aliases csv: %w", err)
	}
	return buffer.String(), rows.Err()
}

func (r *Repository) exportAliasesCSV(ctx context.Context) (string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			sia.supplier_id,
			s.name,
			i.canonical_item_number,
			sia.supplier_item_number,
			sia.units_per_order
		FROM supplier_item_aliases sia
		JOIN suppliers s ON s.id = sia.supplier_id
		JOIN items i ON i.id = sia.item_id
		ORDER BY s.name, sia.supplier_item_number
	`)
	if err != nil {
		return "", fmt.Errorf("query aliases export: %w", err)
	}
	defer rows.Close()

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"supplier_id", "supplier_name", "canonical_item_number", "supplier_item_number", "units_per_order"}); err != nil {
		return "", fmt.Errorf("write alias csv header: %w", err)
	}
	for rows.Next() {
		var supplierID, supplierName, itemNumber, aliasNumber string
		var unitsPerOrder int
		if err := rows.Scan(&supplierID, &supplierName, &itemNumber, &aliasNumber, &unitsPerOrder); err != nil {
			return "", fmt.Errorf("scan aliases export row: %w", err)
		}
		if err := writer.Write([]string{supplierID, supplierName, itemNumber, aliasNumber, strconv.Itoa(unitsPerOrder)}); err != nil {
			return "", fmt.Errorf("write aliases export row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("flush aliases export: %w", err)
	}
	return buffer.String(), rows.Err()
}

func (r *Repository) importItemsWithAliasesCSVTx(ctx context.Context, tx *sql.Tx, headers []string, rows [][]string) (map[string]int, error) {
	summary := map[string]int{
		"item_inserted":  0,
		"item_updated":   0,
		"alias_inserted": 0,
		"alias_updated":  0,
		"alias_only":     0,
	}
	for _, row := range rows {
		record := csvRecord(headers, row)
		itemNumber := strings.TrimSpace(record["canonical_item_number"])
		description := strings.TrimSpace(record["description"])
		manufacturerName := strings.TrimSpace(record["manufacturer"])
		categoryName := strings.TrimSpace(record["category"])
		defaultSupplierID := strings.TrimSpace(record["default_supplier_id"])
		supplierID := strings.TrimSpace(record["supplier_id"])
		aliasNumber := strings.TrimSpace(record["supplier_item_number"])
		note := strings.TrimSpace(record["note"])
		if itemNumber == "" {
			return nil, fmt.Errorf("items_with_aliases csv requires canonical_item_number")
		}

		hasDescription := description != ""
		hasManufacturer := manufacturerName != ""
		hasCategory := categoryName != ""
		hasItemFields := hasDescription || hasManufacturer || hasCategory
		hasCompleteItem := hasDescription && hasManufacturer && hasCategory
		if hasItemFields && !hasCompleteItem {
			return nil, fmt.Errorf("items_with_aliases csv requires description, manufacturer, and category together for item upsert: %s", itemNumber)
		}

		aliasSupplierID := supplierID
		if aliasSupplierID == "" {
			aliasSupplierID = defaultSupplierID
		}
		hasAliasFields := aliasNumber != "" || supplierID != "" || strings.TrimSpace(record["units_per_order"]) != ""
		if !hasCompleteItem && !hasAliasFields {
			return nil, fmt.Errorf("items_with_aliases csv row must include item fields or supplier alias fields: %s", itemNumber)
		}
		if hasAliasFields && (aliasSupplierID == "" || aliasNumber == "") {
			return nil, fmt.Errorf("items_with_aliases alias rows require supplier_id or default_supplier_id, and supplier_item_number: %s", itemNumber)
		}

		itemID := ""
		if hasCompleteItem {
			if defaultSupplierID != "" {
				if err := r.ensureSupplierExistsTx(ctx, tx, defaultSupplierID); err != nil {
					return nil, err
				}
			}
			manufacturerKey := normalizeLookupKey(manufacturerName)
			categoryKey := normalizeLookupKey(categoryName)
			if err := r.upsertManufacturerTx(ctx, tx, manufacturerKey, manufacturerName); err != nil {
				return nil, err
			}
			if err := r.upsertCategoryTx(ctx, tx, categoryKey, categoryName); err != nil {
				return nil, err
			}

			err := tx.QueryRowContext(ctx, `SELECT id FROM items WHERE canonical_item_number = $1`, itemNumber).Scan(&itemID)
			if err != nil && err != sql.ErrNoRows {
				return nil, fmt.Errorf("query import item: %w", err)
			}
			if itemID == "" {
				itemID = fmt.Sprintf("item-%d", time.Now().UnixNano())
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description, default_supplier_id, note, active)
					VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, TRUE)
				`, itemID, manufacturerKey, categoryKey, itemNumber, description, defaultSupplierID, note); err != nil {
					return nil, fmt.Errorf("insert import item: %w", err)
				}
				summary["item_inserted"]++
			} else {
				if _, err := tx.ExecContext(ctx, `
					UPDATE items
					SET manufacturer_key = $2,
					    category_key = $3,
					    description = $4,
					    default_supplier_id = NULLIF($5, ''),
					    note = $6
					WHERE id = $1
				`, itemID, manufacturerKey, categoryKey, description, defaultSupplierID, note); err != nil {
					return nil, fmt.Errorf("update import item: %w", err)
				}
				summary["item_updated"]++
			}
		} else {
			if err := tx.QueryRowContext(ctx, `SELECT id FROM items WHERE canonical_item_number = $1`, itemNumber).Scan(&itemID); err != nil {
				if err == sql.ErrNoRows {
					return nil, fmt.Errorf("canonical item not found for alias-only row: %s", itemNumber)
				}
				return nil, fmt.Errorf("query alias-only item: %w", err)
			}
			summary["alias_only"]++
		}

		if !hasAliasFields {
			continue
		}
		if err := r.ensureSupplierExistsTx(ctx, tx, aliasSupplierID); err != nil {
			return nil, err
		}
		unitsPerOrder := parsePositiveInt(record["units_per_order"], 1)
		inserted, err := r.upsertAliasCSVTx(ctx, tx, itemID, aliasSupplierID, aliasNumber, unitsPerOrder)
		if err != nil {
			return nil, err
		}
		if inserted {
			summary["alias_inserted"]++
		} else {
			summary["alias_updated"]++
		}
	}
	return summary, nil
}

func (r *Repository) importItemsCSVTx(ctx context.Context, tx *sql.Tx, headers []string, rows [][]string) (int, int, error) {
	inserted, updated := 0, 0
	for _, row := range rows {
		record := csvRecord(headers, row)
		itemNumber := strings.TrimSpace(record["canonical_item_number"])
		description := strings.TrimSpace(record["description"])
		manufacturerName := strings.TrimSpace(record["manufacturer"])
		categoryName := strings.TrimSpace(record["category"])
		supplierID := strings.TrimSpace(record["default_supplier_id"])
		note := strings.TrimSpace(record["note"])
		if itemNumber == "" || description == "" || manufacturerName == "" || categoryName == "" {
			return 0, 0, fmt.Errorf("items csv requires canonical_item_number, description, manufacturer, and category")
		}
		manufacturerKey := normalizeLookupKey(manufacturerName)
		categoryKey := normalizeLookupKey(categoryName)
		if err := r.upsertManufacturerTx(ctx, tx, manufacturerKey, manufacturerName); err != nil {
			return 0, 0, err
		}
		if err := r.upsertCategoryTx(ctx, tx, categoryKey, categoryName); err != nil {
			return 0, 0, err
		}
		if supplierID != "" {
			if err := r.ensureSupplierExistsTx(ctx, tx, supplierID); err != nil {
				return 0, 0, err
			}
		}

		var existingID string
		err := tx.QueryRowContext(ctx, `SELECT id FROM items WHERE canonical_item_number = $1`, itemNumber).Scan(&existingID)
		if err != nil && err != sql.ErrNoRows {
			return 0, 0, fmt.Errorf("query import item: %w", err)
		}
		if existingID == "" {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO items (id, manufacturer_key, category_key, canonical_item_number, description, default_supplier_id, note, active)
				VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, TRUE)
			`, fmt.Sprintf("item-%d", time.Now().UnixNano()), manufacturerKey, categoryKey, itemNumber, description, supplierID, note); err != nil {
				return 0, 0, fmt.Errorf("insert import item: %w", err)
			}
			inserted++
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE items
			SET manufacturer_key = $2,
			    category_key = $3,
			    description = $4,
			    default_supplier_id = NULLIF($5, ''),
			    note = $6
			WHERE id = $1
		`, existingID, manufacturerKey, categoryKey, description, supplierID, note); err != nil {
			return 0, 0, fmt.Errorf("update import item: %w", err)
		}
		updated++
	}
	return inserted, updated, nil
}

func (r *Repository) importAliasesCSVTx(ctx context.Context, tx *sql.Tx, headers []string, rows [][]string) (int, int, error) {
	inserted, updated := 0, 0
	for _, row := range rows {
		record := csvRecord(headers, row)
		supplierID := strings.TrimSpace(record["supplier_id"])
		itemNumber := strings.TrimSpace(record["canonical_item_number"])
		aliasNumber := strings.TrimSpace(record["supplier_item_number"])
		unitsPerOrder := parsePositiveInt(record["units_per_order"], 1)
		if supplierID == "" || itemNumber == "" || aliasNumber == "" {
			return 0, 0, fmt.Errorf("aliases csv requires supplier_id, canonical_item_number, and supplier_item_number")
		}
		if err := r.ensureSupplierExistsTx(ctx, tx, supplierID); err != nil {
			return 0, 0, err
		}
		var itemID string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM items WHERE canonical_item_number = $1`, itemNumber).Scan(&itemID); err != nil {
			if err == sql.ErrNoRows {
				return 0, 0, fmt.Errorf("canonical item not found for alias import: %s", itemNumber)
			}
			return 0, 0, fmt.Errorf("query alias item: %w", err)
		}
		insertedAlias, err := r.upsertAliasCSVTx(ctx, tx, itemID, supplierID, aliasNumber, unitsPerOrder)
		if err != nil {
			return 0, 0, err
		}
		if insertedAlias {
			inserted++
			continue
		}
		updated++
	}
	return inserted, updated, nil
}

func (r *Repository) upsertAliasCSVTx(ctx context.Context, tx *sql.Tx, itemID, supplierID, aliasNumber string, unitsPerOrder int) (bool, error) {
	var existingAliasID, existingItemID string
	err := tx.QueryRowContext(ctx, `
		SELECT id, item_id
		FROM supplier_item_aliases
		WHERE supplier_id = $1 AND supplier_item_number = $2
	`, supplierID, aliasNumber).Scan(&existingAliasID, &existingItemID)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("query alias duplicate: %w", err)
	}
	if existingAliasID == "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO supplier_item_aliases (id, item_id, supplier_id, supplier_item_number, units_per_order)
			VALUES ($1, $2, $3, $4, $5)
		`, fmt.Sprintf("alias-%d", time.Now().UnixNano()), itemID, supplierID, aliasNumber, unitsPerOrder); err != nil {
			return false, fmt.Errorf("insert alias import: %w", err)
		}
		return true, nil
	}
	if existingItemID != itemID {
		return false, fmt.Errorf("supplier alias already belongs to another item: %s", aliasNumber)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE supplier_item_aliases
		SET units_per_order = $2
		WHERE id = $1
	`, existingAliasID, unitsPerOrder); err != nil {
		return false, fmt.Errorf("update alias import: %w", err)
	}
	return false, nil
}

func (r *Repository) recordImportFailure(ctx context.Context, importType, fileName string, importErr error) (ImportJob, error) {
	jobID := fmt.Sprintf("imp-%d", time.Now().UnixNano())
	createdAt := time.Now().UTC()
	summaryBytes, _ := json.Marshal(map[string]string{"error": importErr.Error()})
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO import_jobs (id, import_type, status, file_name, summary, created_at)
		VALUES ($1, $2, 'failed', $3, $4::jsonb, $5)
	`, jobID, defaultString(importType, "unknown"), defaultString(fileName, "unknown.csv"), string(summaryBytes), createdAt); err != nil {
		return ImportJob{}, fmt.Errorf("record failed import job: %w", err)
	}
	return ImportJob{
		ID:         jobID,
		ImportType: importType,
		Status:     "failed",
		FileName:   fileName,
		Summary:    string(summaryBytes),
		CreatedAt:  createdAt.Format(time.RFC3339),
	}, nil
}

func (r *Repository) upsertManufacturerTx(ctx context.Context, tx *sql.Tx, key, name string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO manufacturers (key, name)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET name = EXCLUDED.name
	`, key, name); err != nil {
		return fmt.Errorf("upsert manufacturer: %w", err)
	}
	return nil
}

func (r *Repository) upsertCategoryTx(ctx context.Context, tx *sql.Tx, key, name string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO categories (key, name)
		VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET name = EXCLUDED.name
	`, key, name); err != nil {
		return fmt.Errorf("upsert category: %w", err)
	}
	return nil
}

func (r *Repository) ensureSupplierExistsTx(ctx context.Context, tx *sql.Tx, supplierID string) error {
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM suppliers WHERE id = $1`, supplierID).Scan(&exists); err != nil {
		return fmt.Errorf("query supplier: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("supplier not found: %s", supplierID)
	}
	return nil
}

func csvRecord(headers, row []string) map[string]string {
	record := map[string]string{}
	for index, header := range headers {
		if index < len(row) {
			record[header] = strings.TrimSpace(row[index])
		}
	}
	return record
}

func normalizeCSVHeaders(headers []string) []string {
	out := make([]string, 0, len(headers))
	for _, header := range headers {
		out = append(out, strings.ToLower(strings.TrimSpace(header)))
	}
	return out
}

func normalizeLookupKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "-", " ", "-")
	value = replacer.Replace(value)
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-':
			builder.WriteRune(r)
		}
	}
	return strings.Trim(builder.String(), "-")
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (r *Repository) CreateReservation(ctx context.Context, input ReservationCreateInput) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reservation tx: %w", err)
	}
	defer tx.Rollback()

	id := fmt.Sprintf("res-%d", time.Now().UnixNano())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO reservations (
			id, item_id, device_scope_id, quantity, status, requested_by, note, purpose, priority,
			needed_by_at, planned_use_at, hold_until_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'requested', $5, $6, $7, $8, NULLIF($9, '')::timestamptz, NULLIF($10, '')::timestamptz, NULLIF($11, '')::timestamptz, NOW(), NOW())
	`,
		id,
		input.ItemID,
		input.DeviceScopeID,
		input.Quantity,
		emptyDefault(input.RequestedBy, "local-user"),
		input.Note,
		input.Purpose,
		defaultString(input.Priority, "normal"),
		input.NeededByAt,
		input.PlannedUseAt,
		input.HoldUntilAt,
	); err != nil {
		return fmt.Errorf("insert reservation: %w", err)
	}
	if err := r.recordReservationEventTx(ctx, tx, id, "requested", input.Quantity, input.RequestedBy, map[string]any{
		"note": input.Note,
	}); err != nil {
		return err
	}
	if err := recordAuditEventTx(ctx, tx, input.RequestedBy, "reservation.created", "reservation", id, map[string]any{
		"itemId":   input.ItemID,
		"scopeId":  input.DeviceScopeID,
		"quantity": input.Quantity,
	}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reservation tx: %w", err)
	}
	return nil
}

func (r *Repository) AdjustInventory(ctx context.Context, input InventoryAdjustInput) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin inventory adjustment tx: %w", err)
	}
	defer tx.Rollback()

	if input.QuantityDelta > 0 {
		if err := incrementBalanceTx(ctx, tx, input.ItemID, input.LocationCode, input.QuantityDelta); err != nil {
			return err
		}
	} else {
		if err := decrementBalanceTx(ctx, tx, input.ItemID, input.LocationCode, -input.QuantityDelta); err != nil {
			return err
		}
	}
	if _, err := insertInventoryEventTx(ctx, tx, inventoryEventInsert{
		ItemID:           input.ItemID,
		LocationCode:     input.LocationCode,
		FromLocationCode: input.LocationCode,
		ToLocationCode:   input.LocationCode,
		EventType:        "adjust",
		QuantityDelta:    input.QuantityDelta,
		DeviceScopeID:    input.DeviceScopeID,
		SourceType:       "manual",
		Note:             input.Note,
	}); err != nil {
		return err
	}
	if err := recordAuditEventTx(ctx, tx, "", "inventory.adjusted", "item", input.ItemID, map[string]any{
		"locationCode":  input.LocationCode,
		"quantityDelta": input.QuantityDelta,
	}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit inventory adjustment tx: %w", err)
	}
	return nil
}

func emptyDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
