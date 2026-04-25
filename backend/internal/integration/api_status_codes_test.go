// Integration tests for API endpoint status codes.
// Tests the full HTTP handler stack using httptest, with a real database.
//
//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/config"
	"backend/internal/httpapi"
	"backend/internal/inventory"
	"backend/internal/testutil"
)

// newTestRouter creates a real HTTP router backed by the test database.
// Auth is disabled so we can test endpoint logic directly.
func newTestRouter(t *testing.T) (http.Handler, *inventory.Service) {
	t.Helper()
	db := testutil.SetupTestDB(t)

	cfg := config.Config{
		App:  config.AppConfig{Name: "test", Env: "test", Mode: "local"},
		HTTP: config.HTTPConfig{AllowedOrigins: []string{"http://localhost:5173"}},
		Auth: config.AuthConfig{Mode: "none", RBAC: "none"},
	}

	logger := slog.Default()
	repo := inventory.NewRepository(db)
	svc := inventory.NewService(repo)

	readyCheck := func(ctx context.Context) error {
		return db.PingContext(ctx)
	}

	router := httpapi.NewRouter(cfg, logger, nil, svc, nil, nil, readyCheck)
	return router, svc
}

// ============================================================
// Health endpoints
// ============================================================

func TestAPI_HealthEndpoint_Returns200(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /health: want 200, got %d", rec.Code)
	}
}

func TestAPI_ReadyEndpoint_Returns200(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /ready: want 200, got %d", rec.Code)
	}
}

// ============================================================
// Inventory: Receive
// ============================================================

func TestAPI_ReceiveInventory_201OnSuccess(t *testing.T) {
	router, _ := newTestRouter(t)

	body, _ := json.Marshal(map[string]any{
		"itemId":       "item-er2",
		"locationCode": "TOKYO-A1",
		"quantity":     5,
	})

	req := httptest.NewRequest("POST", "/api/v1/inventory/receives", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("POST /inventory/receives: want 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestAPI_ReceiveInventory_400OnInvalidJSON(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("POST", "/api/v1/inventory/receives", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /inventory/receives with invalid JSON: want 400, got %d", rec.Code)
	}
}

func TestAPI_ReceiveInventory_400OnMissingFields(t *testing.T) {
	router, _ := newTestRouter(t)

	// Missing quantity
	body, _ := json.Marshal(map[string]any{
		"itemId":       "item-er2",
		"locationCode": "TOKYO-A1",
	})

	req := httptest.NewRequest("POST", "/api/v1/inventory/receives", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /inventory/receives missing fields: want 400, got %d", rec.Code)
	}
}

// ============================================================
// Inventory: Adjustments
// ============================================================

func TestAPI_AdjustInventory_201OnSuccess(t *testing.T) {
	router, svc := newTestRouter(t)
	ctx := context.Background()

	// Seed some inventory first
	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 20,
	})

	body, _ := json.Marshal(map[string]any{
		"itemId":        "item-er2",
		"locationCode":  "TOKYO-A1",
		"quantityDelta": -3,
	})

	req := httptest.NewRequest("POST", "/api/v1/inventory/adjustments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// adjust returns 200 on success (via writeJSON)
	if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
		t.Errorf("POST /inventory/adjustments: want 200 or 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestAPI_AdjustInventory_400OnZeroDelta(t *testing.T) {
	router, _ := newTestRouter(t)

	body, _ := json.Marshal(map[string]any{
		"itemId":        "item-er2",
		"locationCode":  "TOKYO-A1",
		"quantityDelta": 0,
	})

	req := httptest.NewRequest("POST", "/api/v1/inventory/adjustments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /inventory/adjustments with delta=0: want 400, got %d", rec.Code)
	}
}

// ============================================================
// Inventory: Move
// ============================================================

func TestAPI_MoveInventory_201OnSuccess(t *testing.T) {
	router, svc := newTestRouter(t)
	ctx := context.Background()

	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 20,
	})

	body, _ := json.Marshal(map[string]any{
		"itemId":           "item-er2",
		"fromLocationCode": "TOKYO-A1",
		"toLocationCode":   "TOKYO-B2",
		"quantity":         5,
	})

	req := httptest.NewRequest("POST", "/api/v1/inventory/movements", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("POST /inventory/movements: want 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestAPI_MoveInventory_400OnInsufficientStock(t *testing.T) {
	router, svc := newTestRouter(t)
	ctx := context.Background()

	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 2,
	})

	body, _ := json.Marshal(map[string]any{
		"itemId":           "item-er2",
		"fromLocationCode": "TOKYO-A1",
		"toLocationCode":   "TOKYO-B2",
		"quantity":         100,
	})

	req := httptest.NewRequest("POST", "/api/v1/inventory/movements", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /inventory/movements insufficient: want 400, got %d", rec.Code)
	}
}

// ============================================================
// Reservations
// ============================================================

func TestAPI_CreateReservation_201OnSuccess(t *testing.T) {
	router, _ := newTestRouter(t)

	body, _ := json.Marshal(map[string]any{
		"itemId":        "item-er2",
		"deviceScopeId": "ds-er2-powerboard",
		"quantity":      3,
	})

	req := httptest.NewRequest("POST", "/api/v1/operator/reservations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("POST /reservations: want 201, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestAPI_CreateReservation_400OnMissingFields(t *testing.T) {
	router, _ := newTestRouter(t)

	body, _ := json.Marshal(map[string]any{
		"itemId": "item-er2",
		// missing deviceScopeId and quantity
	})

	req := httptest.NewRequest("POST", "/api/v1/operator/reservations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST /reservations missing fields: want 400, got %d", rec.Code)
	}
}

// ============================================================
// GET endpoints: list/overview
// ============================================================

func TestAPI_InventoryOverview_200(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/inventory/overview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /inventory/overview: want 200, got %d", rec.Code)
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("response missing 'data' envelope")
	}
}

func TestAPI_ReservationList_200(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/operator/reservations", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /reservations: want 200, got %d", rec.Code)
	}
}

func TestAPI_ShortageList_200(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/operator/shortages", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /shortages: want 200, got %d", rec.Code)
	}
}

func TestAPI_MasterSummary_200(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/admin/master-data", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /admin/master-data: want 200, got %d", rec.Code)
	}
}

func TestAPI_InventoryEvents_200(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/inventory/events", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /inventory/events: want 200, got %d", rec.Code)
	}
}

func TestAPI_InventoryLocations_200(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("GET", "/api/v1/inventory/locations", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /inventory/locations: want 200, got %d", rec.Code)
	}
}

// ============================================================
// Undo endpoint
// ============================================================

func TestAPI_UndoInventoryEvent_200OnSuccess(t *testing.T) {
	router, svc := newTestRouter(t)
	ctx := context.Background()

	entry, _ := svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 10,
	})

	body, _ := json.Marshal(map[string]any{
		"actorId": "test-user",
		"reason":  "test undo",
	})

	req := httptest.NewRequest("POST", "/api/v1/inventory/events/"+entry.ID+"/undo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("POST /inventory/events/{id}/undo: want 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestAPI_UndoInventoryEvent_400OnNonexistentEvent(t *testing.T) {
	router, _ := newTestRouter(t)

	req := httptest.NewRequest("POST", "/api/v1/inventory/events/nonexistent-id/undo", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("POST undo nonexistent: want 400, got %d", rec.Code)
	}
}

// ============================================================
// Response envelope format
// ============================================================

func TestAPI_ResponseEnvelope_HasDataField(t *testing.T) {
	router, svc := newTestRouter(t)
	ctx := context.Background()

	svc.ReceiveInventory(ctx, inventory.InventoryReceiveInput{
		ItemID: "item-er2", LocationCode: "TOKYO-A1", Quantity: 5,
	})

	endpoints := []string{
		"/api/v1/inventory/overview",
		"/api/v1/operator/reservations",
		"/api/v1/operator/shortages",
		"/api/v1/inventory/events",
	}

	for _, path := range endpoints {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status: want 200, got %d", rec.Code)
			}

			var resp map[string]json.RawMessage
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("invalid JSON response: %v", err)
			}
			if _, ok := resp["data"]; !ok {
				t.Error("response missing 'data' envelope field")
			}
		})
	}
}
